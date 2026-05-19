<?php

namespace App\Http\Controllers\Api;

use App\Enums\MlsProvider;
use App\Http\Controllers\Controller;
use App\Http\Requests\Search\SearchRequest;
use App\Services\Bridge\BridgeHttpService;
use App\Services\Bridge\BridgeImageUrlRewriter;
use App\Services\Bridge\BridgeSearchAction;
use App\Services\Bridge\ListingPriceConversionService;
use App\Services\Bridge\ListingsCacheService;
use App\Services\Mls\BridgeClient;
use App\Services\Mls\MlsActivePendingListingsFetcher;
use App\Services\Mls\MlsClientFactory;
use App\Services\Mls\MlsProxyAuditLogger;
use App\Services\Mls\SparkClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Response;
use Illuminate\Support\Facades\Http;
use Symfony\Component\HttpFoundation\Response as SymfonyResponse;

class BridgeProxyController extends Controller
{
    public function __construct(
        private readonly BridgeHttpService $bridge,
        private readonly MlsClientFactory $mlsClients,
        private readonly ListingsCacheService $listingsCache,
        private readonly MlsProxyAuditLogger $audit,
        private readonly BridgeImageUrlRewriter $imageUrlRewriter,
        private readonly ListingPriceConversionService $listingPricing,
        private readonly BridgeSearchAction $bridgeSearchAction,
        private readonly MlsActivePendingListingsFetcher $listingsActivePendingFetcher,
    ) {}

    /**
     * Revenue impact: shared search cache cuts duplicate Bridge OData spend while
     * teaser gating still applies only on the outbound JSON payload.
     */
    public function search(SearchRequest $request): JsonResponse
    {
        return ($this->bridgeSearchAction)($request);
    }

    /**
     * Revenue impact: collection caching + teaser gating reduce Bridge costs while
     * preserving fast first paint for organic SEO traffic (more leads per dollar).
     */
    public function listings(Request $request): SymfonyResponse
    {
        $feedCode = (string) $request->attributes->get('mls.feed_code');

        $domainSlug = $request->attributes->get('bridge.domain_slug');
        $useCache = is_string($domainSlug) && $domainSlug !== '' && ! $request->filled('filters');

        $fetchResponse = function () use ($request, $feedCode): \Illuminate\Http\Client\Response {
            if ($request->filled('filters')) {
                $client = $this->mlsClients->bridgeClientForFeed($feedCode);
                $url = $client->webUrl('listings');

                return $client->getJsonFromUrl($url, $request);
            }

            $payload = $this->listingsActivePendingFetcher->fetchMergedCollectionForCache($request);

            return Http::response($payload['body'], 200, array_filter([
                'Content-Type' => 'application/json',
                'ETag' => $payload['etag'],
            ]));
        };

        if ($useCache) {
            $body = $this->listingsCache->rememberListingsCollection($domainSlug, $feedCode, function () use ($fetchResponse): array {
                $response = $fetchResponse();

                return [
                    'body' => $response->body(),
                    'etag' => $response->header('ETag'),
                ];
            });
        } else {
            $response = $fetchResponse();
            if (! $response->successful()) {
                return $this->auditAndPassthrough($request, 'listings.collection', $response);
            }
            $body = $response->body();
        }

        return $this->finalizeJson($request, 'listings.collection', $body);
    }

