<?php

namespace Tests\Unit\Bridge;

use App\Services\Bridge\BridgeRateLimitGuard;
use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Cache;
use Tests\TestCase;

class BridgeRateLimitGuardTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        Cache::flush();
        config([
            'bridge.sync_max_requests_per_minute' => 280,
            'bridge.sync_min_fetch_interval_ms' => 200,
        ]);
    }

    public function test_acquire_increments_minute_request_count(): void
    {
        $guard = new BridgeRateLimitGuard;

        $guard->acquire();
        $guard->acquire();

        $state = Cache::get('bridge.sync.rate_limit_state');
        $this->assertIsArray($state);
        $this->assertSame(2, $state['request_count'] ?? null);
    }

    public function test_record_from_response_sets_extra_delay_when_burst_remaining_is_low(): void
    {
        $guard = new BridgeRateLimitGuard;

        $response = new Response(
            new \GuzzleHttp\Psr7\Response(200, ['Burst-RateLimit-Remaining' => '5'])
        );

        $guard->recordFromResponse($response);

        $state = Cache::get('bridge.sync.rate_limit_state');
        $this->assertIsArray($state);
        $this->assertGreaterThanOrEqual(2000, (int) ($state['extra_delay_ms'] ?? 0));
    }

    public function test_delay_seconds_for_next_fetch_respects_minimum_interval(): void
    {
        config(['bridge.sync_min_fetch_interval_ms' => 500]);

        $guard = new BridgeRateLimitGuard;

        $this->assertSame(1, $guard->delaySecondsForNextFetch());
    }
}
