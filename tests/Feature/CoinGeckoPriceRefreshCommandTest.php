<?php

namespace Tests\Feature;

use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class CoinGeckoPriceRefreshCommandTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'coingecko.base_url' => 'https://api.coingecko.test/api/v3',
            'coingecko.api_key' => 'demo-key',
            'coingecko.asset_ids' => ['btc', 'eth', 'sol', 'xrp', 'ada'],
            'coingecko.vs_currencies' => ['usd', 'cad', 'eur', 'gbp', 'mxn'],
            'coingecko.cache_key' => 'coingecko.pricing.matrix',
        ]);
    }

    public function test_crypto_refresh_prices_command_persists_quotes_and_warms_cache(): void
    {
        Http::fake([
            'https://api.coingecko.test/*' => Http::response([
                'bitcoin' => ['usd' => 100000.0, 'cad' => 136000.0, 'eur' => 92000.0, 'gbp' => 78000.0, 'mxn' => 1700000.0],
                'ethereum' => ['usd' => 4500.0, 'cad' => 6120.0, 'eur' => 4140.0, 'gbp' => 3510.0, 'mxn' => 76500.0],
                'solana' => ['usd' => 200.0, 'cad' => 272.0, 'eur' => 184.0, 'gbp' => 156.0, 'mxn' => 3400.0],
                'ripple' => ['usd' => 2.0, 'cad' => 2.72, 'eur' => 1.84, 'gbp' => 1.56, 'mxn' => 34.0],
                'cardano' => ['usd' => 1.2, 'cad' => 1.63, 'eur' => 1.10, 'gbp' => 0.94, 'mxn' => 20.4],
            ], 200),
        ]);

        $this->artisan('crypto:refresh-prices')
            ->assertSuccessful();

        $this->assertDatabaseCount('crypto_price_snapshots', 25);

        $cached = Cache::get('coingecko.pricing.matrix');
        $this->assertIsArray($cached);
        $this->assertSame(100000.0, $cached['quotes']['btc']['usd'] ?? null);
    }

    public function test_crypto_refresh_prices_command_does_not_overwrite_existing_data_on_invalid_payload(): void
    {
        Cache::put('coingecko.pricing.matrix', [
            'quotes' => ['btc' => ['usd' => 12345.0]],
            'as_of' => now()->toIso8601String(),
            'status' => 'ok',
        ], now()->addMinutes(10));

        Http::fake([
            'https://api.coingecko.test/*' => Http::response(['unexpected' => 'shape'], 200),
        ]);

        $this->artisan('crypto:refresh-prices')
            ->assertFailed();

        $cached = Cache::get('coingecko.pricing.matrix');
        $this->assertSame(12345.0, $cached['quotes']['btc']['usd'] ?? null);
    }
}
