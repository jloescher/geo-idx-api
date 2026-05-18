<?php

namespace App\Services\Bridge;

use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Http;

class BridgeHttpService
{
    public function __construct(
        private readonly BridgeRateLimitGuard $rateLimitGuard,
    ) {}

    /**
     * Revenue impact: server-driven replication/sync must not depend on an inbound web request;
     * Bearer token stays server-side per Stellar MLS IDX / MLS GRID Rule 11 server-to-server posture.
     *
     * @param  array<string, scalar|list<string>>  $query
     */
    public function serverJsonGet(string $fullUrl, array $query = []): Response
    {
        return $this->rateLimitedRequest(function () use ($fullUrl, $query): Response {
            $token = $this->resolveBearerToken();
            if ($token === null) {
                abort(503, 'Bridge server token is not configured.');
            }

            $timeout = (int) config('bridge.timeout_seconds');

            return BridgeUpstreamRetry::run(fn () => Http::timeout($timeout)
                ->withToken($token)
                ->acceptJson()
                ->get($fullUrl, $query));
        });
    }

    /**
     * Revenue impact: a single HTTP wrapper keeps Bridge credentials off edge devices
     * and centralizes timeouts so slow MLS responses do not tie up checkout flows.
     */
    public function getJsonFromUrl(string $url, Request $incoming, array $query = [], array $stripQueryKeys = []): Response
    {
        $token = $this->resolveBearerToken();
        if ($token === null) {
            abort(503, 'Bridge server token is not configured.');
        }

        $mergedQuery = array_merge($incoming->query(), $query);
        foreach (array_merge(['domain', 'teaser'], $stripQueryKeys) as $internal) {
            unset($mergedQuery[$internal]);
        }

        $timeout = (int) config('bridge.timeout_seconds');

        return $this->rateLimitedRequest(function () use ($incoming, $url, $token, $mergedQuery, $timeout): Response {
            return BridgeUpstreamRetry::run(fn () => Http::timeout($timeout)
                ->withHeaders($this->forwardClientHeaders($incoming))
                ->withToken($token)
                ->acceptJson()
                ->get($url, $mergedQuery));
        });
    }

    /**
     * GET a fully qualified URL (e.g. OData @odata.nextLink) with Bridge auth — no query merge from the request.
     *
     * Revenue impact: cursor-chained Property pulls complete Active+Pending sets so row cache eviction matches MLS reality.
     *
     * Compliance: Stellar MLS / MLS GRID IDX — Bearer token remains server-side; nextLink URLs are upstream-controlled.
     */
    public function getAuthorizedJson(string $url, Request $incoming): Response
    {
        $token = $this->resolveBearerToken();
        if ($token === null) {
            abort(503, 'Bridge server token is not configured.');
        }

        $timeout = (int) config('bridge.timeout_seconds');

        return $this->rateLimitedRequest(function () use ($incoming, $url, $token, $timeout): Response {
            return BridgeUpstreamRetry::run(fn () => Http::timeout($timeout)
                ->withHeaders($this->forwardClientHeaders($incoming))
                ->withToken($token)
                ->acceptJson()
                ->get($url));
        });
    }

    /**
     * Bridge Web API paths documented as /{dataset}/listings, /{dataset}/agents, etc.
     */
    public function webUrl(string $resourcePath, ?string $datasetOverride = null): string
    {
        $resourcePath = ltrim($resourcePath, '/');

        return $this->joinUrl($this->datasetPathPrefix($datasetOverride).'/'.$resourcePath);
    }

    /**
     * RESO resources (Property, Member, Office, OpenHouse, Lookup).
     */
    public function resoCollectionUrl(string $resource, ?string $datasetOverride = null): string
    {
        return $this->resoCollectionUrls($resource, $datasetOverride)[0];
    }

    /**
     * @return list<string>
     */
    public function resoCollectionUrls(string $resource, ?string $datasetOverride = null): array
    {
        $resource = ltrim($resource, '/');
        $dataset = $this->resolveDataset($datasetOverride);
        $prefix = trim((string) config('bridge.path_prefix', ''), '/');
        $resoRoot = trim((string) config('bridge.reso_root', ''), '/');
        $paths = [];

        if ($prefix !== '') {
            if ($resoRoot !== '') {
                $paths[] = $prefix.'/'.$resoRoot.'/'.$dataset.'/'.$resource;
            }

            $paths[] = $prefix.'/OData/'.$dataset.'/'.$resource;
        }

        if ($resoRoot !== '') {
            $paths[] = $resoRoot.'/'.$dataset.'/'.$resource;
        }

        $paths[] = $dataset.'/'.$resource;

        return array_values(array_unique(array_map(
            fn (string $path): string => $this->joinUrl($path),
            $paths
        )));
    }

