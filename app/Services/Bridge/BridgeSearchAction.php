<?php

namespace App\Services\Bridge;

use App\Enums\MlsProvider;
use App\Http\Controllers\Api\BridgeProxyController;
use App\Http\Requests\Search\SearchRequest;
use App\Http\Responses\Search\ListingResult;
use App\Http\Responses\Search\SearchResult;
use App\Http\Responses\Search\SearchStats;
use App\Services\Mls\MlsClientFactory;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Executes MLS listing search with Bridge teaser rules for domain-scoped traffic.
 * Shared by {@see BridgeProxyController::search} and widget BFF.
 */
final readonly class BridgeSearchAction
{
    public function __construct(
        private ListingsCacheService $listingsCache,
        private BridgeProxyAuditLogger $audit,
        private BridgeSearchClient $searchClient,
        private BridgeSearchTranslator $searchTranslator,
        private MlsDatasetResolver $resolver,
        private HybridSearchService $hybridSearch,
        private MlsClientFactory $mlsClients,
    ) {}

    public function __invoke(SearchRequest $request): JsonResponse
    {
        if ($this->mlsClients->providerForRequest($request) !== MlsProvider::Bridge) {
            return response()->json([
                'message' => 'Structured search is not yet available for this MLS feed.',
            ], 501);
        }

        $domainSlug = $request->attributes->get('bridge.domain_slug');
        $tokenName = $request->attributes->get('bridge.token_name');
        $userId = $request->attributes->get('bridge.user_id');

        $dataset = $this->resolver->resolveDataset($request);
        $translated = $this->searchTranslator->translate($request, $dataset);
        $bridgeTop = $translated['top'];

        $partitionKey = $this->searchCachePartitionKey($request);
        $fingerprint = $this->searchFingerprint($dataset, $request->validated());

        if ($partitionKey !== null) {
            $bridgeResult = $this->listingsCache->rememberSearchResult(
                $partitionKey,
                $fingerprint,
                fn (): array => $this->hybridSearch->fetchSearchResultPayload($request, $dataset, $translated),
            );
        } else {
            $bridgeResult = $this->hybridSearch->fetchSearchResultPayload($request, $dataset, $translated);
        }

        $results = array_map(
            fn (array $record) => $this->searchClient->mapToListingResult($record, $dataset),
            $bridgeResult['value'],
        );

        if ($translated['needsFloodZonePostFilter']) {
            $filteredRaw = $this->searchTranslator->filterLowRiskFloodZone($bridgeResult['value'], $dataset);
            $results = array_map(
                fn (array $record) => $this->searchClient->mapToListingResult($record, $dataset),
                $filteredRaw,
            );
        }

        $countAfterFilter = count($results);
        $stats = $this->computeSearchStats($results);

        $hasMore = $countAfterFilter >= $bridgeTop && $bridgeTop < 200;
        $nextSkip = $hasMore ? $translated['skip'] + $bridgeTop : null;

        $this->audit->log(
            $request,
            'search.listings',
            count($results),
            $domainSlug,
            $tokenName,
            $userId,
        );

        $searchResult = new SearchResult(
            totalCount: count($results),
            results: $results,
            hasMore: $hasMore,
            nextSkip: $nextSkip,
            stats: $stats,
        );

        return response()->json($searchResult->toArray());
    }

    private function searchCachePartitionKey(Request $request): ?string
    {
        $feed = (string) $request->attributes->get('mls.feed_code', '');
        $suffix = $feed !== '' ? ':'.$feed : '';

        $slug = $request->attributes->get('bridge.domain_slug');
        if (is_string($slug) && $slug !== '') {
            return $slug.$suffix;
        }

        $userId = $request->attributes->get('bridge.user_id');
        if ($userId !== null && (is_int($userId) || (is_string($userId) && $userId !== ''))) {
            return 'user:'.(string) $userId.$suffix;
        }

        return null;
    }

    /**
     * @param  array<string, mixed>  $validated
     */
    private function searchFingerprint(string $dataset, array $validated): string
    {
        $normalized = $validated;
        $this->ksortRecursive($normalized);

        return hash('sha256', $dataset.'|'.json_encode($normalized, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));
    }

    private function ksortRecursive(array &$array): void
    {
        ksort($array);
        foreach ($array as &$value) {
            if (is_array($value)) {
                $this->ksortRecursive($value);
            }
        }
    }

    /**
     * @param  list<ListingResult>  $results
     */
    private function computeSearchStats(array $results): ?SearchStats
    {
        if ($results === []) {
            return null;
        }

        $count = count($results);
        $domValues = [];
        $priceValues = [];

        foreach ($results as $result) {
            if ($result->daysOnMarket !== null) {
                $domValues[] = $result->daysOnMarket;
            }
            if ($result->listPrice !== null) {
                $priceValues[] = $result->listPrice;
            }
        }

        return new SearchStats(
            resultCount: $count,
            avgDom: $domValues !== [] ? array_sum($domValues) / count($domValues) : null,
            avgPrice: $priceValues !== [] ? array_sum($priceValues) / count($priceValues) : null,
            medianPrice: $this->medianPrice($priceValues),
        );
    }

    /**
     * @param  list<float>  $values
     */
    private function medianPrice(array $values): ?float
    {
        if ($values === []) {
            return null;
        }

        sort($values);
        $count = count($values);
        $mid = intdiv($count, 2);

        return $count % 2 === 0
            ? ($values[$mid - 1] + $values[$mid]) / 2
            : $values[$mid];
    }
}
