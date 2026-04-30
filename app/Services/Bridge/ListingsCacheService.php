<?php

namespace App\Services\Bridge;

use App\Models\ListingsCache;
use Carbon\CarbonInterface;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;

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

    /**
     * Revenue impact: gzip-backed search cache mirrors collection TTL so identical
     * structured searches avoid duplicate Bridge OData round trips across subscribers.
     *
     * @param  callable(): array{value: list<array<string, mixed>>, count: int, nextLink: ?string}  $supplier
     * @return array{value: list<array<string, mixed>>, count: int, nextLink: ?string}
     */
    public function rememberSearchResult(string $partitionKey, string $fingerprint, callable $supplier): array
    {
        $ttlSeconds = (int) config('bridge.listings_cache_ttl_seconds');

        $row = DB::table('bridge_search_cache')
            ->where('partition_key', $partitionKey)
            ->where('fingerprint', $fingerprint)
            ->first();

        $lastUpdated = $row !== null && $row->last_updated !== null
            ? Carbon::parse($row->last_updated)
            : null;

        if ($row !== null && $this->isFresh($lastUpdated, $ttlSeconds)) {
            $decoded = @gzdecode((string) $row->compressed_data);
            if (is_string($decoded) && $decoded !== '') {
                $parsed = json_decode($decoded, true);
                if (is_array($parsed)) {
                    return $this->normalizeSearchPayload($parsed);
                }
            }
        }

        /** @var array{value: list<array<string, mixed>>, count: int, nextLink: ?string} $payload */
        $payload = $supplier();

        DB::table('bridge_search_cache')->updateOrInsert(
            [
                'partition_key' => $partitionKey,
                'fingerprint' => $fingerprint,
            ],
            [
                'compressed_data' => gzencode(json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES), 9),
                'last_updated' => now(),
            ],
        );

        return $payload;
    }

    /**
     * Cache opaque JSON response bodies by partition + fingerprint.
     *
     * @param  callable(): string  $supplier
     */
    public function rememberJsonPayload(string $partitionKey, string $fingerprint, callable $supplier): string
    {
        $ttlSeconds = (int) config('bridge.listings_cache_ttl_seconds');

        $row = DB::table('bridge_search_cache')
            ->where('partition_key', $partitionKey)
            ->where('fingerprint', $fingerprint)
            ->first();

        $lastUpdated = $row !== null && $row->last_updated !== null
            ? Carbon::parse($row->last_updated)
            : null;

        if ($row !== null && $this->isFresh($lastUpdated, $ttlSeconds)) {
            $decoded = @gzdecode((string) $row->compressed_data);
            if (is_string($decoded) && $decoded !== '') {
                return $decoded;
            }
        }

        $payload = $supplier();

        DB::table('bridge_search_cache')->updateOrInsert(
            [
                'partition_key' => $partitionKey,
                'fingerprint' => $fingerprint,
            ],
            [
                'compressed_data' => gzencode($payload, 9),
                'last_updated' => now(),
            ],
        );

        return $payload;
    }

    /**
     * @param  array<string, mixed>  $parsed
     * @return array{value: list<array<string, mixed>>, count: int, nextLink: ?string}
     */
    private function normalizeSearchPayload(array $parsed): array
    {
        $value = $parsed['value'] ?? null;

        return [
            'value' => is_array($value) ? $value : [],
            'count' => is_int($parsed['count'] ?? null) ? $parsed['count'] : count(is_array($value) ? $value : []),
            'nextLink' => is_string($parsed['nextLink'] ?? null) ? $parsed['nextLink'] : null,
        ];
    }

    private function isFresh(?CarbonInterface $lastUpdated, int $ttlSeconds): bool
    {
        if (! $lastUpdated instanceof CarbonInterface) {
            return false;
        }

        return $lastUpdated->greaterThan(now()->subSeconds($ttlSeconds));
    }

    /**
     * Cache Bridge Lookup API responses for 30 days.
     *
     * Lookup data (field enums, PropertySubType values, etc.) changes rarely,
     * so a long TTL avoids repeated upstream calls for metadata that is essentially static.
     *
     * @param  callable(): string  $supplier
     */
    public function rememberLookups(string $partitionKey, string $fingerprint, callable $supplier): string
    {
        $ttlSeconds = (int) config('bridge.lookups_cache_ttl_seconds');

        $row = DB::table('bridge_search_cache')
            ->where('partition_key', $partitionKey)
            ->where('fingerprint', $fingerprint)
            ->first();

        $lastUpdated = $row !== null && $row->last_updated !== null
            ? Carbon::parse($row->last_updated)
            : null;

        if ($row !== null && $this->isFresh($lastUpdated, $ttlSeconds)) {
            $decoded = @gzdecode((string) $row->compressed_data);
            if (is_string($decoded) && $decoded !== '') {
                return $decoded;
            }
        }

        $payload = $supplier();

        DB::table('bridge_search_cache')->updateOrInsert(
            [
                'partition_key' => $partitionKey,
                'fingerprint' => $fingerprint,
            ],
            [
                'compressed_data' => gzencode($payload, 9),
                'last_updated' => now(),
            ],
        );

        return $payload;
    }
}
