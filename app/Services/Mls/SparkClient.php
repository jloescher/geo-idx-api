<?php

namespace App\Services\Mls;

use App\Services\Spark\SparkHttpService;
use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;

final readonly class SparkClient
{
    public function __construct(
        private SparkHttpService $http,
        private string $feedCode,
        private string $sparkDatasetId,
    ) {}

    public function feedCode(): string
    {
        return $this->feedCode;
    }

    public function sparkDatasetId(): string
    {
        return $this->sparkDatasetId;
    }

    public function propertyCollectionUrl(): string
    {
        return $this->http->propertyCollectionUrl();
    }

    /**
     * @return list<string>
     */
    public function resoCollectionUrls(string $resource): array
    {
        $base = rtrim((string) config('spark.reso_base_url'), '/');

        return ["{$base}/{$resource}"];
    }

    /**
     * @return list<string>
     */
    public function resoEntityUrls(string $resource, string $key): array
    {
        if (strcasecmp($resource, 'Property') === 0) {
            return [$this->http->propertyEntityUrl($key)];
        }

        $encoded = rawurlencode($key);
        $base = rtrim((string) config('spark.reso_base_url'), '/');

        return ["{$base}/{$resource}('{$encoded}')"];
    }

    public function getJsonFromUrl(string $url, Request $incoming, array $query = [], array $stripQueryKeys = []): Response
    {
        return $this->http->getJsonFromUrl($url, $incoming, $query, $stripQueryKeys);
    }

    public function listingPhotoUrl(string $listingKey, string $photoId): string
    {
        return $this->http->resolveListingPhotoUrl($listingKey, $photoId);
    }

    public function getBinaryFromUrl(string $url, Request $incoming): Response
    {
        return $this->http->getBinaryFromUrl($url, $incoming);
    }
}
