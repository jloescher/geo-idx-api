<?php

namespace App\Services\Bridge;

use App\Models\ListingsCache;
use Carbon\CarbonInterface;

class ListingsCacheService
{
    /**
     * Revenue impact: 15-minute gzip cache reduces Bridge billable calls and keeps
     * p95 latency low so gated visitors convert before session abandonment.
     */
    public function rememberListingsCollection(string $domainSlug, callable $supplier): string
    {
        $ttlSeconds = (int) config('bridge.listings_cache_ttl_seconds');

        /** @var ListingsCache|null $row */
        $row = ListingsCache::query()->find($domainSlug);

        if ($row instanceof ListingsCache && $this->isFresh($row->last_updated, $ttlSeconds)) {
            $decoded = @gzdecode((string) $row->compressed_data);
            if (is_string($decoded) && $decoded !== '') {
                return $decoded;
            }
        }

        /** @var array{body: string, etag: ?string} $payload */
        $payload = $supplier();

        ListingsCache::query()->updateOrInsert(
            ['domain_slug' => $domainSlug],
            [
                'compressed_data' => gzencode($payload['body'], 9),
                'last_updated' => now(),
                'etag' => $payload['etag'] ?? null,
            ]
        );

        return $payload['body'];
    }

    private function isFresh(?CarbonInterface $lastUpdated, int $ttlSeconds): bool
    {
        if (! $lastUpdated instanceof CarbonInterface) {
            return false;
        }

        return $lastUpdated->greaterThan(now()->subSeconds($ttlSeconds));
    }
}
