<?php

namespace App\Services\Bridge;

use App\Services\Geocoding\GoogleGeocodingService;
use Carbon\CarbonImmutable;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

/**
 * Bridge-backed comparable sales analysis (MVP: modes A–E, radius or postal scope, sold + optional active competition).
 *
 * Revenue impact: gated result caps for non-idx:full traffic mirror search teaser economics while still exposing a CMA-lite hook.
 */
final readonly class BridgeCompsService
{
    public function __construct(
        private BridgeSearchClient $searchClient,
        private MlsDatasetResolver $resolver,
        private BridgeProxyAuditLogger $audit,
        private BpoMarketExtractor $bpoExtractor,
        private BpoAdjustmentEngine $bpoEngine,
        private GoogleGeocodingService $geocodingService,
    ) {}

    /**
     * @param  array<string, mixed>  $validated
     * @return array<string, mixed>
     */
    public function run(Request $request, array $validated): array
    {
        $started = microtime(true);
        $dataset = $this->resolver->resolveDataset($request);
        $fullAccess = true;
        $mode = (string) ($validated['mode'] ?? 'A');

        $subject = $this->resolveSubject($request, $dataset, $validated);
        if ($subject === null) {
            $processingMs = (int) round((microtime(true) - $started) * 1000);

            return [
                'success' => false,
                'error' => 'Subject listing not found',
                'metadata' => $this->buildMetadata($validated, $fullAccess, 0, 0, 0, $processingMs, 'median'),
            ];
        }

        if ($mode === 'rent_hold_cashflow') {
            return $this->handleRentHoldCashflow($request, $validated, $subject, $dataset, $fullAccess, $started);
        }

        if ($mode === 'flip_vs_hold') {
            return $this->handleFlipVsHold($request, $validated, $subject, $dataset, $fullAccess, $started);
        }

        if ($mode === 'appraiser_simulation') {
            return $this->handleAppraiserSimulation($request, $validated, $subject, $dataset, $fullAccess, $started);
        }

        if ($mode === 'bpo') {
            return $this->handleBpo($request, $validated, $subject, $dataset, $fullAccess, $started);
        }

        if ($mode === 'home_value') {
            return $this->handleHomeValue($request, $validated, $subject, $dataset, $fullAccess, $started);
        }

        $filters = $validated['filters'] ?? [];
        $maxSoldRequested = (int) ($filters['max_sold_comps'] ?? 12);
        $maxSold = min(25, max(1, $maxSoldRequested));
        $soldMonthsBack = (int) ($filters['sold_months_back'] ?? 6);
        $includeCompetition = (bool) ($filters['include_active_pending'] ?? true);
        $maxCompetition = min(50, (int) ($filters['max_competition_comps'] ?? 20));
        $includeOverpriced = (bool) ($filters['include_overpriced_signals'] ?? true);
        $aggMethod = ($validated['aggregation_method'] ?? 'median') === 'average' ? 'average' : 'median';

        $normDistMiles = $this->normalizationDistanceMiles($validated);
        $soldSince = CarbonImmutable::now()->subMonths($soldMonthsBack)->format('Y-m-d');
        $subject['_sold_since'] = $soldSince;
        $subject['_sold_months_back'] = $soldMonthsBack;
        $subject['_dataset'] = $dataset;

        $soldFilter = $this->buildSoldFilter($subject, $validated, $filters, $soldSince);
        $soldOrderBy = $this->buildOrderByDistance($subject, $validated);
        $soldTop = min(200, max($maxSold * 8, 40));

        $soldResult = $this->searchClient->search(
            $dataset,
            $soldFilter,
            $soldOrderBy,
            $soldTop,
            0,
            $this->searchClient->compsPropertySelectList($dataset),
            '',
        );

        $soldRows = $soldResult['value'];
        $soldRanked = $this->rankAndSlice($soldRows, $subject, $normDistMiles, $maxSold);

        $medianPpsf = $this->centralPpsfFromSold($soldRanked, $aggMethod);
        $keywords = is_array($validated['keywords'] ?? null) ? $validated['keywords'] : [];

        $soldComps = [];
        foreach ($soldRanked as $row) {
            $soldComps[] = $this->mapSoldComp($row, $subject, $medianPpsf, $filters, $mode, $keywords, $normDistMiles);
        }

        $competitionComps = [];
        $compTotal = 0;
        if ($includeCompetition) {
            $compFilter = $this->buildCompetitionFilter($subject, $validated, $filters);
            $compResult = $this->searchClient->search(
                $dataset,
                $compFilter,
                $soldOrderBy,
                $maxCompetition,
                0,
                $this->searchClient->compsPropertySelectList($dataset),
                '',
            );
            $competitionComps = array_map(
                fn (array $row) => $this->mapCompetitionComp($row, $subject, $normDistMiles),
                $compResult['value'],
            );
            $compTotal = count($compResult['value']);
        }

        $overpricedSignals = [];
        if ($includeOverpriced && $fullAccess) {
            $overpricedSignals = $this->computeOverpricedSignals($soldComps, $competitionComps, $aggMethod);
        }

        $marketConditions = $this->computeMarketConditions($soldRanked, count($soldRows), $compTotal, $soldMonthsBack, $aggMethod);

        $subjectResp = $this->mapSubjectResponse($subject);

        $processingMs = (int) round((microtime(true) - $started) * 1000);

        $this->audit->log(
            $request,
            'comps.run',
            count($soldComps),
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return [
            'success' => true,
            'subject' => $subjectResp,
            'sold_comps' => $soldComps,
            'competition_comps' => $competitionComps,
            'failed_listings' => [],
            'overpriced_signals' => $overpricedSignals,
            'market_conditions' => $marketConditions,
            'metadata' => $this->buildMetadata($validated, $fullAccess, count($soldRows), $compTotal, 0, $processingMs, $aggMethod),
            'warnings' => $this->warningsForCaps($fullAccess, $maxSoldRequested, $maxSold),
        ];
    }

    /**
     * @param  array<string, mixed>  $validated
     * @return array<string, mixed>|null
     */
    private function resolveSubject(Request $request, string $dataset, array $validated): ?array
    {
        $subjectIn = $validated['subject'];
        $mode = (string) ($validated['mode'] ?? 'A');

        // home_value mode builds its subject from owner-provided fields or listing_id
        if ($mode === 'home_value') {
            return $this->subjectFromHomeValueFields($request, $dataset, $subjectIn);
        }

        if (($subjectIn['type'] ?? '') === 'off_market') {
            return $this->subjectOffMarket($subjectIn);
        }

        $listingId = (string) ($subjectIn['listing_id'] ?? '');
        if ($listingId === '') {
            return null;
        }

        $listingKey = $this->normalizeListingKey($dataset, $listingId);
        $record = $this->searchClient->getPropertyForComps($request, $dataset, $listingKey);

        return $record !== null ? $this->subjectFromPropertyRecord($record, $dataset) : null;
    }

    /**
     * @param  array<string, mixed>  $subjectIn
     * @return array<string, mixed>
     */
    private function subjectOffMarket(array $subjectIn): array
    {
        $lat = (float) $subjectIn['lat'];
        $lng = (float) $subjectIn['lng'];
        $lotAcres = null;
        if (isset($subjectIn['lot_size_sqft']) && is_numeric($subjectIn['lot_size_sqft'])) {
            $lotAcres = (float) $subjectIn['lot_size_sqft'] / 43560.0;
        }

        $garage = isset($subjectIn['garage_spaces']) ? (int) $subjectIn['garage_spaces'] : null;
        $parkingTotal = $this->offMarketParkingStallsTotal($subjectIn, $garage);

        return [
            'listing_key' => '',
            'lat' => $lat,
            'lng' => $lng,
            'bedrooms' => isset($subjectIn['bedrooms']) ? (int) $subjectIn['bedrooms'] : null,
            'bathrooms' => isset($subjectIn['bathrooms']) && is_numeric($subjectIn['bathrooms']) ? (float) $subjectIn['bathrooms'] : null,
            'living_area' => isset($subjectIn['living_area_sqft']) ? (int) $subjectIn['living_area_sqft'] : null,
            'lot_acres' => $lotAcres,
            'year_built' => isset($subjectIn['year_built']) ? (int) $subjectIn['year_built'] : null,
            'pool' => isset($subjectIn['pool']) ? (bool) $subjectIn['pool'] : null,
            'hoa' => isset($subjectIn['hoa']) ? (bool) $subjectIn['hoa'] : null,
            'senior_community' => isset($subjectIn['senior_community']) ? (bool) $subjectIn['senior_community'] : null,
            'waterfront' => isset($subjectIn['waterfront']) ? (bool) $subjectIn['waterfront'] : null,
            'garage_spaces' => $garage,
            'parking_stalls_total' => $parkingTotal,
            'subdivision_name' => isset($subjectIn['subdivision_name']) && is_string($subjectIn['subdivision_name']) ? trim($subjectIn['subdivision_name']) : null,
            'mls_area_major' => isset($subjectIn['mls_area_major']) && is_string($subjectIn['mls_area_major']) ? trim($subjectIn['mls_area_major']) : null,
            'view_yn' => isset($subjectIn['view_yn']) ? (bool) $subjectIn['view_yn'] : null,
            'monthly_fees' => isset($subjectIn['monthly_fees']) && is_numeric($subjectIn['monthly_fees']) ? (float) $subjectIn['monthly_fees'] : null,
            'list_price' => isset($subjectIn['asking_price']) ? (float) $subjectIn['asking_price'] : null,
            'property_type' => null,
            'property_sub_type' => null,
            'flood_zone_codes' => $this->floodCodesFromCsvString($subjectIn['flood_zone_code'] ?? null),
            'address' => '',
        ];
    }

    /**
     * @param  array<string, mixed>  $record
     * @return array<string, mixed>
     */
    private function subjectFromPropertyRecord(array $record, string $dataset): array
    {
        $coords = $this->parseCoordinates($record);
        $u = strtoupper($dataset);
        $floodField = "{$u}_FloodZoneCode";
        $feesField = "{$u}_TotalMonthlyFees";

        return [
            'listing_key' => (string) ($record['ListingKey'] ?? ''),
            'lat' => $coords['lat'] ?? 0.0,
            'lng' => $coords['lng'] ?? 0.0,
            'bedrooms' => $this->intOrNull($record['BedroomsTotal'] ?? null),
            'bathrooms' => $this->floatOrNull($record['BathroomsTotalDecimal'] ?? null),
            'living_area' => $this->intOrNull($record['LivingArea'] ?? null),
            'lot_acres' => $this->floatOrNull($record['LotSizeAcres'] ?? null),
            'year_built' => $this->intOrNull($record['YearBuilt'] ?? null),
            'pool' => $this->boolOrNull($record['PoolPrivateYN'] ?? null),
            'hoa' => $this->boolOrNull($record['AssociationYN'] ?? null),
            'senior_community' => $this->boolOrNull($record['SeniorCommunityYN'] ?? null),
            'waterfront' => $this->boolOrNull($record['WaterfrontYN'] ?? null),
            'garage_spaces' => $this->garageSpacesIntFromRow($record),
            'parking_stalls_total' => $this->parkingStallTotalFromRow($record),
            'subdivision_name' => $this->stringOrNull($record['SubdivisionName'] ?? null),
            'mls_area_major' => $this->stringOrNull($record['MLSAreaMajor'] ?? null),
            'view_yn' => $this->boolOrNull($record['ViewYN'] ?? null),
            'monthly_fees' => $this->floatOrNull($record[$feesField] ?? null),
            'list_price' => $this->floatOrNull($record['ListPrice'] ?? null),
            'property_type' => $this->stringOrNull($record['PropertyType'] ?? null),
            'property_sub_type' => $this->stringOrNull($record['PropertySubType'] ?? null),
            'flood_zone_codes' => $this->floodCodesFromCsvString($record[$floodField] ?? null),
            'address' => $this->formatAddress($record),
        ];
    }

    /**
     * @return list<string>
     */
    private function floodCodesFromCsvString(mixed $raw): array
    {
        if (! is_string($raw) || trim($raw) === '') {
            return [];
        }

        return array_values(array_filter(array_map('trim', explode(',', $raw))));
    }

    /**
     * @param  array<string, mixed>  $subjectIn
     */
    private function offMarketParkingStallsTotal(array $subjectIn, ?int $garage): ?int
    {
        if (isset($subjectIn['parking_stalls_total']) && is_numeric($subjectIn['parking_stalls_total'])) {
            return min(12, max(0, (int) $subjectIn['parking_stalls_total']));
        }

        $extra = 0;
        foreach (['carport_spaces', 'covered_spaces', 'open_parking_spaces'] as $k) {
            if (isset($subjectIn[$k]) && is_numeric($subjectIn[$k])) {
                $extra += (int) $subjectIn[$k];
            }
        }

        if ($garage === null && $extra === 0) {
            return null;
        }

        return min(12, ($garage ?? 0) + $extra);
    }

    /**
     * @param  array<string, mixed>  $row
     */
    private function garageSpacesIntFromRow(array $row): ?int
    {
        if (! isset($row['GarageSpaces']) || ! is_numeric($row['GarageSpaces'])) {
            return null;
        }

        $n = (int) round((float) $row['GarageSpaces']);

        return $n >= 0 ? $n : null;
    }

    /**
     * @param  array<string, mixed>  $row
     */
    private function parkingStallTotalFromRow(array $row): ?int
    {
        $sum = 0.0;
        $any = false;
        foreach (['GarageSpaces', 'CarportSpaces', 'CoveredSpaces', 'OpenParkingSpaces'] as $k) {
            if (isset($row[$k]) && is_numeric($row[$k])) {
                $sum += (float) $row[$k];
                $any = true;
            }
        }

        if (! $any) {
            return null;
        }

        return min(12, (int) round($sum));
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $validated
     * @param  array<string, mixed>  $filters
     */
    private function buildSoldFilter(array $subject, array $validated, array $filters, string $soldSince): string
    {
        $clauses = [
            "tolower(StandardStatus) eq 'closed'",
            "CloseDate ge {$soldSince}",
        ];

        if ($subject['listing_key'] !== '') {
            $escaped = str_replace("'", "''", $subject['listing_key']);
            $clauses[] = "ListingKey ne '{$escaped}'";
        }

        $clauses = array_merge($clauses, $this->scopeClauses($subject, $validated));
        $clauses = array_merge($clauses, $this->toleranceClauses($subject, $filters));

        return implode(' and ', $clauses);
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $validated
     * @param  array<string, mixed>  $filters
     */
    private function buildCompetitionFilter(array $subject, array $validated, array $filters): string
    {
        $clauses = [
            "(tolower(StandardStatus) eq 'active' or tolower(StandardStatus) eq 'pending')",
        ];
        if ($subject['listing_key'] !== '') {
            $escaped = str_replace("'", "''", $subject['listing_key']);
            $clauses[] = "ListingKey ne '{$escaped}'";
        }
        $clauses = array_merge($clauses, $this->scopeClauses($subject, $validated));
        $clauses = array_merge($clauses, $this->toleranceClauses($subject, $filters));

        return implode(' and ', $clauses);
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $validated
     * @return list<string>
     */
    private function scopeClauses(array $subject, array $validated): array
    {
        $scope = $validated['scope'] ?? [];
        $type = (string) ($scope['type'] ?? 'radius');
        if ($type === 'zip') {
            $codes = $scope['postal_codes'] ?? [];
            if (! is_array($codes) || $codes === []) {
                return [];
            }
            $parts = [];
            foreach ($codes as $code) {
                if (! is_string($code) || trim($code) === '') {
                    continue;
                }
                $c = str_replace("'", "''", trim($code));
                $parts[] = "PostalCode eq '{$c}'";
            }

            return $parts === [] ? [] : ['('.implode(' or ', $parts).')'];
        }

        $radius = (float) ($scope['radius_miles'] ?? 5.0);
        $lat = $scope['center_lat'] ?? null;
        $lng = $scope['center_lng'] ?? null;
        if ($lat === null || $lng === null) {
            $lat = $subject['lat'];
            $lng = $subject['lng'];
        }

        return ["geo.distance(Coordinates, POINT({$lng} {$lat})) lt {$radius}"];
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $filters
     * @return list<string>
     */
    private function toleranceClauses(array $subject, array $filters): array
    {
        $clauses = [];
        $laPct = (int) ($filters['living_area_pct'] ?? 10);
        if ($subject['living_area'] !== null && $laPct > 0) {
            $la = (int) $subject['living_area'];
            $delta = (int) max(1, round($la * ($laPct / 100.0)));
            $min = max(0, $la - $delta);
            $max = $la + $delta;
            $clauses[] = "LivingArea ge {$min} and LivingArea le {$max}";
        }

        $bedTol = (int) ($filters['beds_tolerance'] ?? 1);
        if ($subject['bedrooms'] !== null && $bedTol >= 0) {
            $b = (int) $subject['bedrooms'];
            $clauses[] = 'BedroomsTotal ge '.($b - $bedTol).' and BedroomsTotal le '.($b + $bedTol);
        }

        $bathTol = (int) ($filters['baths_tolerance'] ?? 1);
        if ($subject['bathrooms'] !== null && $bathTol >= 0) {
            $b = (float) $subject['bathrooms'];
            $clauses[] = 'BathroomsTotalDecimal ge '.($b - $bathTol).' and BathroomsTotalDecimal le '.($b + $bathTol);
        }

        $yearTol = (int) ($filters['year_built_tolerance'] ?? 15);
        if ($subject['year_built'] !== null && $yearTol >= 0) {
            $y = (int) $subject['year_built'];
            $clauses[] = 'YearBuilt ge '.($y - $yearTol).' and YearBuilt le '.($y + $yearTol);
        }

        $lotPct = (int) ($filters['lot_size_pct'] ?? 50);
        if ($subject['lot_acres'] !== null && $lotPct > 0) {
            $ac = (float) $subject['lot_acres'];
            $delta = max(0.0001, $ac * ($lotPct / 100.0));
            $clauses[] = 'LotSizeAcres ge '.($ac - $delta).' and LotSizeAcres le '.($ac + $delta);
        }

        if (($filters['match_pool'] ?? true) && $subject['pool'] !== null) {
            $v = $subject['pool'] ? 'true' : 'false';
            $clauses[] = "PoolPrivateYN eq {$v}";
        }
        if (($filters['match_hoa'] ?? true) && $subject['hoa'] !== null) {
            $v = $subject['hoa'] ? 'true' : 'false';
            $clauses[] = "AssociationYN eq {$v}";
        }
        if (($filters['match_senior_community'] ?? true) && $subject['senior_community'] !== null) {
            $v = $subject['senior_community'] ? 'true' : 'false';
            $clauses[] = "SeniorCommunityYN eq {$v}";
        }
        if (($filters['match_property_sub_type'] ?? true) && $subject['property_sub_type'] !== null) {
            $escaped = str_replace("'", "''", $subject['property_sub_type']);
            $clauses[] = "PropertySubType eq '{$escaped}'";
        }
        if (($filters['match_waterfront'] ?? true) && $subject['waterfront'] !== null) {
            $v = $subject['waterfront'] ? 'true' : 'false';
            $clauses[] = "WaterfrontYN eq {$v}";
        }

        if (($filters['match_view'] ?? false) && array_key_exists('view_yn', $subject) && $subject['view_yn'] !== null) {
            $v = $subject['view_yn'] ? 'true' : 'false';
            $clauses[] = "ViewYN eq {$v}";
        }

        if (($filters['match_subdivision'] ?? false) && ! empty($subject['subdivision_name'])) {
            $escaped = str_replace("'", "''", (string) $subject['subdivision_name']);
            $clauses[] = "SubdivisionName eq '{$escaped}'";
        }

        if (($filters['match_mls_area_major'] ?? false) && ! empty($subject['mls_area_major'])) {
            $escaped = str_replace("'", "''", (string) $subject['mls_area_major']);
            $clauses[] = "MLSAreaMajor eq '{$escaped}'";
        }

        if (($filters['match_garage_spaces'] ?? false) && $subject['garage_spaces'] !== null) {
            $n = (int) $subject['garage_spaces'];
            $clauses[] = "GarageSpaces eq {$n}";
        }

        if (isset($filters['min_garage_spaces']) && is_numeric($filters['min_garage_spaces'])) {
            $n = (int) $filters['min_garage_spaces'];
            $clauses[] = "GarageSpaces ge {$n}";
        }

        if (isset($filters['max_garage_spaces']) && is_numeric($filters['max_garage_spaces'])) {
            $n = (int) $filters['max_garage_spaces'];
            $clauses[] = "GarageSpaces le {$n}";
        }

        return $clauses;
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $validated
     */
    private function buildOrderByDistance(array $subject, array $validated): string
    {
        $scope = $validated['scope'] ?? [];
        $lat = $scope['center_lat'] ?? $subject['lat'];
        $lng = $scope['center_lng'] ?? $subject['lng'];

        return "geo.distance(Coordinates, POINT({$lng} {$lat})) asc";
    }

    private function normalizationDistanceMiles(array $validated): float
    {
        $scope = $validated['scope'] ?? [];
        if (($scope['type'] ?? '') === 'radius' && isset($scope['radius_miles'])) {
            return max(0.1, (float) $scope['radius_miles']);
        }

        return 10.0;
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     * @param  array<string, mixed>  $subject
     * @return list<array<string, mixed>>
     */
    private function rankAndSlice(array $rows, array $subject, float $normDistMiles, int $maxSold): array
    {
        $scored = [];
        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }
            $dist = $this->distanceMiles($subject, $row);
            $score = $this->similarityScore($subject, $row, $dist, $normDistMiles);
            $scored[] = ['row' => $row, 'score' => $score, 'dist' => $dist];
        }

        usort($scored, static fn (array $a, array $b): int => $b['score'] <=> $a['score']);

        $out = [];
        foreach (array_slice($scored, 0, $maxSold) as $item) {
            $out[] = $item['row'];
        }

        return $out;
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $row
     */
    private function distanceMiles(array $subject, array $row): float
    {
        $c = $this->parseCoordinates($row);
        if ($c === null || ($subject['lat'] === 0.0 && $subject['lng'] === 0.0)) {
            return 999.0;
        }

        return $this->haversineMiles($subject['lat'], $subject['lng'], $c['lat'], $c['lng']);
    }

    private function haversineMiles(float $lat1, float $lon1, float $lat2, float $lon2): float
    {
        $r = 3959.0;
        $dLat = deg2rad($lat2 - $lat1);
        $dLon = deg2rad($lon2 - $lon1);
        $a = sin($dLat / 2) ** 2 + cos(deg2rad($lat1)) * cos(deg2rad($lat2)) * sin($dLon / 2) ** 2;

        return 2 * $r * asin(min(1.0, sqrt($a)));
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $row
     */
    private function similarityScore(array $subject, array $row, float $distMiles, float $normDistMiles): float
    {
        $distPart = 1.0 - min(1.0, $distMiles / max(0.01, $normDistMiles));
        $glaPart = 1.0;
        if ($subject['living_area'] !== null && isset($row['LivingArea']) && is_numeric($row['LivingArea'])) {
            $comp = (int) $row['LivingArea'];
            $diff = abs((int) $subject['living_area'] - $comp);
            $glaPart = 1.0 - min(1.0, $diff / max(1.0, (float) $subject['living_area'] * 0.25));
        }
        $bedPart = 1.0;
        if ($subject['bedrooms'] !== null && isset($row['BedroomsTotal']) && is_numeric($row['BedroomsTotal'])) {
            $bedPart = 1.0 - min(1.0, abs((int) $subject['bedrooms'] - (int) $row['BedroomsTotal']) / 3.0);
        }
        $bathPart = 1.0;
        if ($subject['bathrooms'] !== null && isset($row['BathroomsTotalDecimal']) && is_numeric($row['BathroomsTotalDecimal'])) {
            $bathPart = 1.0 - min(1.0, abs((float) $subject['bathrooms'] - (float) $row['BathroomsTotalDecimal']) / 3.0);
        }
        $yearPart = 1.0;
        if ($subject['year_built'] !== null && isset($row['YearBuilt']) && is_numeric($row['YearBuilt'])) {
            $yearPart = 1.0 - min(1.0, abs((int) $subject['year_built'] - (int) $row['YearBuilt']) / 30.0);
        }

        $parkingPart = 1.0;
        $subPark = $subject['parking_stalls_total'] ?? null;
        $rowPark = $this->parkingStallTotalFromRow($row);
        if ($subPark !== null && $rowPark !== null) {
            $parkingPart = 1.0 - min(1.0, abs($subPark - $rowPark) / 4.0);
        }

        $viewPart = 1.0;
        if (array_key_exists('view_yn', $subject) && $subject['view_yn'] !== null && isset($row['ViewYN']) && is_bool($row['ViewYN'])) {
            $viewPart = $subject['view_yn'] === $row['ViewYN'] ? 1.0 : 0.4;
        }

        $recencyPart = 0.85;
        $soldSince = $subject['_sold_since'] ?? null;
        $monthsBack = (int) ($subject['_sold_months_back'] ?? 6);
        if (is_string($soldSince) && isset($row['CloseDate']) && is_string($row['CloseDate']) && $row['CloseDate'] !== '') {
            try {
                $close = new \DateTimeImmutable($row['CloseDate']);
                $now = new \DateTimeImmutable('today');
                $daysAgo = max(0, $now->diff($close)->days);
                $windowDays = max(30.0, (float) $monthsBack * 30.0);
                $recencyPart = 1.0 - min(1.0, $daysAgo / $windowDays);
            } catch (\Throwable) {
                $recencyPart = 0.85;
            }
        }

        $hoaFeePart = 1.0;
        $sf = $subject['monthly_fees'] ?? null;
        $dataset = (string) ($subject['_dataset'] ?? '');
        $cf = null;
        if ($dataset !== '') {
            $cf = $this->floatOrNull($row[strtoupper($dataset).'_TotalMonthlyFees'] ?? null);
        }
        if ($sf !== null && $cf !== null) {
            $mx = max(1.0, max($sf, $cf));
            $hoaFeePart = 1.0 - min(1.0, abs($sf - $cf) / $mx);
        }

        return max(0.0, min(1.0,
            0.30 * $distPart
            + 0.22 * $glaPart
            + 0.12 * $bedPart
            + 0.12 * $bathPart
            + 0.08 * $yearPart
            + 0.06 * $parkingPart
            + 0.04 * $recencyPart
            + 0.03 * $viewPart
            + 0.03 * $hoaFeePart
        ));
    }

    /**
     * @param  list<array<string, mixed>>  $soldRanked
     */
    private function centralPpsfFromSold(array $soldRanked, string $aggMethod): float
    {
        $ppsfs = [];
        foreach ($soldRanked as $row) {
            $cp = $this->floatOrNull($row['ClosePrice'] ?? null);
            $la = $this->intOrNull($row['LivingArea'] ?? null);
            if ($cp !== null && $la !== null && $la > 0) {
                $ppsfs[] = $cp / $la;
            }
        }

        return $this->aggregateNumeric($ppsfs, $aggMethod);
    }

    /**
     * @param  list<float>  $values
     */
    private function aggregateNumeric(array $values, string $aggMethod): float
    {
        if ($values === []) {
            return 0.0;
        }
        sort($values);
        if ($aggMethod === 'average') {
            return array_sum($values) / count($values);
        }
        $n = count($values);
        $mid = intdiv($n, 2);
        if ($n % 2 === 1) {
            return $values[$mid];
        }

        return ($values[$mid - 1] + $values[$mid]) / 2.0;
    }

    /**
     * @param  array<string, mixed>  $row
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $filters
     * @param  array<string, mixed>  $keywords
     * @return array<string, mixed>
     */
    private function mapSoldComp(array $row, array $subject, float $medianPpsf, array $filters, string $mode, array $keywords, float $normDistMiles): array
    {
        $listingKey = (string) ($row['ListingKey'] ?? '');
        $dist = $this->distanceMiles($subject, $row);
        $sim = $this->similarityScore($subject, $row, $dist, $normDistMiles) * 100.0;
        $remarks = is_string($row['PublicRemarks'] ?? null) ? $row['PublicRemarks'] : '';
        $kw = $this->keywordScores($remarks, $keywords, $mode);
        $close = $this->floatOrNull($row['ClosePrice'] ?? null);
        $la = $this->intOrNull($row['LivingArea'] ?? null);
        $ppsf = ($close !== null && $la !== null && $la > 0) ? $close / $la : null;
        $closeDate = null;
        if (isset($row['CloseDate']) && is_string($row['CloseDate']) && $row['CloseDate'] !== '') {
            try {
                $closeDate = (new \DateTimeImmutable($row['CloseDate']))->format('Y-m-d');
            } catch (\Throwable) {
                $closeDate = null;
            }
        }
        $c = $this->parseCoordinates($row);

        $compRowForAdj = $this->rowToAdjustmentShape($row);
        $dataset = (string) ($subject['_dataset'] ?? '');
        $floodField = $dataset !== '' ? strtoupper($dataset).'_FloodZoneCode' : '';
        $floodCodes = $floodField !== '' ? $this->floodCodesFromCsvString($row[$floodField] ?? null) : [];

        return [
            'listing_id' => Str::afterLast($listingKey, ':'),
            'address' => $this->formatAddress($row),
            'lat' => $c['lat'] ?? 0.0,
            'lng' => $c['lng'] ?? 0.0,
            'sold_price' => $close,
            'sold_date' => $closeDate,
            'bedrooms' => $this->intOrNull($row['BedroomsTotal'] ?? null),
            'bathrooms' => $this->floatOrNull($row['BathroomsTotalDecimal'] ?? null),
            'living_area_sqft' => $la,
            'ppsf' => $ppsf,
            'distance_miles' => round($dist, 3),
            'dom' => $this->intOrNull($row['DaysOnMarket'] ?? null),
            'cumulative_days_on_market' => $this->intOrNull($row['CumulativeDaysOnMarket'] ?? null),
            'similarity_score' => round($sim, 2),
            'disrepair_score' => $kw['disrepair'],
            'good_condition_score' => $kw['good_condition'],
            'keyword_matches' => $kw['matches'],
            'year_built' => $this->intOrNull($row['YearBuilt'] ?? null),
            'lot_size_acres' => $this->floatOrNull($row['LotSizeAcres'] ?? null),
            'waterfront' => $this->boolOrNull($row['WaterfrontYN'] ?? null),
            'view_yn' => $this->boolOrNull($row['ViewYN'] ?? null),
            'garage_spaces' => $this->garageSpacesIntFromRow($row),
            'parking_stalls_total' => $this->parkingStallTotalFromRow($row),
            'stories' => $this->intOrNull($row['StoriesTotal'] ?? null),
            'bathrooms_full' => null,
            'bathrooms_half' => null,
            'pool' => $this->boolOrNull($row['PoolPrivateYN'] ?? null),
            'property_type' => $this->stringOrNull($row['PropertyType'] ?? null),
            'property_sub_type' => $this->stringOrNull($row['PropertySubType'] ?? null),
            'subdivision_name' => $this->stringOrNull($row['SubdivisionName'] ?? null),
            'mls_area_major' => $this->stringOrNull($row['MLSAreaMajor'] ?? null),
            'monthly_fees' => $dataset !== '' ? $this->floatOrNull($row[strtoupper($dataset).'_TotalMonthlyFees'] ?? null) : null,
            'adjustments' => $close !== null ? $this->adjustmentGrid($subject, $compRowForAdj, $medianPpsf, $filters) : null,
            'flood_zone_codes' => $floodCodes,
            'flood_zone' => $floodCodes[0] ?? null,
        ];
    }

    /**
     * @param  array<string, mixed>  $row
     * @param  array<string, mixed>  $subject
     * @return array<string, mixed>
     */
    private function mapCompetitionComp(array $row, array $subject, float $normDistMiles): array
    {
        $listingKey = (string) ($row['ListingKey'] ?? '');
        $dist = $this->distanceMiles($subject, $row);
        $sim = $this->similarityScore($subject, $row, $dist, $normDistMiles) * 100.0;
        $lp = $this->floatOrNull($row['ListPrice'] ?? null);
        $la = $this->intOrNull($row['LivingArea'] ?? null);
        $ppsf = ($lp !== null && $la !== null && $la > 0) ? $lp / $la : null;
        $c = $this->parseCoordinates($row);

        $dataset = (string) ($subject['_dataset'] ?? '');
        $floodField = $dataset !== '' ? strtoupper($dataset).'_FloodZoneCode' : '';
        $floodCodes = $floodField !== '' ? $this->floodCodesFromCsvString($row[$floodField] ?? null) : [];

        return [
            'listing_id' => Str::afterLast($listingKey, ':'),
            'address' => $this->formatAddress($row),
            'lat' => $c['lat'] ?? 0.0,
            'lng' => $c['lng'] ?? 0.0,
            'status' => (string) ($row['StandardStatus'] ?? ''),
            'list_price' => $lp,
            'ppsf' => $ppsf,
            'distance_miles' => round($dist, 3),
            'dom' => $this->intOrNull($row['DaysOnMarket'] ?? null),
            'cumulative_days_on_market' => $this->intOrNull($row['CumulativeDaysOnMarket'] ?? null),
            'bedrooms' => $this->intOrNull($row['BedroomsTotal'] ?? null),
            'bathrooms' => $this->floatOrNull($row['BathroomsTotalDecimal'] ?? null),
            'living_area_sqft' => $la,
            'similarity_score' => round($sim, 2),
            'year_built' => $this->intOrNull($row['YearBuilt'] ?? null),
            'lot_size_acres' => $this->floatOrNull($row['LotSizeAcres'] ?? null),
            'waterfront' => $this->boolOrNull($row['WaterfrontYN'] ?? null),
            'view_yn' => $this->boolOrNull($row['ViewYN'] ?? null),
            'garage_spaces' => $this->garageSpacesIntFromRow($row),
            'parking_stalls_total' => $this->parkingStallTotalFromRow($row),
            'stories' => $this->intOrNull($row['StoriesTotal'] ?? null),
            'bathrooms_full' => null,
            'bathrooms_half' => null,
            'pool' => $this->boolOrNull($row['PoolPrivateYN'] ?? null),
            'property_type' => $this->stringOrNull($row['PropertyType'] ?? null),
            'property_sub_type' => $this->stringOrNull($row['PropertySubType'] ?? null),
            'subdivision_name' => $this->stringOrNull($row['SubdivisionName'] ?? null),
            'mls_area_major' => $this->stringOrNull($row['MLSAreaMajor'] ?? null),
            'monthly_fees' => $dataset !== '' ? $this->floatOrNull($row[strtoupper($dataset).'_TotalMonthlyFees'] ?? null) : null,
            'flood_zone_codes' => $floodCodes,
            'flood_zone' => $floodCodes[0] ?? null,
        ];
    }

    /**
     * @param  array<string, mixed>  $subject
     * @return array<string, mixed>
     */
    private function mapSubjectResponse(array $subject): array
    {
        return [
            'listing_id' => $subject['listing_key'] !== '' ? Str::afterLast($subject['listing_key'], ':') : null,
            'address' => $subject['address'],
            'lat' => $subject['lat'],
            'lng' => $subject['lng'],
            'bedrooms' => $subject['bedrooms'],
            'bathrooms' => $subject['bathrooms'],
            'living_area_sqft' => $subject['living_area'],
            'lot_size_sqft' => $subject['lot_acres'] !== null ? (int) round($subject['lot_acres'] * 43560) : null,
            'year_built' => $subject['year_built'],
            'list_price' => $subject['list_price'],
            'property_type' => $subject['property_type'],
            'property_sub_type' => $subject['property_sub_type'],
            'waterfront' => $subject['waterfront'],
            'garage_spaces' => $subject['garage_spaces'],
            'parking_stalls_total' => $subject['parking_stalls_total'] ?? null,
            'subdivision_name' => $subject['subdivision_name'] ?? null,
            'mls_area_major' => $subject['mls_area_major'] ?? null,
            'view_yn' => $subject['view_yn'] ?? null,
            'monthly_fees' => $subject['monthly_fees'] ?? null,
            'senior_community' => $subject['senior_community'],
            'flood_zone_codes' => $subject['flood_zone_codes'],
        ];
    }

    /**
     * @param  list<array<string, mixed>>  $soldRanked
     * @return array<string, mixed>|null
     */
    private function computeMarketConditions(array $soldRanked, int $soldTotalCandidates, int $compTotal, int $soldMonthsBack, string $aggMethod): ?array
    {
        $ppsfs = [];
        $doms = [];
        foreach ($soldRanked as $row) {
            $cp = $this->floatOrNull($row['ClosePrice'] ?? null);
            $la = $this->intOrNull($row['LivingArea'] ?? null);
            if ($cp !== null && $la !== null && $la > 0) {
                $ppsfs[] = $cp / $la;
            }
            $dom = $this->intOrNull($row['DaysOnMarket'] ?? null);
            if ($dom !== null) {
                $doms[] = (float) $dom;
            }
        }

        $medianPpsf = $ppsfs === [] ? null : round($this->aggregateNumeric($ppsfs, $aggMethod), 2);
        $medianDom = $doms === [] ? null : (int) round($this->aggregateNumeric($doms, $aggMethod));

        $ratioSum = 0.0;
        $ratioCount = 0;
        foreach ($soldRanked as $row) {
            $cp = $this->floatOrNull($row['ClosePrice'] ?? null);
            $orig = $this->floatOrNull($row['OriginalListPrice'] ?? null);
            if ($cp !== null && $orig !== null && $orig > 0) {
                $ratioSum += $cp / $orig;
                $ratioCount++;
            }
        }
        $avgListToSale = $ratioCount > 0 ? round($ratioSum / $ratioCount, 3) : null;

        $moi = null;
        if ($soldMonthsBack > 0 && $soldTotalCandidates > 0 && $compTotal > 0) {
            $absorption = $soldTotalCandidates / $soldMonthsBack;
            if ($absorption > 0) {
                $moi = round($compTotal / $absorption, 1);
            }
        }

        return [
            'median_sold_ppsf' => $medianPpsf,
            'median_sold_dom' => $medianDom,
            'avg_list_to_sale_ratio' => $avgListToSale,
            'months_of_inventory' => $moi,
        ];
    }

    /**
     * @param  list<array<string, mixed>>  $soldComps
     * @param  list<array<string, mixed>>  $competition
     * @return list<array<string, mixed>>
     */
    private function computeOverpricedSignals(array $soldComps, array $competition, string $aggMethod): array
    {
        $doms = array_values(array_filter(array_map(
            static fn (array $s) => isset($s['dom']) && is_int($s['dom']) ? (float) $s['dom'] : null,
            $soldComps,
        )));
        $medianDom = $doms === [] ? 0.0 : $this->aggregateNumeric($doms, $aggMethod);

        $signals = [];
        if ($medianDom > 0) {
            foreach ($competition as $c) {
                $dom = $c['dom'] ?? null;
                if (! is_int($dom)) {
                    continue;
                }
                if ((float) $dom > $medianDom) {
                    $severity = 'low';
                    if ($dom > $medianDom * 2) {
                        $severity = 'high';
                    } elseif ($dom > $medianDom * 1.5) {
                        $severity = 'moderate';
                    }
                    $signals[] = [
                        'listing_id' => $c['listing_id'] ?? '',
                        'indicator' => 'dom_above_median',
                        'detail' => "DOM {$dom} vs median ".(int) $medianDom,
                        'severity' => $severity,
                    ];
                }
            }
        }

        return $signals;
    }

    /**
     * @param  array<string, mixed>  $row
     * @return array<string, mixed>
     */
    private function rowToAdjustmentShape(array $row): array
    {
        return [
            'ClosePrice' => $this->floatOrNull($row['ClosePrice'] ?? null),
            'LivingArea' => $this->intOrNull($row['LivingArea'] ?? null),
            'PoolPrivateYN' => $this->boolOrNull($row['PoolPrivateYN'] ?? null),
            'GarageSpaces' => $this->garageSpacesIntFromRow($row),
            'WaterfrontYN' => $this->boolOrNull($row['WaterfrontYN'] ?? null),
            'YearBuilt' => $this->intOrNull($row['YearBuilt'] ?? null),
            'LotSizeAcres' => $this->floatOrNull($row['LotSizeAcres'] ?? null),
        ];
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $comp
     * @param  array<string, mixed>  $filters
     * @return array<string, mixed>|null
     */
    private function adjustmentGrid(array $subject, array $comp, float $medianPpsf, array $filters): ?array
    {
        if ($comp['ClosePrice'] === null) {
            return null;
        }

        $lines = [];
        $absSum = 0.0;
        $poolAdj = (float) ($filters['adj_pool_value'] ?? 20_000);
        $wfAdj = (float) ($filters['adj_waterfront_value'] ?? 50_000);
        $yearPer = (float) ($filters['adj_year_built_per_year'] ?? 500);
        $lotPer = (float) ($filters['adj_lot_per_acre'] ?? 25_000);

        if ($subject['living_area'] !== null && $comp['LivingArea'] !== null && $medianPpsf > 0) {
            $diff = (float) ($subject['living_area'] - $comp['LivingArea']);
            $adj = round($diff * $medianPpsf);
            $lines[] = ['feature' => 'gla', 'subject_value' => $subject['living_area'], 'comp_value' => $comp['LivingArea'], 'adjustment' => $adj, 'reasoning' => 'GLA adjustment using median sold PPSF.'];
            $absSum += abs($adj);
        }

        if ($subject['pool'] !== null && $comp['PoolPrivateYN'] !== null && $subject['pool'] !== $comp['PoolPrivateYN']) {
            $adj = $subject['pool'] ? $poolAdj : -$poolAdj;
            $lines[] = ['feature' => 'pool', 'subject_value' => $subject['pool'], 'comp_value' => $comp['PoolPrivateYN'], 'adjustment' => $adj, 'reasoning' => 'Pool presence mismatch.'];
            $absSum += abs($adj);
        }

        if ($subject['waterfront'] !== null && $comp['WaterfrontYN'] !== null && $subject['waterfront'] !== $comp['WaterfrontYN']) {
            $adj = $subject['waterfront'] ? $wfAdj : -$wfAdj;
            $lines[] = ['feature' => 'waterfront', 'subject_value' => $subject['waterfront'], 'comp_value' => $comp['WaterfrontYN'], 'adjustment' => $adj, 'reasoning' => 'Waterfront mismatch.'];
            $absSum += abs($adj);
        }

        if ($subject['year_built'] !== null && $comp['YearBuilt'] !== null) {
            $diff = $subject['year_built'] - $comp['YearBuilt'];
            if ($diff !== 0) {
                $adj = round($diff * $yearPer);
                $lines[] = ['feature' => 'year_built', 'subject_value' => $subject['year_built'], 'comp_value' => $comp['YearBuilt'], 'adjustment' => $adj, 'reasoning' => 'Year built delta.'];
                $absSum += abs($adj);
            }
        }

        if ($subject['lot_acres'] !== null && $comp['LotSizeAcres'] !== null) {
            $diff = $subject['lot_acres'] - $comp['LotSizeAcres'];
            if (abs($diff) > 0.0001) {
                $adj = round($diff * $lotPer);
                $lines[] = ['feature' => 'lot_size', 'subject_value' => $subject['lot_acres'], 'comp_value' => $comp['LotSizeAcres'], 'adjustment' => $adj, 'reasoning' => 'Lot size delta.'];
                $absSum += abs($adj);
            }
        }

        $garagePer = (float) ($filters['adj_garage_per_space'] ?? 10_000);
        if ($subject['garage_spaces'] !== null && $comp['GarageSpaces'] !== null && $garagePer > 0) {
            $diff = (int) $subject['garage_spaces'] - (int) $comp['GarageSpaces'];
            if ($diff !== 0) {
                $adj = (int) round($diff * $garagePer);
                $lines[] = ['feature' => 'garage', 'subject_value' => $subject['garage_spaces'], 'comp_value' => $comp['GarageSpaces'], 'adjustment' => $adj, 'reasoning' => 'Garage stall count delta.'];
                $absSum += abs($adj);
            }
        }

        $net = 0.0;
        foreach ($lines as $l) {
            $net += $l['adjustment'];
        }

        $close = $comp['ClosePrice'];
        $grossPct = $close > 0 ? round($absSum / $close * 100, 1) : 0.0;

        return [
            'lines' => $lines,
            'net_adjustment' => round($net),
            'adjusted_price' => round($close + $net),
            'gross_adjustment_pct' => $grossPct,
            'high_adjustment_warning' => $grossPct > 25,
        ];
    }

    /**
     * @param  array<string, list<array{phrase: string, weight: float, is_regex?: bool}>>  $keywords
     * @return array{disrepair: float, good_condition: float, matches: array<string, list<string>>}
     */
    private function keywordScores(string $remarks, array $keywords, string $mode): array
    {
        if ($mode === 'D' || $mode === 'E') {
            return [
                'disrepair' => 0.5,
                'good_condition' => 0.5,
                'matches' => ['disrepair' => [], 'good_condition' => []],
            ];
        }

        $lower = strtolower($remarks);
        $dis = $this->scoreKeywordCategory($lower, $remarks, $keywords['disrepair'] ?? []);
        $good = $this->scoreKeywordCategory($lower, $remarks, $keywords['good_condition'] ?? []);

        return [
            'disrepair' => $dis['score'],
            'good_condition' => $good['score'],
            'matches' => ['disrepair' => $dis['matched'], 'good_condition' => $good['matched']],
        ];
    }

    /**
     * @param  list<array{phrase: string, weight: float, is_regex?: bool}>  $items
     * @return array{score: float, matched: list<string>}
     */
    private function scoreKeywordCategory(string $lowerRemarks, string $rawRemarks, array $items): array
    {
        if ($items === []) {
            return ['score' => 0.0, 'matched' => []];
        }
        $totalWeight = 0.0;
        $matchedWeight = 0.0;
        $matched = [];
        foreach ($items as $item) {
            $phrase = (string) ($item['phrase'] ?? '');
            $w = (float) ($item['weight'] ?? 0);
            $totalWeight += $w;
            $isRegex = (bool) ($item['is_regex'] ?? false);
            $hit = false;
            if ($phrase !== '') {
                if ($isRegex) {
                    set_error_handler(static fn () => true);
                    try {
                        $hit = @preg_match($phrase, $rawRemarks) === 1;
                    } catch (\Throwable) {
                        $hit = false;
                    }
                    restore_error_handler();
                } else {
                    $hit = str_contains($lowerRemarks, strtolower($phrase));
                }
            }
            if ($hit) {
                $matchedWeight += $w;
                $matched[] = $phrase;
            }
        }
        if ($totalWeight <= 0) {
            return ['score' => 0.0, 'matched' => []];
        }
        $score = min(1.0, $matchedWeight / $totalWeight);

        return ['score' => $score, 'matched' => $matched];
    }

    /**
     * @param  array<string, mixed>  $record
     * @return array{lat: float, lng: float}|null
     */
    private function parseCoordinates(array $record): ?array
    {
        $coords = $record['Coordinates'] ?? null;
        if (is_array($coords)) {
            if (isset($coords['coordinates']) && is_array($coords['coordinates'])) {
                $pair = $coords['coordinates'];
                if (count($pair) >= 2 && is_numeric($pair[0]) && is_numeric($pair[1])) {
                    return ['lng' => (float) $pair[0], 'lat' => (float) $pair[1]];
                }
            }
            if (array_key_exists(0, $coords) && array_key_exists(1, $coords)
                && is_numeric($coords[0]) && is_numeric($coords[1])) {
                return ['lng' => (float) $coords[0], 'lat' => (float) $coords[1]];
            }
        }

        $lat = $record['Latitude'] ?? null;
        $lng = $record['Longitude'] ?? null;
        if (is_numeric($lat) && is_numeric($lng)) {
            return ['lat' => (float) $lat, 'lng' => (float) $lng];
        }

        return null;
    }

    /**
     * @param  array<string, mixed>  $record
     */
    private function formatAddress(array $record): string
    {
        $parts = array_filter([
            (string) ($record['StreetNumber'] ?? ''),
            (string) ($record['StreetDirPrefix'] ?? ''),
            (string) ($record['StreetName'] ?? ''),
            (string) ($record['StreetSuffix'] ?? ''),
            (string) ($record['StreetDirSuffix'] ?? ''),
        ]);
        $street = trim(implode(' ', $parts));
        if (isset($record['UnitNumber']) && is_string($record['UnitNumber']) && $record['UnitNumber'] !== '') {
            $street .= ' #'.$record['UnitNumber'];
        }
        $tail = array_filter([
            (string) ($record['City'] ?? ''),
            (string) ($record['StateOrProvince'] ?? ''),
            (string) ($record['PostalCode'] ?? ''),
        ]);
        $t = trim(implode(', ', $tail));
        if ($street !== '' && $t !== '') {
            return $street.', '.$t;
        }

        return $street !== '' ? $street : $t;
    }

    private function normalizeListingKey(string $dataset, string $listingId): string
    {
        if (str_contains($listingId, ':')) {
            return $listingId;
        }

        return strtolower($dataset).':'.$listingId;
    }

    private function intOrNull(mixed $v): ?int
    {
        return is_numeric($v) ? (int) $v : null;
    }

    private function floatOrNull(mixed $v): ?float
    {
        return is_numeric($v) ? (float) $v : null;
    }

    private function boolOrNull(mixed $v): ?bool
    {
        return is_bool($v) ? $v : null;
    }

    private function stringOrNull(mixed $v): ?string
    {
        return is_string($v) && trim($v) !== '' ? trim($v) : null;
    }

    /**
     * @param  array<string, mixed>  $validated
     * @param  array<string, mixed>  $subject
     * @return array{closed: list<array<string,mixed>>, active: list<array<string,mixed>>, closed_total: int, active_total: int}
     */
    private function loadRentalComps(string $dataset, array $validated, array $subject): array
    {
        $filters = $validated['filters'] ?? [];
        $monthsBack = (int) ($filters['sold_months_back'] ?? 12);
        $closedSince = CarbonImmutable::now()->subMonths($monthsBack)->format('Y-m-d');
        $orderBy = $this->buildOrderByDistance($subject, $validated);

        $closedFilter = $this->buildRentalFilter($subject, $validated, $filters, true, $closedSince);
        $activeFilter = $this->buildRentalFilter($subject, $validated, $filters, false, $closedSince);

        $closedResult = $this->searchClient->search(
            $dataset,
            $closedFilter,
            $orderBy,
            120,
            0,
            $this->searchClient->compsPropertySelectList($dataset),
            '',
        );
        $activeResult = $this->searchClient->search(
            $dataset,
            $activeFilter,
            $orderBy,
            120,
            0,
            $this->searchClient->compsPropertySelectList($dataset),
            '',
        );

        return [
            'closed' => $this->rankAndSlice($closedResult['value'], $subject, $this->normalizationDistanceMiles($validated), 25),
            'active' => $this->rankAndSlice($activeResult['value'], $subject, $this->normalizationDistanceMiles($validated), 25),
            'closed_total' => count($closedResult['value']),
            'active_total' => count($activeResult['value']),
        ];
    }

    /**
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $validated
     * @param  array<string, mixed>  $filters
     */
    private function buildRentalFilter(array $subject, array $validated, array $filters, bool $closed, string $closedSince): string
    {
        $clauses = ["PropertyType eq 'Residential Lease'"];
        if ($closed) {
            $clauses[] = "tolower(StandardStatus) eq 'closed'";
            $clauses[] = "CloseDate ge {$closedSince}";
        } else {
            $clauses[] = "(tolower(StandardStatus) eq 'active' or tolower(StandardStatus) eq 'pending')";
            $clauses[] = 'ListPrice gt 0';
        }

        if ($subject['listing_key'] !== '') {
            $escaped = str_replace("'", "''", $subject['listing_key']);
            $clauses[] = "ListingKey ne '{$escaped}'";
        }

        $clauses = array_merge($clauses, $this->scopeClauses($subject, $validated));

        // Rental similarity should be broad; only keep hard bedroom/bath/size/year guardrails.
        $rentalFilters = $filters;
        $rentalFilters['match_pool'] = false;
        $rentalFilters['match_hoa'] = false;
        $rentalFilters['match_property_sub_type'] = false;
        $rentalFilters['match_waterfront'] = false;
        $rentalFilters['match_view'] = false;
        $rentalFilters['match_subdivision'] = false;
        $rentalFilters['match_mls_area_major'] = false;
        $rentalFilters['match_senior_community'] = false;
        $rentalFilters['living_area_pct'] = (int) ($filters['living_area_pct'] ?? 20);
        $rentalFilters['lot_size_pct'] = (int) ($filters['lot_size_pct'] ?? 100);
        $rentalFilters['year_built_tolerance'] = (int) ($filters['year_built_tolerance'] ?? 30);
        $rentalFilters['beds_tolerance'] = (int) ($filters['beds_tolerance'] ?? 2);
        $rentalFilters['baths_tolerance'] = (int) ($filters['baths_tolerance'] ?? 2);

        $clauses = array_merge($clauses, $this->toleranceClauses($subject, $rentalFilters));

        return implode(' and ', $clauses);
    }

    private function resolveMonthlyRent(array $row): ?float
    {
        $status = strtolower((string) ($row['StandardStatus'] ?? ''));
        $base = $status === 'closed'
            ? $this->floatOrNull($row['ClosePrice'] ?? null)
            : $this->floatOrNull($row['ListPrice'] ?? null);
        if ($base === null) {
            return null;
        }

        $frequency = strtolower(trim((string) ($row['LeaseAmountFrequency'] ?? 'monthly')));
        if ($frequency === 'annually' || $frequency === 'annual') {
            return round($base / 12.0, 2);
        }

        return $base;
    }

    /**
     * @param  list<array<string,mixed>>  $rows
     * @return list<float>
     */
    private function monthlyRentsFromRows(array $rows): array
    {
        $rents = [];
        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }
            $rent = $this->resolveMonthlyRent($row);
            if ($rent !== null && $rent > 0) {
                $rents[] = $rent;
            }
        }

        return $rents;
    }

    /**
     * @param  list<array<string,mixed>>  $rows
     * @param  array<string,mixed>  $subject
     * @return list<array<string,mixed>>
     */
    private function mapRentalComps(array $rows, array $subject): array
    {
        $out = [];
        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }
            $coords = $this->parseCoordinates($row);
            $out[] = [
                'listing_id' => Str::afterLast((string) ($row['ListingKey'] ?? ''), ':'),
                'address' => $this->formatAddress($row),
                'status' => (string) ($row['StandardStatus'] ?? ''),
                'monthly_rent' => $this->resolveMonthlyRent($row),
                'lease_amount_frequency' => $this->stringOrNull($row['LeaseAmountFrequency'] ?? null),
                'bedrooms' => $this->intOrNull($row['BedroomsTotal'] ?? null),
                'bathrooms' => $this->floatOrNull($row['BathroomsTotalDecimal'] ?? null),
                'living_area_sqft' => $this->intOrNull($row['LivingArea'] ?? null),
                'distance_miles' => round($this->distanceMiles($subject, $row), 3),
                'dom' => $this->intOrNull($row['DaysOnMarket'] ?? null),
                'lat' => $coords['lat'] ?? 0.0,
                'lng' => $coords['lng'] ?? 0.0,
            ];
        }

        return $out;
    }

    /**
     * @param  array<string,mixed>  $params
     * @return array<string,float>
     */
    private function normalizeRentalParams(array $params): array
    {
        return [
            'purchase_price' => (float) ($params['purchase_price'] ?? 0),
            'down_payment_percent' => (float) ($params['down_payment_percent'] ?? 20),
            'down_payment_amount' => (float) ($params['down_payment_amount'] ?? 0),
            'interest_rate' => (float) ($params['interest_rate'] ?? 7.25),
            'loan_term_years' => (float) ($params['loan_term_years'] ?? 30),
            'annual_tax' => (float) ($params['annual_tax'] ?? 0),
            'annual_insurance' => (float) ($params['annual_insurance'] ?? 0),
            'monthly_hoa' => (float) ($params['monthly_hoa'] ?? 0),
            'monthly_maintenance' => (float) ($params['monthly_maintenance'] ?? 0),
            'monthly_management' => (float) ($params['monthly_management'] ?? 0),
            'vacancy_rate' => (float) ($params['vacancy_rate'] ?? 5),
        ];
    }

    /**
     * @param  array<string,float>  $params
     * @return array<string,float|null>
     */
    private function computeRentalFinancials(float $estimatedMonthlyRent, array $params): array
    {
        $purchase = max(0.0, $params['purchase_price']);
        $downFromPct = $purchase * (max(0.0, min(100.0, $params['down_payment_percent'])) / 100.0);
        $downPayment = $params['down_payment_amount'] > 0 ? $params['down_payment_amount'] : $downFromPct;
        $loan = max(0.0, $purchase - $downPayment);

        $monthlyRate = max(0.0, $params['interest_rate']) / 100.0 / 12.0;
        $n = max(1.0, $params['loan_term_years'] * 12.0);
        $monthlyPi = 0.0;
        if ($loan > 0) {
            if ($monthlyRate > 0) {
                $monthlyPi = $loan * ($monthlyRate * (1 + $monthlyRate) ** $n) / (((1 + $monthlyRate) ** $n) - 1);
            } else {
                $monthlyPi = $loan / $n;
            }
        }

        $vacancyPct = max(0.0, min(100.0, $params['vacancy_rate'])) / 100.0;
        $effectiveRent = $estimatedMonthlyRent * (1.0 - $vacancyPct);
        $monthlyTax = max(0.0, $params['annual_tax']) / 12.0;
        $monthlyInsurance = max(0.0, $params['annual_insurance']) / 12.0;
        $monthlyOperating = $monthlyTax + $monthlyInsurance + max(0.0, $params['monthly_hoa']) + max(0.0, $params['monthly_maintenance']) + max(0.0, $params['monthly_management']);
        $monthlyCashflow = $effectiveRent - $monthlyOperating - $monthlyPi;
        $annualNoi = ($effectiveRent - $monthlyOperating) * 12.0;
        $annualDebt = $monthlyPi * 12.0;
        $dscr = $annualDebt > 0 ? $annualNoi / $annualDebt : null;
        $capRate = $purchase > 0 ? $annualNoi / $purchase : null;
        $cashInvested = max(1.0, $downPayment);
        $cashOnCash = ($monthlyCashflow * 12.0) / $cashInvested;

        return [
            'estimated_monthly_rent' => round($estimatedMonthlyRent, 2),
            'effective_monthly_rent' => round($effectiveRent, 2),
            'monthly_pi' => round($monthlyPi, 2),
            'monthly_operating_expenses' => round($monthlyOperating, 2),
            'monthly_cashflow' => round($monthlyCashflow, 2),
            'annual_noi' => round($annualNoi, 2),
            'dscr' => $dscr !== null ? round($dscr, 3) : null,
            'cap_rate' => $capRate !== null ? round($capRate, 4) : null,
            'cash_on_cash' => round($cashOnCash, 4),
            'loan_amount' => round($loan, 2),
            'down_payment' => round($downPayment, 2),
        ];
    }

    /**
     * @param  array<string,mixed>  $validated
     * @param  array<string,mixed>  $subject
     * @return array<string,mixed>
     */
    private function handleRentHoldCashflow(Request $request, array $validated, array $subject, string $dataset, bool $fullAccess, float $started): array
    {
        $aggMethod = ($validated['aggregation_method'] ?? 'median') === 'average' ? 'average' : 'median';
        $subject['_dataset'] = $dataset;
        $subject['_sold_months_back'] = (int) (($validated['filters']['sold_months_back'] ?? 12));
        $subject['_sold_since'] = CarbonImmutable::now()->subMonths((int) $subject['_sold_months_back'])->format('Y-m-d');

        $rentalRows = $this->loadRentalComps($dataset, $validated, $subject);
        $closedRents = $this->monthlyRentsFromRows($rentalRows['closed']);
        $activeRents = $this->monthlyRentsFromRows($rentalRows['active']);
        $all = array_merge($closedRents, $activeRents);
        $estimatedRent = $all === [] ? 0.0 : $this->aggregateNumeric($all, $aggMethod);
        $params = $this->normalizeRentalParams(is_array($validated['rental_params'] ?? null) ? $validated['rental_params'] : []);
        $financials = $this->computeRentalFinancials($estimatedRent, $params);

        $processingMs = (int) round((microtime(true) - $started) * 1000);
        $this->audit->log(
            $request,
            'comps.run.rent_hold_cashflow',
            count($rentalRows['closed']),
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return [
            'success' => true,
            'subject' => $this->mapSubjectResponse($subject),
            'sold_comps' => [],
            'competition_comps' => [],
            'failed_listings' => [],
            'overpriced_signals' => [],
            'market_conditions' => null,
            'rental_comps' => [
                'closed' => $this->mapRentalComps($rentalRows['closed'], $subject),
                'active' => $this->mapRentalComps($rentalRows['active'], $subject),
            ],
            'rental_result' => array_merge($financials, [
                'closed_comp_count' => count($rentalRows['closed']),
                'active_comp_count' => count($rentalRows['active']),
            ]),
            'metadata' => $this->buildMetadata(
                $validated,
                $fullAccess,
                $rentalRows['closed_total'],
                $rentalRows['active_total'],
                0,
                $processingMs,
                $aggMethod
            ),
            'warnings' => [],
        ];
    }

    /**
     * @param  list<array<string,mixed>>  $soldComps
     * @return array<string,float|null>
     */
    private function computeFlipFinancials(array $soldComps, array $flipParams): array
    {
        $arv = null;
        $prices = [];
        foreach ($soldComps as $c) {
            if (is_array($c) && isset($c['sold_price']) && is_numeric($c['sold_price'])) {
                $prices[] = (float) $c['sold_price'];
            }
        }
        if ($prices !== []) {
            $arv = $this->aggregateNumeric($prices, 'median');
        }
        if (isset($flipParams['arv_override']) && is_numeric($flipParams['arv_override']) && (float) $flipParams['arv_override'] > 0) {
            $arv = (float) $flipParams['arv_override'];
        }

        $purchase = (float) ($flipParams['purchase_price'] ?? 0);
        $rehab = (float) ($flipParams['rehab_budget'] ?? 0);
        $buyCosts = (float) ($flipParams['closing_costs_buy'] ?? 0);
        $sellPct = (float) ($flipParams['closing_costs_sell_pct'] ?? 7);
        $holdMonths = (float) ($flipParams['holding_months'] ?? 6);
        $holdMonthly = (float) ($flipParams['holding_cost_monthly'] ?? 0);

        $totalCost = $purchase + $rehab + $buyCosts + ($holdMonths * $holdMonthly);
        $saleProceeds = $arv !== null ? $arv * (1.0 - max(0.0, min(100.0, $sellPct)) / 100.0) : null;
        $profit = $saleProceeds !== null ? $saleProceeds - $totalCost : null;
        $roi = ($profit !== null && $totalCost > 0) ? $profit / $totalCost : null;

        return [
            'arv' => $arv !== null ? round($arv, 2) : null,
            'sale_proceeds' => $saleProceeds !== null ? round($saleProceeds, 2) : null,
            'total_project_cost' => round($totalCost, 2),
            'projected_profit' => $profit !== null ? round($profit, 2) : null,
            'projected_roi' => $roi !== null ? round($roi, 4) : null,
        ];
    }

    /**
     * @param  array<string,mixed>  $validated
     * @param  array<string,mixed>  $subject
     * @return array<string,mixed>
     */
    private function handleFlipVsHold(Request $request, array $validated, array $subject, string $dataset, bool $fullAccess, float $started): array
    {
        $filters = $validated['filters'] ?? [];
        $aggMethod = ($validated['aggregation_method'] ?? 'median') === 'average' ? 'average' : 'median';
        $subject['_dataset'] = $dataset;
        $subject['_sold_months_back'] = (int) ($filters['sold_months_back'] ?? 12);
        $subject['_sold_since'] = CarbonImmutable::now()->subMonths((int) $subject['_sold_months_back'])->format('Y-m-d');
        $normDistMiles = $this->normalizationDistanceMiles($validated);

        $soldFilter = $this->buildSoldFilter($subject, $validated, $filters, $subject['_sold_since']);
        $soldRows = $this->searchClient->search(
            $dataset,
            $soldFilter,
            $this->buildOrderByDistance($subject, $validated),
            120,
            0,
            $this->searchClient->compsPropertySelectList($dataset),
            '',
        )['value'];
        $soldRanked = $this->rankAndSlice($soldRows, $subject, $normDistMiles, 15);
        $medianPpsf = $this->centralPpsfFromSold($soldRanked, $aggMethod);
        $soldMapped = array_map(fn (array $row) => $this->mapSoldComp($row, $subject, $medianPpsf, $filters, 'A', [], $normDistMiles), $soldRanked);

        $rentalRows = $this->loadRentalComps($dataset, $validated, $subject);
        $rentValues = array_merge($this->monthlyRentsFromRows($rentalRows['closed']), $this->monthlyRentsFromRows($rentalRows['active']));
        $estimatedRent = $rentValues === [] ? 0.0 : $this->aggregateNumeric($rentValues, $aggMethod);
        $holdFinancials = $this->computeRentalFinancials(
            $estimatedRent,
            $this->normalizeRentalParams(is_array($validated['rental_params'] ?? null) ? $validated['rental_params'] : [])
        );

        $flipFinancials = $this->computeFlipFinancials($soldMapped, is_array($validated['flip_params'] ?? null) ? $validated['flip_params'] : []);

        $processingMs = (int) round((microtime(true) - $started) * 1000);
        $this->audit->log(
            $request,
            'comps.run.flip_vs_hold',
            count($soldMapped),
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return [
            'success' => true,
            'subject' => $this->mapSubjectResponse($subject),
            'sold_comps' => $soldMapped,
            'competition_comps' => [],
            'failed_listings' => [],
            'overpriced_signals' => [],
            'market_conditions' => $this->computeMarketConditions($soldRanked, count($soldRows), 0, (int) $subject['_sold_months_back'], $aggMethod),
            'flip_vs_hold_result' => [
                'flip' => $flipFinancials,
                'hold' => $holdFinancials,
                'recommendation' => $this->recommendFlipOrHold($flipFinancials, $holdFinancials),
            ],
            'rental_comps' => [
                'closed' => $this->mapRentalComps($rentalRows['closed'], $subject),
                'active' => $this->mapRentalComps($rentalRows['active'], $subject),
            ],
            'metadata' => $this->buildMetadata($validated, $fullAccess, count($soldRows), 0, 0, $processingMs, $aggMethod),
            'warnings' => [],
        ];
    }

    /**
     * @param  array<string,float|null>  $flip
     * @param  array<string,float|null>  $hold
     */
    private function recommendFlipOrHold(array $flip, array $hold): string
    {
        $flipRoi = $flip['projected_roi'] ?? null;
        $holdCoC = $hold['cash_on_cash'] ?? null;
        if ($flipRoi === null && $holdCoC === null) {
            return 'insufficient_data';
        }
        if ($flipRoi !== null && $holdCoC !== null) {
            return $flipRoi > $holdCoC ? 'flip' : 'hold';
        }

        return $flipRoi !== null ? 'flip' : 'hold';
    }

    /**
     * @param  array<string,mixed>  $validated
     * @param  array<string,mixed>  $subject
     * @return array<string,mixed>
     */
    private function handleAppraiserSimulation(Request $request, array $validated, array $subject, string $dataset, bool $fullAccess, float $started): array
    {
        $filters = $validated['filters'] ?? [];
        $aggMethod = ($validated['aggregation_method'] ?? 'median') === 'average' ? 'average' : 'median';
        $subject['_dataset'] = $dataset;
        $subject['_sold_months_back'] = (int) ($filters['sold_months_back'] ?? 12);
        $subject['_sold_since'] = CarbonImmutable::now()->subMonths((int) $subject['_sold_months_back'])->format('Y-m-d');
        $normDistMiles = $this->normalizationDistanceMiles($validated);
        $simParams = is_array($validated['simulation_params'] ?? null) ? $validated['simulation_params'] : [];
        $supportN = min(25, max(3, (int) ($simParams['supporting_comp_count'] ?? 8)));
        $highAdj = (float) ($simParams['high_adjustment_threshold_pct'] ?? 25.0);

        $soldFilter = $this->buildSoldFilter($subject, $validated, $filters, $subject['_sold_since']);
        $soldRows = $this->searchClient->search(
            $dataset,
            $soldFilter,
            $this->buildOrderByDistance($subject, $validated),
            120,
            0,
            $this->searchClient->compsPropertySelectList($dataset),
            '',
        )['value'];
        $soldRanked = $this->rankAndSlice($soldRows, $subject, $normDistMiles, $supportN);
        $medianPpsf = $this->centralPpsfFromSold($soldRanked, $aggMethod);
        $soldMapped = array_map(fn (array $row) => $this->mapSoldComp($row, $subject, $medianPpsf, $filters, 'A', [], $normDistMiles), $soldRanked);

        $adjustedValues = [];
        $grosses = [];
        foreach ($soldMapped as $comp) {
            if (isset($comp['adjustments']['adjusted_price']) && is_numeric($comp['adjustments']['adjusted_price'])) {
                $adjustedValues[] = (float) $comp['adjustments']['adjusted_price'];
            } elseif (isset($comp['sold_price']) && is_numeric($comp['sold_price'])) {
                $adjustedValues[] = (float) $comp['sold_price'];
            }
            if (isset($comp['adjustments']['gross_adjustment_pct']) && is_numeric($comp['adjustments']['gross_adjustment_pct'])) {
                $grosses[] = (float) $comp['adjustments']['gross_adjustment_pct'];
            }
        }

        $estimate = $adjustedValues === [] ? null : $this->aggregateNumeric($adjustedValues, $aggMethod);
        $avgGrossAdj = $grosses === [] ? null : $this->aggregateNumeric($grosses, 'average');
        $risk = $this->simulationRiskScore(count($soldMapped), $avgGrossAdj, $highAdj);
        $bandWidth = $risk >= 70 ? 0.10 : ($risk >= 40 ? 0.075 : 0.05);

        $processingMs = (int) round((microtime(true) - $started) * 1000);
        $this->audit->log(
            $request,
            'comps.run.appraiser_simulation',
            count($soldMapped),
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return [
            'success' => true,
            'subject' => $this->mapSubjectResponse($subject),
            'sold_comps' => $soldMapped,
            'competition_comps' => [],
            'failed_listings' => [],
            'overpriced_signals' => [],
            'market_conditions' => $this->computeMarketConditions($soldRanked, count($soldRows), 0, (int) $subject['_sold_months_back'], $aggMethod),
            'simulation_result' => [
                'indicated_value' => $estimate !== null ? round($estimate, 2) : null,
                'bpo_low' => $estimate !== null ? round($estimate * (1 - $bandWidth), 2) : null,
                'bpo_high' => $estimate !== null ? round($estimate * (1 + $bandWidth), 2) : null,
                'avg_gross_adjustment_pct' => $avgGrossAdj !== null ? round($avgGrossAdj, 2) : null,
                'risk_score' => $risk,
                'risk_band' => $risk >= 70 ? 'high' : ($risk >= 40 ? 'moderate' : 'low'),
                'supporting_comp_count' => count($soldMapped),
            ],
            'metadata' => $this->buildMetadata($validated, $fullAccess, count($soldRows), 0, 0, $processingMs, $aggMethod),
            'warnings' => [],
        ];
    }

    private function simulationRiskScore(int $compCount, ?float $avgGrossAdj, float $highAdjThreshold): int
    {
        $score = 0.0;
        if ($compCount < 5) {
            $score += 40;
        } elseif ($compCount < 8) {
            $score += 20;
        }

        if ($avgGrossAdj !== null) {
            if ($avgGrossAdj >= $highAdjThreshold) {
                $score += 40;
            } elseif ($avgGrossAdj >= $highAdjThreshold * 0.7) {
                $score += 20;
            }
        } else {
            $score += 20;
        }

        return (int) max(0, min(100, round($score)));
    }

    /**
     * @param  array<string, mixed>  $validated
     * @return array<string, mixed>
     */
    private function buildMetadata(
        array $validated,
        bool $fullAccess,
        int $soldCandidates,
        int $compCandidates,
        int $failedCandidates,
        int $processingMs,
        string $aggMethod,
    ): array {
        $scope = $validated['scope'] ?? [];
        $filters = $validated['filters'] ?? [];

        return [
            'total_sold_candidates' => $soldCandidates,
            'total_competition_candidates' => $compCandidates,
            'total_failed_candidates' => $failedCandidates,
            'scope_applied' => (string) ($scope['type'] ?? ''),
            'radius_miles' => (float) ($scope['radius_miles'] ?? 0),
            'processing_ms' => $processingMs,
            'mlg_can_view_filtered' => 0,
            'overpriced_available' => (bool) (($filters['include_overpriced_signals'] ?? true) && $fullAccess),
            'overpriced_unavailable_reason' => $fullAccess ? null : 'Requires idx:full token for competition and overpriced signals.',
            'failed_listings_available' => false,
            'aggregation_method' => $aggMethod,
            'comps_teaser_cap' => $fullAccess ? null : 3,
        ];
    }

    /**
     * @return list<string>
     */
    private function warningsForCaps(bool $fullAccess, int $requestedMax, int $appliedMax): array
    {
        if ($fullAccess || $requestedMax <= $appliedMax) {
            return [];
        }

        return ['Sold comps capped at '.$appliedMax.' for non-full-access requests (requested '.$requestedMax.').'];
    }

    /**
     * Build a subject array from home_value mode fields.
     *
     * Two paths:
     *  1. listing_id provided → fetch from Bridge, auto-populate all fields
     *  2. address provided → geocode, use owner-supplied fields
     *
     * @param  array<string, mixed>  $subjectIn
     * @return array<string, mixed>|null
     */
    private function subjectFromHomeValueFields(Request $request, string $dataset, array $subjectIn): ?array
    {
        $listingId = trim((string) ($subjectIn['listing_id'] ?? ''));
        $address = trim((string) ($subjectIn['address'] ?? ''));

        // Path 1: listing_id provided — fetch from Bridge
        if ($listingId !== '') {
            $listingKey = $this->normalizeListingKey($dataset, $listingId);
            $record = $this->searchClient->getPropertyForComps($request, $dataset, $listingKey);

            if ($record === null) {
                return null;
            }

            return $this->subjectFromHomeValueListing($record, $subjectIn, $dataset);
        }

        // Path 2: address provided — geocode
        if ($address === '') {
            return null;
        }

        $result = $this->geocodingService->geocode($address);
        if ($result === null) {
            return null;
        }

        return $this->subjectFromHomeValueAddress($subjectIn, $result);
    }

    /**
     * Build a subject from a Bridge listing record for home_value mode.
     *
     * @param  array<string, mixed>  $record
     * @param  array<string, mixed>  $subjectIn
     * @return array<string, mixed>
     */
    private function subjectFromHomeValueListing(array $record, array $subjectIn, string $dataset): array
    {
        $coords = $this->parseCoordinates($record);
        $propertySubType = $this->stringOrNull($record['PropertySubType'] ?? null);

        // Derive condition: explicit override > PropertyCondition > PublicRemarks analysis
        $condition = $subjectIn['condition'] ?? null;
        if ($condition === null || trim((string) $condition) === '') {
            $condition = $this->deriveCondition($record);
        }

        $fullBath = $this->intOrNull($record['BathroomsFull'] ?? null);
        $halfBath = $this->intOrNull($record['BathroomsHalf'] ?? null) ?? 0;
        $totalBath = $fullBath !== null ? (float) $fullBath + ($halfBath * 0.5) : null;

        $lotAcres = $this->floatOrNull($record['LotSizeAcres'] ?? null);

        $garage = $this->garageSpacesIntFromRow($record);

        return [
            'listing_key' => (string) ($record['ListingKey'] ?? ''),
            'lat' => $coords['lat'] ?? 0.0,
            'lng' => $coords['lng'] ?? 0.0,
            'bedrooms' => $this->intOrNull($record['BedroomsTotal'] ?? null),
            'bathrooms' => $totalBath,
            'living_area' => $this->intOrNull($record['LivingArea'] ?? null),
            'lot_acres' => $lotAcres,
            'year_built' => $this->intOrNull($record['YearBuilt'] ?? null),
            'pool' => $this->boolOrNull($record['PoolPrivateYN'] ?? null),
            'hoa' => $this->boolOrNull($record['AssociationYN'] ?? null),
            'senior_community' => null,
            'waterfront' => $this->boolOrNull($record['WaterfrontYN'] ?? null),
            'garage_spaces' => $garage,
            'parking_stalls_total' => $this->parkingStallTotalFromRow($record),
            'subdivision_name' => $this->stringOrNull($record['SubdivisionName'] ?? null),
            'mls_area_major' => $this->stringOrNull($record['MLSAreaMajor'] ?? null),
            'view_yn' => $this->boolOrNull($record['ViewYN'] ?? null),
            'monthly_fees' => null,
            'list_price' => $this->floatOrNull($record['ListPrice'] ?? null),
            'property_type' => $propertySubType !== null ? 'Residential' : null,
            'property_sub_type' => $propertySubType,
            'flood_zone_codes' => [],
            'address' => $this->formatAddress($record),
            'condition' => $condition,
            'renovations' => $this->buildRenovationsArray($subjectIn),
            'stories' => $this->intOrNull($record['StoriesTotal'] ?? null),
            'full_bathrooms' => $fullBath,
            'half_bathrooms' => $halfBath,
            'property_type_enum' => $this->mapPropertySubTypeToWidgetEnum($propertySubType),
        ];
    }

    /**
     * Build a subject from owner-provided address and fields for home_value mode.
     *
     * @param  array<string, mixed>  $subjectIn
     * @param  object{lat: float, lng: float, formattedAddress: string}  $geocodeResult
     * @return array<string, mixed>
     */
    private function subjectFromHomeValueAddress(array $subjectIn, object $geocodeResult): array
    {
        $lotAcres = null;
        if (isset($subjectIn['lot_size_sqft']) && is_numeric($subjectIn['lot_size_sqft'])) {
            $lotAcres = (float) $subjectIn['lot_size_sqft'] / 43560.0;
        }

        $propertySubType = $this->mapWidgetEnumToPropertySubType($subjectIn['property_type'] ?? '');

        $fullBath = isset($subjectIn['full_bathrooms']) ? (int) $subjectIn['full_bathrooms'] : null;
        $halfBath = isset($subjectIn['half_bathrooms']) ? (int) $subjectIn['half_bathrooms'] : 0;
        $totalBath = $fullBath !== null ? (float) $fullBath + ($halfBath * 0.5) : null;

        $condition = $subjectIn['condition'] ?? null;
        if ($condition !== null && trim((string) $condition) === '') {
            $condition = null;
        }

        return [
            'listing_key' => '',
            'lat' => $geocodeResult->lat,
            'lng' => $geocodeResult->lng,
            'bedrooms' => isset($subjectIn['bedrooms']) ? (int) $subjectIn['bedrooms'] : null,
            'bathrooms' => $totalBath,
            'living_area' => isset($subjectIn['living_area_sqft']) ? (int) $subjectIn['living_area_sqft'] : null,
            'lot_acres' => $lotAcres,
            'year_built' => isset($subjectIn['year_built']) ? (int) $subjectIn['year_built'] : null,
            'pool' => isset($subjectIn['pool']) ? (bool) $subjectIn['pool'] : null,
            'hoa' => null,
            'senior_community' => null,
            'waterfront' => isset($subjectIn['waterfront']) ? (bool) $subjectIn['waterfront'] : null,
            'garage_spaces' => isset($subjectIn['garage_spaces']) ? (int) $subjectIn['garage_spaces'] : null,
            'parking_stalls_total' => isset($subjectIn['garage_spaces']) ? (int) $subjectIn['garage_spaces'] : null,
            'subdivision_name' => null,
            'mls_area_major' => null,
            'view_yn' => null,
            'monthly_fees' => isset($subjectIn['hoa_monthly_fee']) && is_numeric($subjectIn['hoa_monthly_fee']) ? (float) $subjectIn['hoa_monthly_fee'] : null,
            'list_price' => null,
            'property_type' => $propertySubType !== null ? 'Residential' : null,
            'property_sub_type' => $propertySubType,
            'flood_zone_codes' => [],
            'address' => $geocodeResult->formattedAddress,
            'condition' => $condition,
            'renovations' => $this->buildRenovationsArray($subjectIn),
            'stories' => isset($subjectIn['stories']) ? (int) $subjectIn['stories'] : null,
            'full_bathrooms' => $fullBath,
            'half_bathrooms' => $halfBath,
            'property_type_enum' => $subjectIn['property_type'] ?? null,
        ];
    }

    /**
     * Derive condition from PropertyCondition field or PublicRemarks keyword analysis.
     *
     * @param  array<string, mixed>  $record
     */
    private function deriveCondition(array $record): ?string
    {
        // 1. Check PropertyCondition field
        // Stellar MLS PropertyCondition describes construction state, not quality:
        //   "Existing", "New Construction", "Completed", "Fixer", "Proposed",
        //   "Pre-Construction", "Under Construction", "Under Renovation"
        // Only "Fixer" maps to a condition quality rating.
        $propertyCondition = $record['PropertyCondition'] ?? null;
        if ($propertyCondition !== null) {
            $conditions = is_array($propertyCondition) ? $propertyCondition : [$propertyCondition];
            foreach ($conditions as $value) {
                $lower = strtolower(trim((string) $value));
                if ($lower === 'fixer') {
                    return 'poor';
                }
            }
        }

        // 2. Analyze PublicRemarks for condition keywords
        $remarks = strtolower(trim((string) ($record['PublicRemarks'] ?? '')));
        if ($remarks === '') {
            return null;
        }

        $excellentKeywords = ['mint condition', 'turnkey', 'completely renovated', 'like new', 'shows like a model', 'no expense spared', 'gut renovated', 'fully renovated', 'brand new'];
        $goodKeywords = ['well maintained', 'well-maintained', 'move-in ready', 'move in ready', 'updated', 'good condition', 'nicely kept', 'immaculate', 'pristine'];
        $fairKeywords = ['needs some tlc', 'needs updating', 'being sold as-is', 'as-is', 'handyman special', 'needs cosmetic work', 'some updating needed', 'needs some work', 'priced to sell'];
        $poorKeywords = ['needs major work', 'fixer upper', 'fixer-upper', 'tear down', 'tear-down', 'needs everything', 'major renovation', 'distressed', 'investor special', 'total rehab'];

        foreach ($poorKeywords as $keyword) {
            if (str_contains($remarks, $keyword)) {
                return 'poor';
            }
        }

        foreach ($excellentKeywords as $keyword) {
            if (str_contains($remarks, $keyword)) {
                return 'excellent';
            }
        }

        foreach ($goodKeywords as $keyword) {
            if (str_contains($remarks, $keyword)) {
                return 'good';
            }
        }

        foreach ($fairKeywords as $keyword) {
            if (str_contains($remarks, $keyword)) {
                return 'fair';
            }
        }

        return null;
    }

    /**
     * Map a RESO PropertySubType value back to our simplified widget enum.
     */
    private function mapPropertySubTypeToWidgetEnum(?string $propertySubType): ?string
    {
        return match ($propertySubType) {
            'Single Family Residence' => 'sfr',
            'Townhouse' => 'townhouse',
            'Condominium', 'Condo - Hotel' => 'condo',
            'Manufactured Home', 'Manufactured On Land', 'Mobile Home' => 'manufactured',
            'Duplex', '1/2 Duplex' => 'duplex',
            'Triplex' => 'triplex',
            'Quadruplex' => 'quadplex',
            'Modular Home' => 'modular',
            'Apartment', 'Villa', 'Multi Family (5+)', 'Residential' => 'sfr',
            default => null,
        };
    }

    /**
     * Map our simplified widget enum to a RESO PropertySubType value.
     */
    private function mapWidgetEnumToPropertySubType(string $enum): ?string
    {
        return match ($enum) {
            'sfr' => 'Single Family Residence',
            'townhouse' => 'Townhouse',
            'condo' => 'Condominium',
            'manufactured' => 'Manufactured Home',
            'duplex' => 'Duplex',
            'triplex' => 'Triplex',
            'quadplex' => 'Quadruplex',
            'modular' => 'Modular Home',
            default => null,
        };
    }

    /**
     * @param  array<string, mixed>  $subjectIn
     * @return array<string, int|null>
     */
    private function buildRenovationsArray(array $subjectIn): array
    {
        return [
            'kitchen_year' => isset($subjectIn['renovated_kitchen_year']) ? (int) $subjectIn['renovated_kitchen_year'] : null,
            'bathrooms_year' => isset($subjectIn['renovated_bathrooms_year']) ? (int) $subjectIn['renovated_bathrooms_year'] : null,
            'hvac_year' => isset($subjectIn['renovated_hvac_year']) ? (int) $subjectIn['renovated_hvac_year'] : null,
        ];
    }

    /**
     * Revenue impact: Home Value Estimator is a lead-gen widget that exposes
     * the BPO engine to off-market homeowners, driving subscription conversions.
     *
     * @param  array<string, mixed>  $validated
     * @param  array<string, mixed>  $subject
     * @return array<string, mixed>
     */
    private function handleHomeValue(Request $request, array $validated, array $subject, string $dataset, bool $fullAccess, float $started): array
    {
        $homeValueParams = is_array($validated['home_value_params'] ?? null) ? $validated['home_value_params'] : [];
        $aggMethod = ($validated['aggregation_method'] ?? 'median') === 'average' ? 'average' : 'median';
        $subject['_dataset'] = $dataset;
        $soldMonthsBack = (int) ($homeValueParams['sold_months_back'] ?? 12);
        $subject['_sold_months_back'] = $soldMonthsBack;
        $soldSince = CarbonImmutable::now()->subMonths($soldMonthsBack)->format('Y-m-d');
        $subject['_sold_since'] = $soldSince;
        $normDistMiles = $this->normalizationDistanceMiles($validated);
        $maxComps = min(25, max(3, (int) ($homeValueParams['max_comps'] ?? 8)));

        $condition = $subject['condition'] ?? null;
        $renovations = $subject['renovations'] ?? null;

        // Build filters for home_value with property type constraint
        $filters = $validated['filters'] ?? [];
        $soldFilter = $this->buildSoldFilter($subject, $validated, $filters, $soldSince);

        // Add property type constraint if available
        if ($subject['property_sub_type'] !== null) {
            $escaped = str_replace("'", "''", $subject['property_sub_type']);
            $soldFilter .= " and PropertySubType eq '{$escaped}'";
        }

        // Add stories constraint if provided
        if (isset($subject['stories']) && $subject['stories'] !== null) {
            $soldFilter .= ' and StoriesTotal eq '.((int) $subject['stories']);
        }

        $soldRows = $this->searchClient->search(
            $dataset,
            $soldFilter,
            $this->buildOrderByDistance($subject, $validated),
            120,
            0,
            $this->searchClient->compsPropertySelectList($dataset),
            '',
        )['value'];

        $soldRanked = $this->rankAndSlice($soldRows, $subject, $normDistMiles, $maxComps);

        $rates = $this->bpoExtractor->extractRates($soldRanked);

        $gridEntries = [];
        $adjustedComps = [];
        foreach ($soldRanked as $i => $row) {
            $grid = $this->bpoEngine->adjust($subject, $row, $rates, $soldRanked, $condition, $renovations);
            $close = $this->floatOrNull($row['ClosePrice'] ?? null);
            $closeDate = null;
            if (isset($row['CloseDate']) && is_string($row['CloseDate']) && $row['CloseDate'] !== '') {
                try {
                    $closeDate = (new \DateTimeImmutable($row['CloseDate']))->format('Y-m-d');
                } catch (\Throwable) {
                    $closeDate = null;
                }
            }

            $gridEntries[] = [
                'comp_index' => $i,
                'listing_id' => Str::afterLast((string) ($row['ListingKey'] ?? ''), ':'),
                'address' => $this->formatAddress($row),
                'sale_price' => $close,
                'sale_date' => $closeDate,
                'distance_miles' => round($this->distanceMiles($subject, $row), 3),
                'lines' => $grid['lines'],
                'net_adjustment' => $grid['net_adjustment'],
                'gross_adjustment' => $grid['gross_adjustment'],
                'gross_adjustment_pct' => $grid['gross_adjustment_pct'],
                'adjusted_price' => $grid['adjusted_price'],
            ];

            $adjustedComps[] = [
                'adjusted_price' => $grid['adjusted_price'],
                'gross_adjustment_pct' => $grid['gross_adjustment_pct'],
                'comp' => $row,
            ];
        }

        $reconciliation = $this->bpoEngine->reconcile($subject, $adjustedComps, $rates);

        $marketConditions = $this->computeMarketConditions($soldRanked, count($soldRows), 0, $soldMonthsBack, $aggMethod);

        $processingMs = (int) round((microtime(true) - $started) * 1000);

        $this->audit->log(
            $request,
            'comps.run.home_value',
            count($soldRanked),
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return [
            'success' => true,
            'subject' => $this->mapSubjectResponse($subject),
            'sold_comps' => [],
            'competition_comps' => [],
            'failed_listings' => [],
            'overpriced_signals' => [],
            'market_conditions' => $marketConditions,
            'home_value_result' => [
                'point_estimate' => $reconciliation['point_estimate'],
                'range' => $reconciliation['range'],
                'confidence' => $reconciliation['confidence'],
                'confidence_band' => $reconciliation['confidence_band'],
                'reconciliation_summary' => $reconciliation['reconciliation_summary'],
                'condition' => $condition,
                'renovation_credits_applied' => $renovations !== null,
                'market_rates' => [
                    'gla_per_sf' => round($rates['gla_per_sf'], 2),
                    'bed_per_room' => round($rates['bed_per_room']),
                    'bath_per_full' => round($rates['bath_per_full']),
                    'age_per_year' => round($rates['age_per_year']),
                    'lot_per_acre' => round($rates['lot_per_acre']),
                    'garage_per_space' => round($rates['garage_per_space']),
                    'pool_value' => round($rates['pool_value']),
                    'waterfront_value' => round($rates['waterfront_value']),
                    'time_per_month_pct' => round($rates['time_per_month_pct'], 4),
                    'method' => $rates['method'],
                    'r_squared' => $rates['r_squared'],
                    'comp_count' => $rates['comp_count'],
                    'warnings' => $rates['warnings'],
                ],
                'adjustment_grid' => $gridEntries,
            ],
            'metadata' => $this->buildMetadata($validated, $fullAccess, count($soldRows), 0, 0, $processingMs, $aggMethod),
            'warnings' => $rates['warnings'] ?? [],
        ];
    }

    /**
     * Revenue impact: BPO mode is the flagship Mega-tier comp engine feature.
     * Market-derived adjustments produce appraisal-grade estimates, driving
     * upgrade conversions from Smart/Pro tiers.
     *
     * @param  array<string, mixed>  $validated
     * @param  array<string, mixed>  $subject
     * @return array<string, mixed>
     */
    private function handleBpo(Request $request, array $validated, array $subject, string $dataset, bool $fullAccess, float $started): array
    {
        $bpoParams = is_array($validated['bpo_params'] ?? null) ? $validated['bpo_params'] : [];
        $aggMethod = ($validated['aggregation_method'] ?? 'median') === 'average' ? 'average' : 'median';
        $subject['_dataset'] = $dataset;
        $soldMonthsBack = (int) ($bpoParams['sold_months_back'] ?? 12);
        $subject['_sold_months_back'] = $soldMonthsBack;
        $soldSince = CarbonImmutable::now()->subMonths($soldMonthsBack)->format('Y-m-d');
        $subject['_sold_since'] = $soldSince;
        $normDistMiles = $this->normalizationDistanceMiles($validated);
        $maxComps = min(25, max(3, (int) ($bpoParams['max_comps'] ?? 8)));

        $soldFilter = $this->buildSoldFilter($subject, $validated, $validated['filters'] ?? [], $soldSince);
        $soldRows = $this->searchClient->search(
            $dataset,
            $soldFilter,
            $this->buildOrderByDistance($subject, $validated),
            120,
            0,
            $this->searchClient->compsPropertySelectList($dataset),
            '',
        )['value'];

        $soldRanked = $this->rankAndSlice($soldRows, $subject, $normDistMiles, $maxComps);

        $rates = $this->bpoExtractor->extractRates($soldRanked);

        $gridEntries = [];
        $adjustedComps = [];
        foreach ($soldRanked as $i => $row) {
            $grid = $this->bpoEngine->adjust($subject, $row, $rates, $soldRanked);
            $coords = $this->parseCoordinates($row);
            $close = $this->floatOrNull($row['ClosePrice'] ?? null);
            $closeDate = null;
            if (isset($row['CloseDate']) && is_string($row['CloseDate']) && $row['CloseDate'] !== '') {
                try {
                    $closeDate = (new \DateTimeImmutable($row['CloseDate']))->format('Y-m-d');
                } catch (\Throwable) {
                    $closeDate = null;
                }
            }

            $gridEntries[] = [
                'comp_index' => $i,
                'listing_id' => Str::afterLast((string) ($row['ListingKey'] ?? ''), ':'),
                'address' => $this->formatAddress($row),
                'sale_price' => $close,
                'sale_date' => $closeDate,
                'distance_miles' => round($this->distanceMiles($subject, $row), 3),
                'lines' => $grid['lines'],
                'net_adjustment' => $grid['net_adjustment'],
                'gross_adjustment' => $grid['gross_adjustment'],
                'gross_adjustment_pct' => $grid['gross_adjustment_pct'],
                'adjusted_price' => $grid['adjusted_price'],
            ];

            $adjustedComps[] = [
                'adjusted_price' => $grid['adjusted_price'],
                'gross_adjustment_pct' => $grid['gross_adjustment_pct'],
                'comp' => $row,
            ];
        }

        $reconciliation = $this->bpoEngine->reconcile($subject, $adjustedComps, $rates);

        $marketConditions = $this->computeMarketConditions($soldRanked, count($soldRows), 0, $soldMonthsBack, $aggMethod);

        $processingMs = (int) round((microtime(true) - $started) * 1000);

        $this->audit->log(
            $request,
            'comps.run.bpo',
            count($soldRanked),
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return [
            'success' => true,
            'subject' => $this->mapSubjectResponse($subject),
            'sold_comps' => [],
            'competition_comps' => [],
            'failed_listings' => [],
            'overpriced_signals' => [],
            'market_conditions' => $marketConditions,
            'bpo_result' => [
                'point_estimate' => $reconciliation['point_estimate'],
                'range' => $reconciliation['range'],
                'confidence' => $reconciliation['confidence'],
                'confidence_band' => $reconciliation['confidence_band'],
                'reconciliation_summary' => $reconciliation['reconciliation_summary'],
                'market_rates' => [
                    'gla_per_sf' => round($rates['gla_per_sf'], 2),
                    'bed_per_room' => round($rates['bed_per_room']),
                    'bath_per_full' => round($rates['bath_per_full']),
                    'age_per_year' => round($rates['age_per_year']),
                    'lot_per_acre' => round($rates['lot_per_acre']),
                    'garage_per_space' => round($rates['garage_per_space']),
                    'pool_value' => round($rates['pool_value']),
                    'waterfront_value' => round($rates['waterfront_value']),
                    'time_per_month_pct' => round($rates['time_per_month_pct'], 4),
                    'method' => $rates['method'],
                    'r_squared' => $rates['r_squared'],
                    'comp_count' => $rates['comp_count'],
                    'warnings' => $rates['warnings'],
                ],
                'adjustment_grid' => $gridEntries,
            ],
            'metadata' => $this->buildMetadata($validated, $fullAccess, count($soldRows), 0, 0, $processingMs, $aggMethod),
            'warnings' => $rates['warnings'] ?? [],
        ];
    }
}
