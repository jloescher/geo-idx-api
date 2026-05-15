<?php

namespace App\Services\Bridge;

use App\Models\Listing;
use App\Models\ListingSyncCursor;
use Carbon\CarbonImmutable;
use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

/**
 * Revenue impact: compliant Bridge replication lowers per-search MLS costs while keeping
 * idx-api maps fast; reckless polling risks API suspension — see Bridge RESO docs and
 * docs/bridge_interactive/reso_web_api.md ($top 200 / 2000, Link next, ModificationTimestamp).
 *
 * Compliance: Stellar MLS IDX / MLS Grid Rule 11 — server-token access only; display rules
 * remain enforced downstream. Replication is MLS-discretionary — handle 403 without tight retries.
 *
 * Mirror policy: indexed rows are Active + Pending only; Closed / other statuses reconcile via
 * delete batches so teaser surfaces never leak wrong lifecycle states from local cache.
 */
final class BridgeSyncService
{
    /**
     * Columns refreshed on UPSERT aside from immutable keys (PostgreSQL batches only).
     *
     * @var list<string>
     */
    private const UPSERT_UPDATE_COLUMNS = [
        'mls_listing_id',
        'standard_status',
        'list_price',
        'bedrooms_total',
        'bathrooms_total_decimal',
        'living_area',
        'lot_size_acres',
        'year_built',
        'stories_total',
        'city',
        'county_or_parish',
        'postal_code',
        'state_or_province',
        'property_type',
        'property_sub_type',
        'on_market_date',
        'close_date',
        'modification_timestamp',
        'bridge_modification_timestamp',
        'price_change_timestamp',
        'previous_list_price',
        'stellar_flood_zone_code',
        'stellar_total_monthly_fees',
        'latitude',
        'longitude',
        'waterfront_yn',
        'pool_private_yn',
        'dock_yn',
        'new_construction_yn',
        'garage_yn',
        'association_yn',
        'spa_yn',
        'fireplace_yn',
        'senior_community_yn',
        'subdivision_name',
        'elementary_school',
        'middle_or_junior_school',
        'high_school',
        'special_listing_conditions',
        'raw_data',
        'custom_fields',
        'street_number',
        'street_name',
        'list_agent_mls_id',
        'list_office_mls_id',
        'updated_at',
    ];

    public function __construct(
        private readonly BridgeHttpService $http,
    ) {}

    public function syncAllDatasets(): void
    {
        $datasets = config('bridge.datasets', ['stellar']);
        $list = is_array($datasets) ? array_values(array_filter(array_map(trim(...), $datasets))) : ['stellar'];
        foreach ($list as $dataset) {
            if (is_string($dataset) && $dataset !== '') {
                $this->syncDataset($dataset);
            }
        }
    }

    public function syncDataset(string $dataset): void
    {
        if (DB::connection()->getDriverName() !== 'pgsql') {
            return;
        }

        /** @var ListingSyncCursor $cursor */
        $cursor = ListingSyncCursor::query()->firstOrCreate(
            ['dataset_slug' => $dataset],
            ['replication_in_progress' => false],
        );

        $replicationPages = 0;
        $maxRep = (int) config('bridge.sync_max_replication_pages_per_job', 12);

        while ($replicationPages < $maxRep && $this->shouldContinueReplication($cursor)) {
            $done = $this->runReplicationPage($dataset, $cursor);
            $replicationPages++;
            if ($done) {
                break;
            }
        }

        if ($cursor->replication_in_progress && $cursor->replication_next_url === null) {
            $cursor->replication_in_progress = false;
            $cursor->save();
        }

        if (! $cursor->replication_in_progress) {
            $this->runIncrementalSync($dataset, $cursor);
        }

        $cursor->last_sync_finished_at = now();
        $cursor->save();
    }

    private function shouldContinueReplication(ListingSyncCursor $cursor): bool
    {
        if ($cursor->replication_next_url !== null && $cursor->replication_next_url !== '') {
            $cursor->replication_in_progress = true;
            $cursor->save();

            return true;
        }

        if ($cursor->replication_in_progress) {
            return true;
        }

        $count = Listing::query()->where('dataset_slug', $cursor->dataset_slug)->count();

        return $count === 0;
    }

