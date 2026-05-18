<?php

namespace App\Services\Bridge;

use App\Http\Requests\Search\SearchRequest;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

/**
 * Revenue impact: routes cheap Active/Pending workloads to Postgres while reserving Bridge
 * capacity for Closed and historian queries — misroutes directly hit MLS egress spend.
 *
 * Compliance: hybrid mode must not weaken MLS display attribution; fallback preserves Bridge authoritative rows.
 */
final readonly class HybridSearchService
{
    private const SORT_FIELD_MAP = [
        'list_price' => 'ListPrice',
        'on_market_date' => 'OnMarketDate',
        'year_built' => 'YearBuilt',
        'living_area' => 'LivingArea',
        'lot_size_acres' => 'LotSizeAcres',
        'bedrooms_total' => 'BedroomsTotal',
        'bathrooms_total' => 'BathroomsTotalDecimal',
    ];

    public function __construct(
        private PostgisSearchService $postgisSearch,
        private BridgeSearchClient $bridgeSearch,
        private HybridReplicaSearchDecision $decision,
        private BridgeSearchTranslator $searchTranslator,
    ) {}

    /**
     * Returns the Bridge-shaped OData payload consumed by {@see ListingsCacheService::rememberSearchResult}.
     *
     * @param  array{filter: string, orderby: string, top: int, skip: int, select: string, unselect: string, needsFloodZonePostFilter: bool, lowRiskFloodzone: bool}  $translated
     * @return array{value: list<array<string, mixed>>, count: int, nextLink: ?string}
     */
    public function fetchSearchResultPayload(SearchRequest $request, string $dataset, array $translated): array
    {
        $validated = $request->validated();
        $mode = $this->decision->routeMode($validated);

        if ($mode === HybridSearchRouteMode::Split && DB::connection()->getDriverName() === 'pgsql') {
            return $this->fetchSplitPayload($request, $dataset, $translated, $validated);
        }

        if ($mode === HybridSearchRouteMode::PostgresOnly && DB::connection()->getDriverName() === 'pgsql') {
            try {
                $local = $this->postgisSearch->search($validated, $dataset, $translated);
                if ($this->decision->geoEmptyShouldRetryBridge($validated, $local['count'])) {
                    return $this->bridgeSearchDataset($dataset, $translated);
                }

                return $local;
            } catch (\Throwable $e) {
                Log::warning('bridge.hybrid.postgis_failed', [
                    'dataset' => $dataset,
                    'message' => $e->getMessage(),
                ]);

                return $this->bridgeSearchDataset($dataset, $translated);
            }
        }

        return $this->bridgeSearchDataset($dataset, $translated);
    }

    /**
     * @param  array{filter: string, orderby: string, top: int, skip: int, select: string, unselect: string, needsFloodZonePostFilter: bool, lowRiskFloodzone: bool}  $translated
     * @param  array<string, mixed>  $validated
     * @return array{value: list<array<string, mixed>>, count: int, nextLink: ?string}
     */
    private function fetchSplitPayload(
        SearchRequest $request,
        string $dataset,
        array $translated,
        array $validated,
    ): array {
        $pageSkip = max(0, (int) $translated['skip']);
        $pageTop = max(1, min(200, (int) $translated['top']));
        $fetchWindow = min(200, $pageSkip + $pageTop);
        $pageOverride = ['skip' => 0, 'top' => $fetchWindow];

        $replicaSlugs = $this->decision->replicaStatusesForSplit($validated);
        $validatedReplica = $this->validatedForReplicaLeg($validated, $replicaSlugs);

        $translatedReplica = $this->searchTranslator->translate(
            $request,
            $dataset,
            $replicaSlugs,
            $pageOverride,
        );

        $translatedClosed = $this->searchTranslator->translate(
            $request,
            $dataset,
            ['closed'],
            $pageOverride,
        );

        try {
            $local = $this->postgisSearch->search($validatedReplica, $dataset, $translatedReplica);
        } catch (\Throwable $e) {
            Log::warning('bridge.hybrid.split.postgis_failed', [
                'dataset' => $dataset,
                'message' => $e->getMessage(),
            ]);

            return $this->bridgeSearchDataset($dataset, $translated);
        }

        $closed = $this->bridgeSearch->search(
            dataset: $dataset,
            filter: $translatedClosed['filter'],
            orderby: $translatedClosed['orderby'],
            top: $translatedClosed['top'],
            skip: $translatedClosed['skip'],
            select: $translatedClosed['select'],
            unselect: $translatedClosed['unselect'],
        );

        $merged = $this->mergeSearchRows($local['value'], $closed['value']);
        $sorted = $this->sortMergedRows($merged, $validated, $translated['orderby'] ?? '');
        $paged = array_slice($sorted, $pageSkip, $pageTop);

        return [
            'value' => $paged,
            'count' => count($paged),
            'nextLink' => null,
        ];
    }

    /**
     * @param  array<string, mixed>  $validated
     * @param  list<string>  $replicaSlugs
     * @return array<string, mixed>
     */
    private function validatedForReplicaLeg(array $validated, array $replicaSlugs): array
    {
        $copy = $validated;
        $copy['statuses'] = array_map(
            static fn (string $slug): string => ucfirst($slug),
            $replicaSlugs,
        );
        $copy['active_only'] = false;

        return $copy;
    }

    /**
     * @param  list<array<string, mixed>>  $localRows
     * @param  list<array<string, mixed>>  $closedRows
     * @return list<array<string, mixed>>
     */
    private function mergeSearchRows(array $localRows, array $closedRows): array
    {
        $byKey = [];
        foreach (array_merge($localRows, $closedRows) as $row) {
            if (! is_array($row)) {
                continue;
            }
            $key = isset($row['ListingKey']) && is_string($row['ListingKey']) ? $row['ListingKey'] : null;
            if ($key === null || $key === '') {
                continue;
            }
            $byKey[$key] = $row;
        }

        return array_values($byKey);
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     * @param  array<string, mixed>  $validated
     * @return list<array<string, mixed>>
     */
    private function sortMergedRows(array $rows, array $validated, string $odataOrderby): array
    {
        if ($rows === []) {
            return [];
        }

        $sortKey = $validated['sort'] ?? null;
        $ascending = strtolower((string) ($validated['sort_dir'] ?? 'desc')) === 'asc';

        if ($sortKey === 'distance' && isset($validated['geo']['distance']['lat'], $validated['geo']['distance']['lng'])) {
            $lat = (float) $validated['geo']['distance']['lat'];
            $lng = (float) $validated['geo']['distance']['lng'];

            usort($rows, function (array $a, array $b) use ($lat, $lng, $ascending): int {
                $da = $this->distanceMilesFromPoint($a, $lat, $lng);
                $db = $this->distanceMilesFromPoint($b, $lat, $lng);
                if ($da === $db) {
                    return 0;
                }

                return ($da < $db ? -1 : 1) * ($ascending ? 1 : -1);
            });

            return $rows;
        }

        $field = self::SORT_FIELD_MAP[$sortKey] ?? 'ModificationTimestamp';

        usort($rows, function (array $a, array $b) use ($field, $ascending): int {
            $av = $a[$field] ?? null;
            $bv = $b[$field] ?? null;
            if ($av === $bv) {
                return 0;
            }
            if ($av === null) {
                return 1;
            }
            if ($bv === null) {
                return -1;
            }

            $cmp = $av <=> $bv;

            return $cmp * ($ascending ? 1 : -1);
        });

        if ($sortKey === null && $odataOrderby !== '' && preg_match('/^ModificationTimestamp|^BridgeModificationTimestamp/i', trim($odataOrderby))) {
            usort($rows, function (array $a, array $b): int {
                $av = $a['ModificationTimestamp'] ?? $a['BridgeModificationTimestamp'] ?? '';
                $bv = $b['ModificationTimestamp'] ?? $b['BridgeModificationTimestamp'] ?? '';

                return strcmp((string) $bv, (string) $av);
            });
        }

        return $rows;
    }

    /**
     * @param  array<string, mixed>  $row
     */
    private function distanceMilesFromPoint(array $row, float $lat, float $lng): float
    {
        $coords = $row['Coordinates'] ?? null;
        if (is_array($coords) && count($coords) >= 2) {
            $rowLng = (float) ($coords[0] ?? 0);
            $rowLat = (float) ($coords[1] ?? 0);
        } else {
            $rowLat = isset($row['Latitude']) ? (float) $row['Latitude'] : 0.0;
            $rowLng = isset($row['Longitude']) ? (float) $row['Longitude'] : 0.0;
        }

        $theta = deg2rad($lng - $rowLng);
        $dist = sin(deg2rad($lat)) * sin(deg2rad($rowLat))
            + cos(deg2rad($lat)) * cos(deg2rad($rowLat)) * cos($theta);

        return acos(max(-1.0, min(1.0, $dist))) * 69.0;
    }

    /**
     * @param  array{filter: string, orderby: string, top: int, skip: int, select: string, unselect: string, needsFloodZonePostFilter: bool, lowRiskFloodzone: bool}  $translated
     * @return array{value: list<array<string, mixed>>, count: int, nextLink: ?string}
     */
    private function bridgeSearchDataset(string $dataset, array $translated): array
    {
        return $this->bridgeSearch->search(
            dataset: $dataset,
            filter: $translated['filter'],
            orderby: $translated['orderby'],
            top: $translated['top'],
            skip: $translated['skip'],
            select: $translated['select'],
            unselect: $translated['unselect'],
        );
    }
}
