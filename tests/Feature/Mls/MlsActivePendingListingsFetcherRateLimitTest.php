<?php

namespace Tests\Feature\Mls;

use App\Services\Mls\MlsActivePendingListingsFetcher;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class MlsActivePendingListingsFetcherRateLimitTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        Cache::flush();

        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.server_token' => 'test-token',
            'bridge.sync_max_requests_per_minute' => 280,
            'mls.listings_sync_page_size' => 1,
            'mls.listings_sync_max_pages' => 5,
            'mls.listings_sync_max_rows' => 10,
        ]);
    }

    public function test_mls_cache_pagination_records_rate_limit_per_bridge_page(): void
    {
        $pageOne = 'https://bridge.test/stellar/Property*';
        $nextUrl = 'https://bridge.test/stellar/Property?page=2';

        Http::fake([
            $pageOne => Http::response([
                'value' => [['ListingKey' => 'A-1', 'StandardStatus' => 'Active']],
                '@odata.nextLink' => $nextUrl,
            ], 200),
            $nextUrl => Http::response([
                'value' => [['ListingKey' => 'A-2', 'StandardStatus' => 'Pending']],
            ], 200),
        ]);

        $request = Request::create('/api/v1/listings', 'GET');
        $request->attributes->set('mls.feed_code', 'bridge_stellar');

        app(MlsActivePendingListingsFetcher::class)->fetchMergedCollectionForCache($request);

        $state = Cache::get('bridge.sync.rate_limit_state');
        $this->assertIsArray($state);
        $this->assertGreaterThanOrEqual(2, (int) ($state['minute_request_count'] ?? 0));
    }
}