    /**
     * @return bool True when replication segment is complete (no more pages this run).
     */
    private function runReplicationPage(string $dataset, ListingSyncCursor $cursor): bool
    {
        $select = $this->syncSelectList($dataset);
        $top = (int) config('bridge.sync_replication_top', 2000);

        if ($cursor->replication_next_url !== null && $cursor->replication_next_url !== '') {
            $url = $cursor->replication_next_url;
            $query = [];
        } else {
            $url = $this->buildPropertyReplicationUrl($dataset);
            $query = [
                '$top' => $top,
                '$select' => $select,
            ];
            $cursor->replication_in_progress = true;
            $cursor->save();
        }

        $response = $this->http->serverJsonGet($url, $query);

        if ($response->status() === 403) {
            Log::warning('bridge.replication.forbidden', ['dataset' => $dataset]);
            $cursor->replication_in_progress = false;
            $cursor->replication_next_url = null;
            $cursor->save();

            return true;
        }

        if (! $response->successful()) {
            Log::warning('bridge.replication.http_error', [
                'dataset' => $dataset,
                'status' => $response->status(),
            ]);

            return true;
        }

        $body = $response->json();
        $value = is_array($body['value'] ?? null) ? $body['value'] : [];
        $maxBridgeTs = $this->hydrateReplicaBatch($dataset, $value);
        $this->advanceCursorTimestamp($cursor, $maxBridgeTs);

        $next = $this->extractNextUrl($response);
        $cursor->replication_next_url = $next;
        if ($next === null) {
            $cursor->replication_in_progress = false;
        }
        $cursor->save();

        return $next === null;
    }

    private function runIncrementalSync(string $dataset, ListingSyncCursor $cursor): void
    {
        if ($cursor->last_bridge_modification_timestamp === null) {
            return;
        }

        $top = (int) config('bridge.sync_incremental_top', 200);
        $maxPages = (int) config('bridge.sync_max_incremental_pages_per_job', 40);
        $select = $this->syncSelectList($dataset);

        $cursorTs = $cursor->last_bridge_modification_timestamp;
        $filterTs = CarbonImmutable::parse($cursorTs->format(\DateTimeInterface::ATOM));

        $filterLiteral = "ModificationTimestamp gt datetime'".$filterTs->utc()->format('Y-m-d\TH:i:s\Z')."'";
        $orderby = 'ModificationTimestamp asc';

        $url = $this->buildPropertyCollectionUrl($dataset);
        $pages = 0;
        $skip = 0;

        while ($pages < $maxPages) {
            $query = [
                '$filter' => $filterLiteral,
                '$orderby' => $orderby,
                '$top' => $top,
                '$skip' => $skip,
                '$select' => $select,
            ];

            $response = $this->http->serverJsonGet($url, $query);

            if ($response->status() === 400 || $response->status() === 501) {
                Log::info('bridge.incremental.fallback_bridge_ts', ['dataset' => $dataset]);
                $filterLiteral = "BridgeModificationTimestamp gt datetime'".$filterTs->utc()->format('Y-m-d\TH:i:s\Z')."'";
                $orderby = 'BridgeModificationTimestamp asc';
                $query['$filter'] = $filterLiteral;
                $query['$orderby'] = $orderby;
                $response = $this->http->serverJsonGet($url, $query);
            }

            if (! $response->successful()) {
                Log::warning('bridge.incremental.http_error', [
                    'dataset' => $dataset,
                    'status' => $response->status(),
                ]);
                break;
            }

            $body = $response->json();
            $value = is_array($body['value'] ?? null) ? $body['value'] : [];
            if ($value === []) {
                break;
            }

            $maxBridgeTs = $this->hydrateReplicaBatch($dataset, $value);
            $this->advanceCursorTimestamp($cursor, $maxBridgeTs);
            $cursor->save();

            $pages++;
            if (count($value) < $top) {
                break;
            }

            $skip += $top;
            if ($skip >= 10_000) {
                Log::warning('bridge.incremental.skip_cap', [
                    'dataset' => $dataset,
                    'message' => 'Exceeded 10k skip; use replication for catch-up per Bridge docs.',
                ]);
                break;
            }
        }
    }

