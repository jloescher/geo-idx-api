<?php

namespace Tests\Feature\Mls;

use App\Models\Domain;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class CheckMlsAccessTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.datasets' => ['stellar', 'miami'],
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);
    }

    public function test_gis_route_bypasses_mls_feed_gate(): void
    {
        Http::fake([
            'egis.pinellas.gov/*' => Http::response('{"type":"FeatureCollection","features":[]}', 200, ['Content-Type' => 'application/json']),
            'services9.arcgis.com/*' => Http::response('bad', 502),
        ]);

        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
            'allowed_mls_datasets' => ['miami'],
        ]);

        $bbox = '-82.83,27.95,-82.79,27.98';
        $this->getJson('/api/v1/gis?bbox='.$bbox, [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk();
    }

    public function test_listings_rejects_dataset_not_in_domain_allowlist(): void
    {
        Http::fake();

        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
            'allowed_mls_datasets' => ['miami'],
        ]);

        $this->getJson('/api/v1/listings?dataset=stellar', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertStatus(403);
    }
}
