<?php

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\ListingsCacheService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Tests\TestCase;

class ListingsCacheRowsTest extends TestCase
{
    use RefreshDatabase;

    public function test_remember_listings_persists_rows_and_purges_removed_keys(): void
    {
        config(['bridge.listings_cache_ttl_seconds' => 0]);

        $svc = app(ListingsCacheService::class);

        $svc->rememberListingsCollection('dom-a', 'bridge_stellar', function (): array {
            return [
                'body' => json_encode([
                    'value' => [
                        ['ListingKey' => 'A:1', 'StandardStatus' => 'Active'],
                        ['ListingKey' => 'A:2', 'StandardStatus' => 'Pending'],
                    ],
                ], JSON_THROW_ON_ERROR),
                'etag' => null,
            ];
        });

        $this->assertSame(2, DB::table('listings_cache')->where('domain_slug', 'dom-a')->count());

        $svc->rememberListingsCollection('dom-a', 'bridge_stellar', function (): array {
            return [
                'body' => json_encode([
                    'value' => [
                        ['ListingKey' => 'A:1', 'StandardStatus' => 'Active'],
                    ],
                ], JSON_THROW_ON_ERROR),
                'etag' => null,
            ];
        });

        $this->assertSame(1, DB::table('listings_cache')->where('domain_slug', 'dom-a')->count());
        $this->assertSame('A:1', DB::table('listings_cache')->where('domain_slug', 'dom-a')->value('listing_key'));
    }
}
