<?php

namespace Tests\Unit\AgentPortal;

use App\Services\AgentPortal\BridgeSearchExecutionBroker;
use App\Services\Bridge\BridgeHttpService;
use App\Services\Bridge\BridgeSearchClient;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class BridgeSearchExecutionBrokerTest extends TestCase
{
    public function test_execute_applies_include_polygon_geometry_filter(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.timeout_seconds' => 10,
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    [
                        'ListingKey' => 'stellar:in-area',
                        'StandardStatus' => 'Active',
                        'ListPrice' => 600000,
                        'Coordinates' => ['coordinates' => [-82.45, 27.95]],
                    ],
                    [
                        'ListingKey' => 'stellar:out-area',
                        'StandardStatus' => 'Active',
                        'ListPrice' => 650000,
                        'Coordinates' => ['coordinates' => [-81.20, 28.40]],
                    ],
                ],
            ], 200),
        ]);

        $broker = new BridgeSearchExecutionBroker(new BridgeSearchClient(new BridgeHttpService));
        $rows = $broker->execute([
            'stellar@stellar' => [
                'mls_code' => 'stellar',
                'dataset_code' => 'stellar',
                'filters' => [],
                'geometries' => [
                    [
                        'geometry_type' => 'polygon',
                        'mode' => 'include',
                        'geojson' => [
                            'coordinates' => [[
                                [-82.80, 27.70],
                                [-82.10, 27.70],
                                [-82.10, 28.20],
                                [-82.80, 28.20],
                                [-82.80, 27.70],
                            ]],
                        ],
                    ],
                ],
            ],
        ]);

        $merged = $broker->merge($rows);

        $this->assertSame(1, $merged['meta']['total_items']);
        $this->assertSame('in-area', $merged['items'][0]['listingId']);
    }
}
