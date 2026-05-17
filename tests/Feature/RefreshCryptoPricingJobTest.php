<?php

namespace Tests\Feature;

use App\Jobs\RefreshCryptoPricingJob;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class RefreshCryptoPricingJobTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'coingecko.base_url' => 'https://api.coingecko.test/api/v3',
            'coingecko.asset_ids' => ['btc', 'eth'],
            'coingecko.vs_currencies' => ['usd'],
            'coingecko.cache_key' => 'coingecko.pricing.matrix',
        ]);
    }

    public function test_job_maps_internal_asset_ids_to_coingecko_api_ids(): void
    {
        Http::fake([
            'https://api.coingecko.test/*' => function ($request) {
                $this->assertStringContainsString('ids=bitcoin%2Cethereum', (string) $request->url());

                return Http::response([
                    'bitcoin' => ['usd' => 100000.0],
                    'ethereum' => ['usd' => 4500.0],
                ], 200);
            },
        ]);

        RefreshCryptoPricingJob::dispatchSync();

        $cached = cache()->get('coingecko.pricing.matrix');
        $this->assertSame(100000.0, $cached['quotes']['btc']['usd'] ?? null);
        $this->assertSame(4500.0, $cached['quotes']['eth']['usd'] ?? null);
    }
}
