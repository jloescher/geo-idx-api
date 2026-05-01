<?php

namespace App\Services\Bridge;

use App\Http\Requests\Search\SearchRequest;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

/**
 * Revenue impact: routes cheap Active/Pending workloads to Postgres while reserving Bridge
 * capacity for historian/comps-heavy queries — misroutes directly hit MLS egress spend.
 *
 * Compliance: hybrid mode must not weaken MLS display attribution; fallback preserves Bridge authoritative rows.
 */
final readonly class HybridSearchService
{
    public function __construct(
        private PostgisSearchService $postgisSearch,
        private BridgeSearchClient $bridgeSearch,
        private HybridReplicaSearchDecision $decision,
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

        if ($this->decision->prefersPostgresReplica($validated) && DB::connection()->getDriverName() === 'pgsql') {
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
