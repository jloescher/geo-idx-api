<?php

namespace Tests\Unit\Services\Mls;

use App\Services\Mls\SparkClient;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class SparkClientTokenTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.timeout_seconds' => 5,
            'mls.spark_token_cache_store' => 'array',
            'mls.spark_token_cache_ttl_seconds' => 120,
        ]);

        Cache::store('array')->clear();
    }

    public function test_client_credentials_token_is_cached_and_reused(): void
    {
        Http::fake(function (\Illuminate\Http\Client\Request $request) {
            if (str_contains($request->url(), 'oauth/token')) {
                return Http::response(['access_token' => 'tok-1', 'expires_in' => 3600], 200);
            }

            return Http::response(['value' => []], 200);
        });

        $client = new SparkClient('space_coast', [
            'provider' => 'spark',
            'client_id' => 'cid',
            'client_secret' => 'sec',
            'token_url' => 'https://auth.spark.test/oauth/token',
            'reso_base_url' => 'https://reso.spark.test/odata/',
            'oauth_scope' => 'api',
        ]);

        $incoming = Request::create('/api/v1/listings', 'GET');
        $client->getActivePendingPropertyCollection($incoming);
        $client->getActivePendingPropertyCollection($incoming);

        Http::assertSent(function (\Illuminate\Http\Client\Request $request): bool {
            return str_contains($request->url(), 'oauth/token');
        });
        $this->assertSame(3, count(Http::recorded()));
    }
}
