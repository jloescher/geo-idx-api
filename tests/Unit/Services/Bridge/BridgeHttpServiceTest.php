<?php

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\BridgeHttpService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class BridgeHttpServiceTest extends TestCase
{
    public function test_reso_collection_url_includes_path_prefix_when_reso_root_is_set(): void
    {
        config([
            'bridge.host' => 'https://api.bridgedataoutput.com',
            'bridge.dataset' => 'stellar',
            'bridge.path_prefix' => 'api/v2',
            'bridge.reso_root' => 'reso/odata',
        ]);

        $service = app(BridgeHttpService::class);

        $this->assertSame(
            'https://api.bridgedataoutput.com/api/v2/reso/odata/stellar/Property',
            $service->resoCollectionUrl('Property'),
        );
    }

    public function test_reso_collection_url_uses_odata_path_when_only_prefix_is_set(): void
    {
        config([
            'bridge.host' => 'https://api.bridgedataoutput.com',
            'bridge.dataset' => 'stellar',
            'bridge.path_prefix' => 'api/v2',
            'bridge.reso_root' => '',
        ]);

        $service = app(BridgeHttpService::class);

        $this->assertSame(
            'https://api.bridgedataoutput.com/api/v2/OData/stellar/Property',
            $service->resoCollectionUrl('Property'),
        );
    }

    public function test_reso_collection_urls_include_fallback_candidates(): void
    {
        config([
            'bridge.host' => 'https://api.bridgedataoutput.com',
            'bridge.dataset' => 'stellar',
            'bridge.path_prefix' => 'api/v2',
            'bridge.reso_root' => 'reso/odata',
        ]);

        $service = app(BridgeHttpService::class);

        $this->assertSame([
            'https://api.bridgedataoutput.com/api/v2/reso/odata/stellar/Property',
            'https://api.bridgedataoutput.com/api/v2/OData/stellar/Property',
            'https://api.bridgedataoutput.com/reso/odata/stellar/Property',
            'https://api.bridgedataoutput.com/stellar/Property',
        ], $service->resoCollectionUrls('Property'));
    }

    public function test_reso_collection_urls_use_dataset_override(): void
    {
        config([
            'bridge.host' => 'https://api.bridgedataoutput.com',
            'bridge.dataset' => 'stellar',
            'bridge.path_prefix' => 'api/v2',
            'bridge.reso_root' => 'reso/odata',
        ]);

        $service = app(BridgeHttpService::class);

        $this->assertSame(
            'https://api.bridgedataoutput.com/api/v2/reso/odata/miami/Property',
            $service->resoCollectionUrl('Property', 'miami'),
        );
    }

    public function test_server_json_get_retries_on_429_then_succeeds(): void
    {
        config([
            'mls.bridge.api_key' => 'test-token',
            'bridge.timeout_seconds' => 2,
            'bridge.sync_max_http_retries' => 1,
        ]);

        $hits = 0;
        Http::fake(function () use (&$hits) {
            $hits++;
            if ($hits === 1) {
                return Http::response(['error' => 'rate'], 429, ['Retry-After' => '0']);
            }

            return Http::response(['value' => []], 200);
        });

        $service = app(BridgeHttpService::class);
        $response = $service->serverJsonGet('https://api.test/OData/stellar/Property', ['$top' => 1]);

        $this->assertTrue($response->successful());
        $this->assertSame(2, $hits);
    }

    public function test_get_json_from_url_retries_on_429_then_succeeds(): void
    {
        config([
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

            return Http::response(['ListingKey' => 'stellar:1'], 200);
        });

        $service = app(BridgeHttpService::class);
        $incoming = Request::create('http://localhost', 'GET');
        $response = $service->getJsonFromUrl('https://api.test/OData/stellar/Property(\'stellar:1\')', $incoming, ['$select' => 'ListingKey']);

        $this->assertTrue($response->successful());
        $this->assertSame(2, $hits);
    }
}
