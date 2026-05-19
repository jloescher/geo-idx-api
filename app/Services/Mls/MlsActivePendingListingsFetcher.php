<?php

namespace App\Services\Mls;

use App\Enums\MlsProvider;
use App\Services\Bridge\BridgeHttpService;
use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;
use JsonException;

/**
 * Revenue impact: full Active+Pending pagination avoids cache rows being mass-deleted after a 200-item
 * partial refresh, keeping IDX pages monetizable and conversion-ready during MLS traffic spikes.
 *
 * Compliance: MLS GRID IDX + Stellar Data Access Agreement — only Active/Pending RESO slices are merged;
 * closed inventory is never synthesized here; upstream remains authoritative for historical status.
 */
final readonly class MlsActivePendingListingsFetcher
{
    public function __construct(
        private MlsClientFactory $mlsClients,
        private MlsFeedResolver $feeds,
        private BridgeHttpService $bridgeHttp,
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

        $def = $this->feeds->feedDefinition($internalFeedCode);
        if (! is_array($def) || ($def['provider'] ?? '') !== MlsProvider::STELLAR->value) {
            return ['body' => json_encode(['value' => []], JSON_THROW_ON_ERROR), 'etag' => null];
        }

        $client = $this->mlsClients->bridgeClientForFeed($internalFeedCode);

        return $this->fetchBridgePropertyMerged($client, $incoming);
    }

    /**
     * @return array{body: string, etag: ?string}
     */
    private function fetchBridgePropertyMerged(BridgeClient $client, Request $incoming): array
    {
        // Bridge standard Property OData allows $top max 200 (replication allows 2000 on /replication only).
        $pageSize = max(50, min(200, (int) config('mls.listings_sync_page_size', 200)));
        $maxPages = max(1, min(5000, (int) config('mls.listings_sync_max_pages', 500)));
        $maxRows = max(1000, min(500000, (int) config('mls.listings_sync_max_rows', 100000)));
        $since = now()->subYear()->utc()->format('Y-m-d\TH:i:s\Z');
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

        $merged = $this->mergeODataPages($incoming, $response, $maxPages, $maxRows);
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
     * @return list<array<string, mixed>>
     */
    private function mergeODataPages(Request $incoming, Response $first, int $maxPages, int $maxRows): array
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

            $response = $this->bridgeHttp->getAuthorizedJson($next, $incoming);
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