    public function resoEntityUrl(string $resource, string $key, ?string $datasetOverride = null): string
    {
        return $this->resoEntityUrls($resource, $key, $datasetOverride)[0];
    }

    /**
     * @return list<string>
     */
    public function resoEntityUrls(string $resource, string $key, ?string $datasetOverride = null): array
    {
        $safeKey = str_replace("'", "''", $key);

        return array_map(
            fn (string $url): string => $url."('{$safeKey}')",
            $this->resoCollectionUrls($resource, $datasetOverride),
        );
    }

    public function listingPhotoUrl(string $listingKey, string $photoId, ?string $datasetOverride = null): string
    {
        $template = (string) config('bridge.listing_photo_path_template');
        $dataset = $this->resolveDataset($datasetOverride);

        if (str_starts_with($template, 'http://') || str_starts_with($template, 'https://')) {
            return str_replace(
                ['{dataset}', '{listingKey}', '{photoId}'],
                [$dataset, rawurlencode($listingKey), rawurlencode($photoId)],
                $template
            );
        }

        $path = str_replace(
            ['{dataset}', '{listingKey}', '{photoId}'],
            [$dataset, rawurlencode($listingKey), rawurlencode($photoId)],
            $template
        );

        return $this->joinUrl(ltrim($path, '/'));
    }

    /**
     * Doc-style absolute paths that are not always under /{dataset}/ (e.g. /pub/parcels).
     */
    public function docPath(string $path): string
    {
        return rtrim((string) config('bridge.host'), '/').'/'.ltrim($path, '/');
    }

    public function getBinaryFromUrl(string $url, Request $incoming): Response
    {
        $token = $this->resolveBearerToken();
        if ($token === null) {
            abort(503, 'Bridge server token is not configured.');
        }

        $timeout = (int) config('bridge.timeout_seconds');

        return BridgeUpstreamRetry::run(fn () => Http::timeout($timeout)
            ->withHeaders($this->forwardClientHeaders($incoming))
            ->withToken($token)
            ->withHeaders([
                'Accept' => 'image/jpeg,image/webp,image/*;q=0.8,*/*;q=0.5',
            ])
            ->get($url));
    }

    /**
     * @return array<string, string>
     */
    private function forwardClientHeaders(Request $incoming): array
    {
        $headers = [];

        $map = [
            'accept' => 'Accept',
            'accept-language' => 'Accept-Language',
            'prefer' => 'Prefer',
            'if-none-match' => 'If-None-Match',
        ];

        foreach ($map as $from => $to) {
            $value = $incoming->headers->get($from);
            if (is_string($value) && $value !== '') {
                $headers[$to] = $value;
            }
        }

        return $headers;
    }

    private function joinUrl(string $path): string
    {
        return rtrim((string) config('bridge.host'), '/').'/'.ltrim($path, '/');
    }

    private function datasetPathPrefix(?string $datasetOverride = null): string
    {
        $prefix = trim((string) config('bridge.path_prefix', ''), '/');
        $dataset = trim($this->resolveDataset($datasetOverride), '/');

        if ($prefix === '') {
            return $dataset;
        }

        return $prefix.'/'.$dataset;
    }

    private function resolveDataset(?string $datasetOverride): string
    {
        if (is_string($datasetOverride) && trim($datasetOverride) !== '') {
            return trim($datasetOverride);
        }

        return trim((string) config('bridge.dataset', 'stellar'), '/');
    }

    /**
     * @param  callable(): Response  $request
     */
    private function rateLimitedRequest(callable $request): Response
    {
        $this->rateLimitGuard->acquire();
        $response = $request();
        $this->rateLimitGuard->recordFromResponse($response);

        return $response;
    }

    private function resolveBearerToken(): ?string
    {
        $fromMls = config('mls.bridge.api_key');
        if (is_string($fromMls) && $fromMls !== '') {
            return $fromMls;
        }

        $legacy = config('bridge.server_token');
        if (is_string($legacy) && $legacy !== '') {
            return $legacy;
        }

        return null;
    }
}
