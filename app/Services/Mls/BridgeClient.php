<?php

namespace App\Services\Mls;

use App\Services\Bridge\BridgeHttpService;
use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;

/**
 * Revenue impact: request-scoped Bridge dataset routing prevents accidental cross-MLS billing under Octane.
 *
 * Compliance: Stellar MLS / MLS GRID IDX — dataset must match domain allowlist; no global mutable config per request.
 */
final readonly class BridgeClient
{
    public function __construct(
        private BridgeHttpService $http,
        private string $feedCode,
        private string $bridgeDatasetId,
    ) {}

    public function feedCode(): string
    {
        return $this->feedCode;
    }

    public function bridgeDatasetId(): string
    {
        return $this->bridgeDatasetId;
    }

    public function webUrl(string $resourcePath): string
    {
        return $this->http->webUrl($resourcePath, $this->bridgeDatasetId);
    }

    /**
     * @return list<string>
     */
    public function resoCollectionUrls(string $resource): array
    {
        return $this->http->resoCollectionUrls($resource, $this->bridgeDatasetId);
    }

    /**
     * @return list<string>
     */
    public function resoEntityUrls(string $resource, string $key): array
    {
        return $this->http->resoEntityUrls($resource, $key, $this->bridgeDatasetId);
    }

    public function listingPhotoUrl(string $listingKey, string $photoId): string
    {
        return $this->http->listingPhotoUrl($listingKey, $photoId, $this->bridgeDatasetId);
    }

    public function getJsonFromUrl(string $url, Request $incoming, array $query = [], array $stripQueryKeys = []): Response
    {
        return $this->http->getJsonFromUrl($url, $incoming, $query, $stripQueryKeys);
    }

    public function docPath(string $path): string
    {
        return $this->http->docPath($path);
    }

    public function getBinaryFromUrl(string $url, Request $incoming): Response
    {
        return $this->http->getBinaryFromUrl($url, $incoming);
    }
}