    /**
     * Revenue impact: batch upserts amortize Postgres round trips; per-row deletes for non-Active/Pending rows
     * keep IDX mirror aligned with teaser scope so local speed never violates lifecycle display rules.
     *
     * @param  list<array<string, mixed>>  $rows
     * @return ?CarbonImmutable High-water timestamp from MLS fields for ReplicationTimestamp cursor bookkeeping.
     */
    private function hydrateReplicaBatch(string $dataset, array $rows): ?CarbonImmutable
    {
        $maxTs = null;
        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }
            $maxTs = $this->maxBridgeTimestamp($maxTs, $row);
        }

        $chunkSize = (int) config('bridge.sync_upsert_chunk_size', 250);
        $pendingUpserts = [];
        $coords = [];
        /** @var array<string, array<string, bool>> $deletesGrouped */
        $deletesGrouped = [];
        /** @var array<string, array<string, bool>> $nullCoordKeys */
        $nullCoordKeys = [];

        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }

            $listingKey = isset($row['ListingKey']) && is_string($row['ListingKey']) ? $row['ListingKey'] : null;
            if ($listingKey === null || $listingKey === '') {
                continue;
            }

            $datasetSlug = $this->datasetSlugFromListingKey($listingKey, $dataset);

            $statusNormalized = strtolower(trim((string) ($row['StandardStatus'] ?? '')));
            if (! in_array($statusNormalized, ['active', 'pending'], true)) {
                if (! isset($deletesGrouped[$datasetSlug])) {
                    $deletesGrouped[$datasetSlug] = [];
                }
                $deletesGrouped[$datasetSlug][$listingKey] = true;

                continue;
            }

            $coordsPair = $this->resolveLanLng($row);
            $lat = $coordsPair['lat'];
            $lng = $coordsPair['lng'];

            $payload = $this->buildUpsertPayload($dataset, $datasetSlug, $listingKey, $row, $lat, $lng);
            $pendingUpserts[] = $payload;

            if ($lng !== null && $lat !== null && is_finite($lat) && is_finite($lng)) {
                $coords[] = [$datasetSlug, $listingKey, $lng, $lat];
            } else {
                if (! isset($nullCoordKeys[$datasetSlug])) {
                    $nullCoordKeys[$datasetSlug] = [];
                }
                $nullCoordKeys[$datasetSlug][$listingKey] = true;
            }

            if (count($pendingUpserts) >= $chunkSize) {
                $this->flushUpsertChunk($pendingUpserts);
                $pendingUpserts = [];
                $this->flushCoordinates($coords);
                $coords = [];
            }
        }

        if ($pendingUpserts !== []) {
            $this->flushUpsertChunk($pendingUpserts);
        }
        $this->flushCoordinates($coords);
        $this->flushNullCoordinates($nullCoordKeys);
        $this->flushDeletes($deletesGrouped);

        return $maxTs;
    }

    /**
     * @param  list<array<string, mixed>>  $records
     */
    private function flushUpsertChunk(array $records): void
    {
        if ($records === []) {
            return;
        }

        Listing::withoutTimestamps(function () use ($records): void {
            Listing::query()->upsert(
                $records,
                ['dataset_slug', 'listing_key'],
                self::UPSERT_UPDATE_COLUMNS,
            );
        });
    }

    /**
     * @param  list<array{0: string, 1: string, 2: float, 3: float}>  $pairs
     */
    private function flushCoordinates(array $pairs): void
    {
        if ($pairs === []) {
            return;
        }

        foreach (array_chunk($pairs, (int) config('bridge.sync_upsert_chunk_size', 250)) as $segment) {
            $sqlParts = [];
            $bindings = [];
            foreach ($segment as [$ds, $key, $lng, $lat]) {
                $sqlParts[] = '(?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)';
                array_push($bindings, $ds, $key, $lng, $lat);
            }
            $valuesSql = implode(',', $sqlParts);
            $updatedAt = now();
            DB::statement(
                "UPDATE listings AS l SET coordinates = v.geom, updated_at = ? FROM (VALUES {$valuesSql}) AS v(ds,k,geom) WHERE l.dataset_slug = v.ds AND l.listing_key = v.k",
                array_merge([$updatedAt], $bindings),
            );
        }
    }

    /**
     * @param  array<string, array<string, bool>>  $grouped
     */
    private function flushNullCoordinates(array $grouped): void
    {
        if ($grouped === []) {
            return;
        }

        $chunkSize = (int) config('bridge.sync_upsert_chunk_size', 250);
        foreach ($grouped as $datasetSlug => $keysMap) {
            $keys = array_keys($keysMap);
            foreach (array_chunk($keys, $chunkSize) as $segment) {
                Listing::query()
                    ->where('dataset_slug', $datasetSlug)
                    ->whereIn('listing_key', $segment)
                    ->update([
                        'coordinates' => null,
                        'updated_at' => now(),
                    ]);
            }
        }
    }

    /**
     * @param  array<string, array<string, bool>>  $groupedNormalized  dataset_slug → listing_key map
     */
    private function flushDeletes(array $groupedNormalized): void
    {
        if ($groupedNormalized === []) {
            return;
        }

        foreach ($groupedNormalized as $datasetSlug => $keysMap) {
            $keys = array_keys($keysMap);
            foreach (array_chunk($keys, (int) config('bridge.sync_upsert_chunk_size', 250)) as $segment) {
                Listing::query()
                    ->where('dataset_slug', $datasetSlug)
                    ->whereIn('listing_key', $segment)
                    ->delete();
            }
        }
    }

    /**
     * @param  array<string, mixed>  $row
     * @return array<string, mixed>
     */
    private function buildUpsertPayload(
        string $datasetUpper,
        string $datasetSlug,
        string $listingKey,
        array $row,
        ?float $lat,
        ?float $lng,
    ): array {
        $u = strtoupper($datasetUpper);
        $custom = $this->extractExtensionFields($row, $u);

        $extFloodField = "{$u}_FloodZoneCode";
        $extFeesField = "{$u}_TotalMonthlyFees";
        $extFlood = $row[$extFloodField] ?? null;
        $extFees = $row[$extFeesField] ?? null;

        $conditions = null;
        if (isset($row['SpecialListingConditions']) && is_array($row['SpecialListingConditions'])) {
            $conditions = array_values(array_filter(
                array_map(static fn ($c) => is_string($c) ? trim($c) : null, $row['SpecialListingConditions']),
                static fn (?string $x) => $x !== null && $x !== '',
            ));
        }

        $now = now();

        return [
            'dataset_slug' => $datasetSlug,
            'listing_key' => $listingKey,
            'mls_listing_id' => is_string($row['ListingId'] ?? null) ? $row['ListingId'] : (is_numeric($row['ListingId'] ?? null) ? (string) $row['ListingId'] : null),
            'standard_status' => is_string($row['StandardStatus'] ?? null) ? $row['StandardStatus'] : null,
            'list_price' => $this->num($row['ListPrice'] ?? null),
            'bedrooms_total' => $this->intOrNull($row['BedroomsTotal'] ?? null),
            'bathrooms_total_decimal' => $this->num($row['BathroomsTotalDecimal'] ?? null),
            'living_area' => $this->intOrNull($row['LivingArea'] ?? null),
            'lot_size_acres' => $this->num($row['LotSizeAcres'] ?? null),
            'year_built' => $this->intOrNull($row['YearBuilt'] ?? null),
            'stories_total' => $this->intOrNull($row['StoriesTotal'] ?? null),
            'city' => $this->str($row['City'] ?? null),
            'county_or_parish' => $this->str($row['CountyOrParish'] ?? null),
            'postal_code' => $this->str($row['PostalCode'] ?? null),
            'state_or_province' => $this->str($row['StateOrProvince'] ?? null),
            'property_type' => $this->str($row['PropertyType'] ?? null),
            'property_sub_type' => $this->str($row['PropertySubType'] ?? null),
            'on_market_date' => $this->dateStr($row['OnMarketDate'] ?? null),
            'close_date' => $this->dateStr($row['CloseDate'] ?? null),
            'modification_timestamp' => $this->toSqlTimestamp($row['ModificationTimestamp'] ?? null),
            'bridge_modification_timestamp' => $this->toSqlTimestamp($row['BridgeModificationTimestamp'] ?? null),
            'price_change_timestamp' => $this->toSqlTimestamp($row['PriceChangeTimestamp'] ?? null),
            'previous_list_price' => $this->num($row['PreviousListPrice'] ?? null),
            'stellar_flood_zone_code' => is_string($extFlood) ? $extFlood : null,
            'stellar_total_monthly_fees' => $this->num($extFees),
            'latitude' => $lat,
            'longitude' => $lng,
            'waterfront_yn' => $this->boolOrNull($row['WaterfrontYN'] ?? null),
            'pool_private_yn' => $this->boolOrNull($row['PoolPrivateYN'] ?? null),
            'dock_yn' => $this->boolOrNull($row['DockYN'] ?? null),
            'new_construction_yn' => $this->boolOrNull($row['NewConstructionYN'] ?? null),
            'garage_yn' => $this->boolOrNull($row['GarageYN'] ?? null),
            'association_yn' => $this->boolOrNull($row['AssociationYN'] ?? null),
            'spa_yn' => $this->boolOrNull($row['SpaYN'] ?? null),
            'fireplace_yn' => $this->boolOrNull($row['FireplaceYN'] ?? null),
            'senior_community_yn' => $this->boolOrNull($row['SeniorCommunityYN'] ?? null),
            'subdivision_name' => $this->str($row['SubdivisionName'] ?? null),
            'elementary_school' => $this->str($row['ElementarySchool'] ?? null),
            'middle_or_junior_school' => $this->str($row['MiddleOrJuniorSchool'] ?? null),
            'high_school' => $this->str($row['HighSchool'] ?? null),
            'special_listing_conditions' => $conditions ?? [],
            'raw_data' => $row,
            'custom_fields' => $custom,
            'street_number' => $this->str($row['StreetNumber'] ?? null),
            'street_name' => $this->str($row['StreetName'] ?? null),
            'list_agent_mls_id' => $this->str($row['ListAgentMlsId'] ?? null),
            'list_office_mls_id' => $this->str($row['ListOfficeMlsId'] ?? null),
            'created_at' => $now,
            'updated_at' => $now,
        ];
    }

    private function toSqlTimestamp(mixed $v): ?string
    {
        if (! is_string($v) || trim($v) === '') {
            return null;
        }
        try {
            return CarbonImmutable::parse($v)->utc()->format('Y-m-d H:i:s.u');
        } catch (\Throwable) {
            return null;
        }
    }

    /**
     * @param  array<string, mixed>  $row
     * @return array{lat: ?float, lng: ?float}
     */
    private function resolveLanLng(array $row): array
    {
        $lat = null;
        $lng = null;
        if (isset($row['Coordinates']) && is_array($row['Coordinates'])) {
            $pair = $row['Coordinates']['coordinates'] ?? null;
            if (is_array($pair) && count($pair) >= 2) {
                $lng = is_numeric($pair[0]) ? (float) $pair[0] : null;
                $lat = is_numeric($pair[1]) ? (float) $pair[1] : null;
            }
        }
        if (($lat === null || $lng === null) && isset($row['Latitude'], $row['Longitude'])) {
            $lat = is_numeric($row['Latitude']) ? (float) $row['Latitude'] : $lat;
            $lng = is_numeric($row['Longitude']) ? (float) $row['Longitude'] : $lng;
        }

        return ['lat' => $lat, 'lng' => $lng];
    }

    private function datasetSlugFromListingKey(string $listingKey, string $fallbackDataset): string
    {
        $idx = strpos($listingKey, ':');
        if ($idx !== false && $idx > 0) {
            return strtolower(substr($listingKey, 0, $idx));
        }

        return strtolower($fallbackDataset);
    }

    /**
     * @return array<string, mixed>
     */
    private function extractExtensionFields(array $record, string $datasetUpper): array
    {
        $prefix = $datasetUpper.'_';
        $out = [];
        foreach ($record as $k => $v) {
            if (is_string($k) && str_starts_with(strtoupper($k), $prefix)) {
                $out[$k] = $v;
            }
        }

        return $out;
    }

    private function maxBridgeTimestamp(?CarbonImmutable $current, array $row): ?CarbonImmutable
    {
        $candidates = [];
        foreach (['ModificationTimestamp', 'BridgeModificationTimestamp'] as $key) {
            $raw = $row[$key] ?? null;
            if (! is_string($raw) || $raw === '') {
                continue;
            }
            try {
                $candidates[] = CarbonImmutable::parse($raw);
            } catch (\Throwable) {
                // ignore
            }
        }
        if ($candidates === []) {
            return $current;
        }
        $max = $candidates[0];
        foreach ($candidates as $c) {
            if ($c->gt($max)) {
                $max = $c;
            }
        }

        if ($current === null || $max->gt($current)) {
            return $max;
        }

        return $current;
    }

    private function advanceCursorTimestamp(ListingSyncCursor $cursor, ?CarbonImmutable $maxBridgeTs): void
    {
        if ($maxBridgeTs === null) {
            return;
        }
        $existing = $cursor->last_bridge_modification_timestamp;
        if (! $existing instanceof \DateTimeInterface || CarbonImmutable::parse($existing->format(\DateTimeInterface::ATOM))->lt($maxBridgeTs)) {
            $cursor->last_bridge_modification_timestamp = $maxBridgeTs;
        }
    }

    private function extractNextUrl(Response $response): ?string
    {
        $link = $response->header('Link');
        if (is_string($link) && $link !== '') {
            foreach (explode(',', $link) as $segment) {
                if (preg_match('/<([^>]+)>\s*;\s*rel\s*=\s*"next"/i', trim($segment), $m)) {
                    return html_entity_decode($m[1], ENT_QUOTES);
                }
                if (preg_match("/<([^>]+)>\s*;\s*rel\s*=\s*'next'/i", trim($segment), $m)) {
                    return html_entity_decode($m[1], ENT_QUOTES);
                }
            }
        }

        $json = $response->json();
        if (is_array($json) && isset($json['@odata.nextLink']) && is_string($json['@odata.nextLink'])) {
            return $json['@odata.nextLink'];
        }

        return null;
    }

    private function buildPropertyCollectionUrl(string $dataset): string
    {
        return $this->buildODataPropertyBase($dataset).'/Property';
    }

    /**
     * Property replication feed (MLS-discretionary).
     */
    private function buildPropertyReplicationUrl(string $dataset): string
    {
        return $this->buildODataPropertyBase($dataset).'/Property/replication';
    }

    private function buildODataPropertyBase(string $dataset): string
    {
        $host = rtrim((string) config('bridge.host'), '/');
        $prefix = trim((string) config('bridge.path_prefix', ''), '/');
        $resoRoot = trim((string) config('bridge.reso_root', ''), '/');

        $basePath = $prefix !== ''
            ? "{$prefix}/OData/{$dataset}"
            : ($resoRoot !== ''
                ? "{$resoRoot}/OData/{$dataset}"
                : "OData/{$dataset}");

        return "{$host}/{$basePath}";
    }

    private function syncSelectList(string $dataset): string
    {
        $datasetUpper = strtoupper($dataset);

        return implode(',', array_unique(array_merge([
            'ListingKey',
            'ListingId',
            'BridgeModificationTimestamp',
            'ModificationTimestamp',
            'StandardStatus',
            'ListPrice',
            'ClosePrice',
            'PreviousListPrice',
            'PriceChangeTimestamp',
            'BedroomsTotal',
            'BathroomsTotalDecimal',
            'LivingArea',
            'LotSizeAcres',
            'YearBuilt',
            'StoriesTotal',
            'City',
            'CountyOrParish',
            'PostalCode',
            'StateOrProvince',
            'PropertyType',
            'PropertySubType',
            'OnMarketDate',
            'CloseDate',
            'Latitude',
            'Longitude',
            'Coordinates',
            'WaterfrontYN',
            'PoolPrivateYN',
            'DockYN',
            'NewConstructionYN',
            'GarageYN',
            'AssociationYN',
            'SpaYN',
            'FireplaceYN',
            'SeniorCommunityYN',
            'SpecialListingConditions',
            'SubdivisionName',
            'ElementarySchool',
            'MiddleOrJuniorSchool',
            'HighSchool',
            'StreetNumber',
            'StreetName',
            'ListAgentMlsId',
            'ListOfficeMlsId',
            'Media',
        ], [
            "{$datasetUpper}_FloodZoneCode",
            "{$datasetUpper}_TotalMonthlyFees",
        ])));
    }

    private function str(mixed $v): ?string
    {
        if (! is_string($v)) {
            return null;
        }
        $t = trim($v);

        return $t === '' ? null : $t;
    }

    private function num(mixed $v): ?float
    {
        return is_numeric($v) ? (float) $v : null;
    }

    private function intOrNull(mixed $v): ?int
    {
        return is_numeric($v) ? (int) $v : null;
    }

    private function boolOrNull(mixed $v): ?bool
    {
        return is_bool($v) ? $v : null;
    }

    private function dateStr(mixed $v): ?string
    {
        if (! is_string($v) || trim($v) === '') {
            return null;
        }
        try {
            return CarbonImmutable::parse($v)->format('Y-m-d');
        } catch (\Throwable) {
            return null;
        }
    }
}
