<?php

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\BridgeHttpService;
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
}
