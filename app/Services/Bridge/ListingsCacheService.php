<?php

namespace App\Services\Bridge;

use Carbon\CarbonInterface;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;
use JsonException;

class ListingsCacheService
{
    /**
     * Revenue impact: 15-minute gzip cache reduces Bridge billable calls and keeps
     * p95 latency low so gated visitors convert before session abandonment.
     *
     * Compliance: rows store only Active/Pending RESO snapshots; closed inventory is never persisted (MLS GRID IDX).
     */
    public function rememberListingsCollection(string $domainSlug, string $feedCode, callable $supplier): string
    {
        $ttlSeconds = (int) config('bridge.listings_cache_ttl_seconds');
        $retentionDays = (int) config('mls.listings_row_retention_days', 365);

        $this->purgeExpiredRetention($domainSlug, $feedCode, $retentionDays);

        if ($this->aggregateSnapshotFresh($domainSlug, $feedCode, $ttlSeconds)) {
            return $this->assembleListingsCollectionBody($domainSlug, $feedCode);
        }

        /** @var array{body: string, etag: ?string} $payload */
        $payload = $supplier();
        $this->replaceRowsFromUpstreamBody($domainSlug, $feedCode, $payload['body']);

        return $this->assembleListingsCollectionBody($domainSlug, $feedCode);
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

    private function aggregateSnapshotFresh(string $domainSlug, string $feedCode, int $ttlSeconds): bool
    {
        $minRefreshed = DB::table('listings_cache')
            ->where('domain_slug', $domainSlug)
            ->where('feed_code', $feedCode)
            ->min('last_refreshed_at');

        if ($minRefreshed === null) {
            return false;
        }

        return Carbon::parse($minRefreshed)->greaterThan(now()->subSeconds($ttlSeconds));
    }

    private function assembleListingsCollectionBody(string $domainSlug, string $feedCode): string
    {
        $rows = DB::table('listings_cache')
            ->where('domain_slug', $domainSlug)
            ->where('feed_code', $feedCode)
            ->orderBy('listing_key')
            ->get(['compressed_payload']);

        $value = [];
        foreach ($rows as $row) {
            $decoded = @gzdecode((string) $row->compressed_payload);
            if (! is_string($decoded) || $decoded === '') {
                continue;
            }
            try {
                $one = json_decode($decoded, true, 512, JSON_THROW_ON_ERROR);
            } catch (JsonException) {
                continue;
            }
            if (is_array($one)) {
                $value[] = $one;
            }
        }

        return json_encode(['value' => $value], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    }

    private function replaceRowsFromUpstreamBody(string $domainSlug, string $feedCode, string $body): void
    {
        $list = $this->extractListingListFromJson($body);
        $allowUnknownStatus = false;
        if ($list !== [] && is_array($list[0]) && ! array_key_exists('StandardStatus', $list[0])) {
            $allowUnknownStatus = true;
        }

        $now = now();
        $seenKeys = [];

        $existingFirst = DB::table('listings_cache')
            ->where('domain_slug', $domainSlug)
            ->where('feed_code', $feedCode)
            ->pluck('first_cached_at', 'listing_key')
            ->all();

        foreach ($list as $item) {
            if (! is_array($item)) {
                continue;
            }
            $status = $this->normalizeStatus($item['StandardStatus'] ?? null);
            if (! $allowUnknownStatus && ! $this->shouldPersistStatus($status)) {
                continue;
            }
            $listingKey = $this->resolveListingKey($item);
            if ($listingKey === null) {
                continue;
            }
            $seenKeys[$listingKey] = true;

            $firstCached = $existingFirst[$listingKey] ?? null;
            $firstAt = $firstCached !== null ? Carbon::parse($firstCached) : $now;

            DB::table('listings_cache')->updateOrInsert(
                [
                    'domain_slug' => $domainSlug,
                    'feed_code' => $feedCode,
                    'listing_key' => $listingKey,
                ],
                [
                    'standard_status' => $status,
                    'compressed_payload' => gzencode(json_encode($item, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES), 9),
                    'first_cached_at' => $firstAt,
                    'last_refreshed_at' => $now,
                ],
            );
        }

        if ($seenKeys !== []) {
            DB::table('listings_cache')
                ->where('domain_slug', $domainSlug)
                ->where('feed_code', $feedCode)
                ->whereNotIn('listing_key', array_keys($seenKeys))
                ->delete();
        }
    }

    /**
     * @return list<array<string, mixed>>
     */
    private function extractListingListFromJson(string $body): array
    {
        try {
            $data = json_decode($body, true, 512, JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            return [];
        }

        if (! is_array($data)) {
            return [];
        }

        foreach (['value', 'bundle', 'd', 'listings'] as $key) {
            if (isset($data[$key]) && is_array($data[$key]) && $this->isList($data[$key])) {
                /** @var list<array<string, mixed>> $out */
                $out = [];
                foreach ($data[$key] as $row) {
                    if (is_array($row)) {
                        $out[] = $row;
                    }
                }

                return $out;
            }
        }

        if ($this->isList($data)) {
            /** @var list<array<string, mixed>> $out */
            $out = [];
            foreach ($data as $row) {
                if (is_array($row)) {
                    $out[] = $row;
                }
            }

            return $out;
        }

        return [];
    }

    /**
     * @param  array<mixed>  $maybeList
     */
    private function isList(array $maybeList): bool
    {
        if ($maybeList === []) {
            return true;
        }

        return array_keys($maybeList) === range(0, count($maybeList) - 1);
    }

    private function resolveListingKey(array $item): ?string
    {
        foreach (['ListingKey', 'Id', 'id', 'ListingId'] as $k) {
            if (isset($item[$k]) && is_string($item[$k]) && trim($item[$k]) !== '') {
                return trim($item[$k]);
            }
            if (isset($item[$k]) && (is_int($item[$k]) || is_float($item[$k]))) {
                return (string) $item[$k];
            }
        }

        return null;
    }

    private function normalizeStatus(mixed $raw): string
    {
        if (! is_string($raw)) {
            return '';
        }

        return trim($raw);
    }

    private function shouldPersistStatus(string $status): bool
    {
        $u = strtoupper(trim($status));

        return in_array($u, ['ACTIVE', 'PENDING'], true);
    }

    private function purgeExpiredRetention(string $domainSlug, string $feedCode, int $retentionDays): void
    {
        DB::table('listings_cache')
            ->where('domain_slug', $domainSlug)
            ->where('feed_code', $feedCode)
            ->where('first_cached_at', '<', now()->subDays($retentionDays))
            ->delete();
    }
}
