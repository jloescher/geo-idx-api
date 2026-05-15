<?php

declare(strict_types=1);

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\BridgeSearchClient;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class BridgeSearchClientRetryTest extends TestCase
{
    public function test_search_retries_on_429_then_returns_value(): void
    {
        config([
            'bridge.host' => 'https://api.test',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.dataset' => 'stellar',
            'mls.bridge.api_key' => 'test-token',
            'bridge.timeout_seconds' => 2,
            'bridge.sync_max_http_retries' => 1,
        ]);

        $hits = 0;
        Http::fake(function () use (&$hits) {
            $hits++;
            if ($hits === 1) {
                return Http::response('', 429, ['Retry-After' => '0']);
            }

            return Http::response([
                'value' => [['ListingKey' => 'stellar:1', 'StandardStatus' => 'Active']],
            ], 200);
        });

        $client = app(BridgeSearchClient::class);
        $out = $client->search('stellar', '', '', 10, 0, 'ListingKey', 'Media');

        $this->assertSame(2, $hits);
        $this->assertCount(1, $out['value']);
        $this->assertSame('stellar:1', $out['value'][0]['ListingKey']);
    }
}