    /**
     * Revenue impact: detail views are high-intent; serving them quickly increases
     * contact form completion versus abandonment on slow MLS round trips.
     */
    public function listing(Request $request, string $listingId): SymfonyResponse
    {
        $feedCode = (string) $request->attributes->get('mls.feed_code');
        $client = $this->mlsClients->bridgeClientForFeed($feedCode);
        $url = $client->webUrl('listings/'.$listingId);
        $response = $client->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'listings.detail', $response);
        }

        return $this->finalizeJson($request, 'listings.detail', $response->body());
    }

    public function agents(Request $request): SymfonyResponse
    {
        return $this->proxyWeb($request, 'agents.collection', 'agents');
    }

    public function agent(Request $request, string $agentId): SymfonyResponse
    {
        return $this->proxyWeb($request, 'agents.detail', 'agents/'.$agentId);
    }

    public function offices(Request $request): SymfonyResponse
    {
        return $this->proxyWeb($request, 'offices.collection', 'offices');
    }

    public function office(Request $request, string $officeId): SymfonyResponse
    {
        return $this->proxyWeb($request, 'offices.detail', 'offices/'.$officeId);
    }

    public function openHouses(Request $request): SymfonyResponse
    {
        return $this->proxyWeb($request, 'openhouses.collection', 'openhouses');
    }

    public function openHouse(Request $request, string $openhouseId): SymfonyResponse
    {
        return $this->proxyWeb($request, 'openhouses.detail', 'openhouses/'.$openhouseId);
    }

    /**
     * Revenue impact: RESO Property is the canonical listing payload for IDX compliance;
     * proxying it avoids leaking raw Bridge hosts to unapproved embedders.
     */
    public function properties(Request $request): SymfonyResponse
    {
        [$query, $stripKeys] = $this->buildResoCompatibilityQuery($request);
        $partitionKey = $this->searchCachePartitionKey($request);
        $fingerprint = $this->propertiesFingerprint($request, $query);

        $body = null;
        if ($partitionKey !== null) {
            try {
                $body = $this->listingsCache->rememberJsonPayload($partitionKey, $fingerprint, function () use ($request, $query, $stripKeys): string {
                    $response = $this->propertyCollectionResponse($request, $query, $stripKeys);
                    if (! $response->successful()) {
                        throw new \RuntimeException('MLS Property collection response failed');
                    }

                    return $response->body();
                });
            } catch (\Throwable) {
                $body = null;
            }
        } else {
            $response = $this->propertyCollectionResponse($request, $query, $stripKeys);

            if (! $response->successful()) {
                return $this->auditAndPassthrough($request, 'reso.property.collection', $response);
            }

            $body = $response->body();
        }
        if (! is_string($body) || $body === '') {
            $response = $this->propertyCollectionResponse($request, $query, $stripKeys);
            if (! $response->successful()) {
                return $this->auditAndPassthrough($request, 'reso.property.collection', $response);
            }
            $body = $response->body();
        }

        $body = $this->rewritePropertyODataLinks($body, $request);

        return $this->finalizeJson($request, 'reso.property.collection', $body);
    }

    public function property(Request $request, string $listingKey): SymfonyResponse
    {
        $response = $this->propertyEntityResponse($request, $listingKey);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.property.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.property.detail', $response->body());
    }

    public function members(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.member.collection', 'Member');
    }

    public function member(Request $request, string $memberKey): SymfonyResponse
    {
        $client = $this->resoClientFromRequest($request);
        $response = $this->proxyResoWithFallback($request, $client, $client->resoEntityUrls('Member', $memberKey));

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.member.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.member.detail', $response->body());
    }

    public function resoOffices(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.office.collection', 'Office');
    }

    public function resoOffice(Request $request, string $officeKey): SymfonyResponse
    {
        $client = $this->resoClientFromRequest($request);
        $response = $this->proxyResoWithFallback($request, $client, $client->resoEntityUrls('Office', $officeKey));

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.office.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.office.detail', $response->body());
    }

    public function resoOpenHouses(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.openhouse.collection', 'OpenHouse');
    }

    public function resoOpenHouse(Request $request, string $openHouseKey): SymfonyResponse
    {
        $client = $this->resoClientFromRequest($request);
        $response = $this->proxyResoWithFallback($request, $client, $client->resoEntityUrls('OpenHouse', $openHouseKey));

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.openhouse.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.openhouse.detail', $response->body());
    }

    public function lookup(Request $request): SymfonyResponse
    {
        $client = $this->resoClientFromRequest($request);
        $feedCode = (string) $request->attributes->get('mls.feed_code');
        $partitionKey = 'lookups:'.$feedCode;

        $queryParams = $request->query();
        unset($queryParams['domain'], $queryParams['teaser']);
        $this->ksortRecursive($queryParams);
        $fingerprint = hash('sha256', json_encode($queryParams, JSON_UNESCAPED_UNICODE));

        $body = $this->listingsCache->rememberLookups($partitionKey, $fingerprint, function () use ($request, $client): string {
            $response = $this->proxyResoWithFallback($request, $client, $client->resoCollectionUrls('Lookup'));

            if (! $response->successful()) {
                $this->audit->log(
                    $request,
                    'reso.lookup.collection',
                    null,
                    $request->attributes->get('bridge.domain_slug'),
                    $request->attributes->get('bridge.token_name'),
                    $request->attributes->get('bridge.user_id'),
                );

                abort($response->status(), $response->body());
            }

            return $response->body();
        });

        $this->audit->log(
            $request,
            'reso.lookup.collection',
            null,
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return response($body, 200, [
            'Content-Type' => 'application/json',
        ]);
    }

    public function pubParcels(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.collection', 'pub/parcels');
    }

    public function pubParcel(Request $request, string $parcelId): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.detail', 'pub/parcels/'.$parcelId);
    }

    public function pubParcelAssessments(Request $request, string $parcelId): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.assessments', 'pub/parcels/'.$parcelId.'/assessments');
    }

    public function pubParcelTransactions(Request $request, string $parcelId): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.transactions', 'pub/parcels/'.$parcelId.'/transactions');
    }

    public function pubAssessments(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.assessments.collection', 'pub/assessments');
    }

    public function pubTransactions(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.transactions.collection', 'pub/transactions');
    }

    private function proxyWeb(Request $request, string $auditType, string $path): SymfonyResponse
    {
        $client = $this->requireBridgeClient($request);
        $url = $client->webUrl($path);
        $response = $client->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, $auditType, $response);
        }

        return $this->finalizeJson($request, $auditType, $response->body());
    }

    private function proxyResoCollection(Request $request, string $auditType, string $resource): SymfonyResponse
    {
        $client = $this->resoClientFromRequest($request);
        $response = $this->proxyResoWithFallback($request, $client, $client->resoCollectionUrls($resource));

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, $auditType, $response);
        }

        return $this->finalizeJson($request, $auditType, $response->body());
    }

    /**
     * @param  list<string>  $urls
     */
    private function proxyResoWithFallback(
        Request $request,
        BridgeClient|SparkClient $client,
        array $urls,
        array $query = [],
        array $stripQueryKeys = [],
    ): \Illuminate\Http\Client\Response {
        $lastResponse = $client->getJsonFromUrl($urls[0], $request, $query, $stripQueryKeys);
        if ($lastResponse->successful()) {
            return $lastResponse;
        }

        foreach (array_slice($urls, 1) as $url) {
            if ($lastResponse->status() !== 404) {
                break;
            }

            $candidate = $client->getJsonFromUrl($url, $request, $query, $stripQueryKeys);
            if ($candidate->successful()) {
                return $candidate;
            }

            $lastResponse = $candidate;
        }

        return $lastResponse;
    }

    /**
     * Revenue impact: RESO ancillary collections share one Bridge client so token spend stays predictable.
     */
    private function requireBridgeClient(Request $request): BridgeClient
    {
        return $this->mlsClients->bridgeClientFromRequest($request);
    }

    private function resoClientFromRequest(Request $request): BridgeClient|SparkClient
    {
        if ($this->mlsClients->providerForRequest($request) === MlsProvider::SPARK) {
            return $this->mlsClients->sparkClientFromRequest($request);
        }

        return $this->mlsClients->bridgeClientFromRequest($request);
    }

    private function propertyCollectionResponse(Request $request, array $query, array $stripKeys): \Illuminate\Http\Client\Response
    {
        $client = $this->resoClientFromRequest($request);

        return $this->proxyResoWithFallback($request, $client, $client->resoCollectionUrls('Property'), $query, $stripKeys);
    }

    private function propertyEntityResponse(Request $request, string $listingKey): \Illuminate\Http\Client\Response
    {
        $client = $this->resoClientFromRequest($request);

        return $this->proxyResoWithFallback($request, $client, $client->resoEntityUrls('Property', $listingKey));
    }

    /**
     * Accept legacy convenience params on /properties and translate to OData.
     *
     * @return array{0: array<string, scalar>, 1: list<string>}
     */
    private function buildResoCompatibilityQuery(Request $request): array
    {
        $query = [];
        $stripKeys = [];

        if (! $this->hasResoInput($request, '$filter') && $this->hasResoInput($request, 'city')) {
            $city = mb_strtolower(trim((string) $this->resoInput($request, 'city')));
            if ($city !== '') {
                $escaped = str_replace("'", "''", $city);
                $query['$filter'] = "contains(tolower(City),'{$escaped}')";
                $stripKeys[] = 'city';
            }
        }

        if (! $this->hasResoInput($request, '$top') && $this->hasResoInput($request, 'limit')) {
            $limit = (int) $this->resoInput($request, 'limit');
            if ($limit > 0) {
                $query['$top'] = min(200, $limit);
                $stripKeys[] = 'limit';
            }
        }

        if (! $this->hasResoInput($request, '$next') && $this->hasResoInput($request, 'cursor')) {
            $cursor = trim((string) $this->resoInput($request, 'cursor'));
            if ($cursor !== '') {
                $query['$next'] = $cursor;
                $stripKeys[] = 'cursor';
            }
        }

        return [$query, $stripKeys];
    }

    private function hasResoInput(Request $request, string $key): bool
    {
        $value = $this->resoInput($request, $key);
        if (is_string($value)) {
            return trim($value) !== '';
        }

        return $value !== null;
    }

    private function resoInput(Request $request, string $key): mixed
    {
        if ($request->query->has($key)) {
            return $request->query($key);
        }

        return $request->input($key);
    }

    /**
     * Rewrites upstream OData links so clients paginate through idx-api while
     * preserving bearer-token auth and cacheable cursor requests.
     */
    private function rewritePropertyODataLinks(string $body, Request $request): string
    {
        try {
            $payload = json_decode($body, true, 512, JSON_THROW_ON_ERROR);
        } catch (\JsonException) {
            return $body;
        }

        if (! is_array($payload)) {
            return $body;
        }

        if (isset($payload['@odata.nextLink']) && is_string($payload['@odata.nextLink'])) {
            $next = $payload['@odata.nextLink'];
            $parsed = parse_url($next);
            parse_str((string) ($parsed['query'] ?? ''), $nextQuery);
            if (isset($nextQuery['$next']) && is_string($nextQuery['$next']) && $nextQuery['$next'] !== '') {
                $payload['@odata.nextLink'] = $request->getSchemeAndHttpHost()
                    .'/api/v1/properties?cursor='.rawurlencode($nextQuery['$next']);
            }
        }

        if (isset($payload['@odata.id']) && is_string($payload['@odata.id'])) {
            if (preg_match("/Property\\('([^']+)'\\)/", $payload['@odata.id'], $m) === 1) {
                $payload['@odata.id'] = $request->getSchemeAndHttpHost()
                    .'/api/v1/properties/'.rawurlencode($m[1]);
            }
        }

        try {
            return json_encode($payload, JSON_THROW_ON_ERROR | JSON_UNESCAPED_SLASHES);
        } catch (\JsonException) {
            return $body;
        }
    }

    /**
     * @param  array<string, scalar>  $query
     */
    private function propertiesFingerprint(Request $request, array $query): string
    {
        $payload = [
            'feed' => (string) $request->attributes->get('mls.feed_code', ''),
            'query' => $query,
        ];
        $this->ksortRecursive($payload);

        return hash('sha256', 'properties.collection|'.json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));
    }

    private function proxyDoc(Request $request, string $auditType, string $path): SymfonyResponse
    {
        $url = $this->bridge->docPath($path);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, $auditType, $response);
        }

        return $this->finalizeJson($request, $auditType, $response->body());
    }

    private function finalizeJson(Request $request, string $auditType, string $body): Response
    {
        // Revenue impact: idx-images URLs keep Cloudflare hot on one stable origin shape,
        // shrinking global TTFB and protecting Bridge credentials from browser DevTools leaks.
        try {
            $body = $this->imageUrlRewriter->rewriteJson($body);
        } catch (\Throwable) {
            // passthrough
        }

        if (in_array($auditType, ['listings.collection', 'listings.detail'], true)) {
            try {
                $body = $this->listingPricing->enrichJson($body);
            } catch (\Throwable) {
                // passthrough
            }
        }

        $listingCount = null;

        $this->audit->log(
            $request,
            $auditType,
            $listingCount,
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return response($body, 200, [
            'Content-Type' => 'application/json',
        ]);
    }

    private function auditAndPassthrough(Request $request, string $auditType, \Illuminate\Http\Client\Response $response): Response
    {
        $this->audit->log(
            $request,
            $auditType,
            null,
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return response($response->body(), $response->status(), array_filter([
            'Content-Type' => $response->header('Content-Type') ?: 'application/json',
        ]));
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

    private function ksortRecursive(array &$array): void
    {
        ksort($array);
        foreach ($array as &$value) {
            if (is_array($value)) {
                $this->ksortRecursive($value);
            }
        }
    }
}
