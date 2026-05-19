<?php

namespace App\Services\Mls;

use App\Enums\MlsProvider;
use App\Services\Bridge\BridgeHttpService;
use App\Services\Spark\SparkHttpService;
use Closure;
use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;
use JsonException;

/**
 * Revenue impact: full Active+Pending pagination avoids cache rows being mass-deleted after a partial
 * refresh, keeping IDX pages monetizable and conversion-ready during MLS traffic spikes.
 *
 * Compliance: MLS GRID IDX — only Active/Pending RESO slices are merged; closed inventory is never
 * synthesized here; upstream live API remains authoritative for historical status.
 */
final readonly class MlsActivePendingListingsFetcher
{
    public function __construct(
        private MlsClientFactory $mlsClients,
        private MlsFeedResolver $feeds,
        private BridgeHttpService $bridgeHttp,
        private SparkHttpService $sparkHttp,
        private MlsMirrorRollingWindow $rollingWindow,
    ) {}

    /**
     * @return array{body: string, etag: ?string}
     */
    public function fetchMergedCollectionForCache(Request $incoming): array
    {
        $internalFeedCode = (string) $incoming->attributes->get('mls.feed_code', '');
        if ($internalFeedCode === '') {
            $internalFeedCode = $this->feeds->resolveFeedCode($incoming);
        }

        $catalogKey = $this->feeds->normalizeWireDatasetToCatalogKey($internalFeedCode);
        $def = $this->feeds->feedDefinition($catalogKey);
        if (! is_array($def)) {
            return ['body' => json_encode(['value' => []], JSON_THROW_ON_ERROR), 'etag' => null];
        }

        $provider = (string) ($def['provider'] ?? '');

        if ($provider === MlsProvider::STELLAR->value) {
            $client = $this->mlsClients->bridgeClientForFeed($internalFeedCode);

            return $this->fetchBridgePropertyMerged($client, $incoming);
        }

        if ($provider === MlsProvider::SPARK->value) {
            $client = $this->mlsClients->sparkClientForFeed($internalFeedCode);

            return $this->fetchSparkPropertyMerged($client, $incoming);
        }

        return ['body' => json_encode(['value' => []], JSON_THROW_ON_ERROR), 'etag' => null];
    }

    /**
     * @return array{body: string, etag: ?string}
     */
    private function fetchBridgePropertyMerged(BridgeClient $client, Request $incoming): array
    {
        $pageSize = max(50, min(200, (int) config('mls.listings_sync_page_size', 200)));
        $maxPages = max(1, min(5000, (int) config('mls.listings_sync_max_pages', 500)));
        $maxRows = max(1000, min(500000, (int) config('mls.listings_sync_max_rows', 100000)));
        $since = $this->rollingWindow->modificationTimestampFilterIso();
        $filter = "(StandardStatus eq 'Active' or StandardStatus eq 'Pending') and ModificationTimestamp ge datetime'{$since}'";
        $query = [
            '$filter' => $filter,
            '$top' => $pageSize,
        ];

        $urls = $client->resoCollectionUrls('Property');
        $response = $this->firstSuccessfulBridgePropertyResponse($client, $incoming, $urls, $query);
        if (! $response->successful()) {
            return ['body' => $response->body(), 'etag' => $response->header('ETag')];
        }

        $merged = $this->mergeODataPages(
            $response,
            $maxPages,
            $maxRows,
            fn (string $url): Response => $this->bridgeHttp->getAuthorizedJson($url, $incoming),
        );
        $etag = $response->header('ETag');

        return [
            'body' => json_encode(['value' => $merged], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
            'etag' => is_string($etag) && $etag !== '' ? $etag : null,
        ];
    }

    /**
     * @return array{body: string, etag: ?string}
     */
    private function fetchSparkPropertyMerged(SparkClient $client, Request $incoming): array
    {
        $pageSize = max(50, min(1000, (int) config('mls.listings_sync_page_size', 200)));
        $maxPages = max(1, min(5000, (int) config('mls.listings_sync_max_pages', 500)));
        $maxRows = max(1000, min(500000, (int) config('mls.listings_sync_max_rows', 100000)));
        $since = $this->rollingWindow->modificationTimestampFilterIso();
        $filter = "(StandardStatus eq 'Active' or StandardStatus eq 'Pending') and ModificationTimestamp ge datetime'{$since}'";
        $query = [
            '$filter' => $filter,
            '$top' => $pageSize,
        ];

        $url = $client->propertyCollectionUrl();
        $response = $client->getJsonFromUrl($url, $incoming, $query, ['limit', 'domain', 'teaser']);
        if (! $response->successful()) {
            return ['body' => $response->body(), 'etag' => $response->header('ETag')];
        }

        $merged = $this->mergeODataPages(
            $response,
            $maxPages,
            $maxRows,
            fn (string $nextUrl): Response => $this->sparkHttp->getAuthorizedJson($nextUrl, $incoming),
        );
        $etag = $response->header('ETag');

        return [
            'body' => json_encode(['value' => $merged], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
            'etag' => is_string($etag) && $etag !== '' ? $etag : null,
        ];
    }

    /**
     * @param  list<string>  $urls
     */
    private function firstSuccessfulBridgePropertyResponse(BridgeClient $client, Request $incoming, array $urls, array $query): Response
    {
        $last = $client->getJsonFromUrl($urls[0], $incoming, $query, ['limit']);
        if ($last->successful()) {
            return $last;
        }

        foreach (array_slice($urls, 1) as $url) {
            if ($last->status() !== 404) {
                break;
            }

            $candidate = $client->getJsonFromUrl($url, $incoming, $query, ['limit']);
            if ($candidate->successful()) {
                return $candidate;
            }

            $last = $candidate;
        }

        return $last;
    }

    /**
     * @param  Closure(string): Response  $fetchNextPage
     * @return list<array<string, mixed>>
     */
    private function mergeODataPages(Response $first, int $maxPages, int $maxRows, Closure $fetchNextPage): array
    {
        $merged = [];
        $response = $first;
        $pages = 0;

        while (true) {
            $pages++;
            $chunk = $this->extractValueRows($response->body());
            foreach ($chunk as $row) {
                $merged[] = $row;
                if (count($merged) >= $maxRows) {
                    return $merged;
                }
            }

            if ($pages >= $maxPages) {
                break;
            }

            $next = $this->extractNextLink($response->body());
            if (! is_string($next) || $next === '') {
                break;
            }

            $response = $fetchNextPage($next);
            if (! $response->successful()) {
                break;
            }
        }

        return $merged;
    }

    /**
     * @return list<array<string, mixed>>
     */
    private function extractValueRows(string $body): array
    {
        try {
            $data = json_decode($body, true, 512, JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            return [];
        }

        if (! is_array($data)) {
            return [];
        }

        $value = $data['value'] ?? null;

        return is_array($value) ? $value : [];
    }

    private function extractNextLink(string $body): ?string
    {
        try {
            $data = json_decode($body, true, 512, JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            return null;
        }

        if (! is_array($data)) {
            return null;
        }

        foreach (['@odata.nextLink', 'odata.nextLink', 'nextLink'] as $key) {
            if (isset($data[$key]) && is_string($data[$key]) && trim($data[$key]) !== '') {
                return trim($data[$key]);
            }
        }

        return null;
    }
}
