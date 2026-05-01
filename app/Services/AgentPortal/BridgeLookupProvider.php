<?php

namespace App\Services\AgentPortal;

use App\Models\LookupCacheSnapshot;
use App\Services\AgentPortal\Contracts\LookupProviderInterface;
use App\Services\AgentPortal\Support\CacheKeyFactory;
use App\Services\Bridge\BridgeHttpService;
use Illuminate\Http\Request;

final class BridgeLookupProvider implements LookupProviderInterface
{
    public function __construct(
        private readonly BridgeHttpService $bridge,
        private readonly CacheKeyFactory $keys,
    ) {}

    public function fetchFieldCatalog(string $mlsCode, string $datasetCode): array
    {
        $payload = $this->rememberLookupPayload(
            mlsCode: $mlsCode,
            datasetCode: $datasetCode,
            scope: 'fields',
            query: ['$filter' => "ResourceName eq 'Property'"],
        );

        return is_array($payload['value'] ?? null) ? $payload['value'] : [];
    }

    public function fetchEnumValues(string $mlsCode, string $datasetCode, string $sourceFieldKey): array
    {
        $payload = $this->rememberLookupPayload(
            mlsCode: $mlsCode,
            datasetCode: $datasetCode,
            scope: 'enums.'.$sourceFieldKey,
            query: ['$filter' => "LookupName eq '".$sourceFieldKey."'"],
        );

        return is_array($payload['value'] ?? null) ? $payload['value'] : [];
    }

    public function fetchBoundaryCatalog(string $mlsCode, string $datasetCode): array
    {
        $payload = $this->rememberLookupPayload(
            mlsCode: $mlsCode,
            datasetCode: $datasetCode,
            scope: 'boundaries',
            query: ['$filter' => "ResourceName eq 'Property' and (LookupName eq 'City' or LookupName eq 'CountyOrParish')"],
        );

        return is_array($payload['value'] ?? null) ? $payload['value'] : [];
    }

    /**
     * @param  array<string, mixed>  $query
     * @return array<string, mixed>
     */
    private function rememberLookupPayload(
        string $mlsCode,
        string $datasetCode,
        string $scope,
        array $query
    ): array {
        $version = date('Y-m-d');
        $cacheKey = $this->keys->lookups($mlsCode, $datasetCode, $scope, 'v'.$version);

        $snapshot = LookupCacheSnapshot::query()->where('cache_key', $cacheKey)->first();
        if ($snapshot instanceof LookupCacheSnapshot && $snapshot->expires_at !== null && $snapshot->expires_at->isFuture()) {
            return is_array($snapshot->payload_json) ? $snapshot->payload_json : [];
        }

        $url = $this->bridge->resoCollectionUrl('Lookup');
        try {
            $response = $this->bridge->getJsonFromUrl($url, new Request, $query);
        } catch (\Throwable) {
            return $snapshot instanceof LookupCacheSnapshot && is_array($snapshot->payload_json)
                ? $snapshot->payload_json
                : [];
        }
        if (! $response->successful()) {
            return $snapshot instanceof LookupCacheSnapshot && is_array($snapshot->payload_json)
                ? $snapshot->payload_json
                : [];
        }

        /** @var array<string, mixed> $payload */
        $payload = is_array($response->json()) ? $response->json() : [];
        $ttl = max(300, (int) config('bridge.lookups_cache_ttl_seconds', 86_400));
        $checksum = hash('sha256', json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));

        LookupCacheSnapshot::query()->updateOrCreate(
            ['cache_key' => $cacheKey],
            [
                'mls_code' => $mlsCode,
                'dataset_code' => $datasetCode,
                'scope' => $scope,
                'payload_json' => $payload,
                'checksum' => $checksum,
                'version_tag' => 'v'.$version,
                'expires_at' => now()->addSeconds($ttl),
                'refreshed_at' => now(),
            ],
        );

        return $payload;
    }
}
