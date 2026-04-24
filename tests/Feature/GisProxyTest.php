<?php

namespace Tests\Feature;

use App\Jobs\PersistGisGeoJsonBackupJob;
use App\Models\Domain;
use App\Models\GisSourceState;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Bus;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Storage;
use Tests\TestCase;

class GisProxyTest extends TestCase
{
    use RefreshDatabase;

    private static function pinellasSampleFc(): string
    {
        return json_encode([
            'type' => 'FeatureCollection',
            'features' => [
                [
                    'type' => 'Feature',
                    'geometry' => [
                        'type' => 'Polygon',
                        'coordinates' => [[
                            [-82.7977, 27.9744],
                            [-82.7975, 27.9744],
                            [-82.7975, 27.9746],
                            [-82.7977, 27.9746],
                            [-82.7977, 27.9744],
                        ]],
                    ],
                    'properties' => [
                        'OBJECTID' => 1,
                        'PARCELID' => '092915574020010010',
                        'OWNER1' => 'Example Owner LLC',
                        'SITE_ADDRESS' => '605 NICHOLSON ST',
                    ],
                ],
            ],
        ], JSON_THROW_ON_ERROR);
    }

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'gis.edge_cache_ttl_seconds' => 900,
            'gis.http_timeout_seconds' => 5,
            'gis.http_connect_timeout_seconds' => 2,
            'gis.max_bbox_span_degrees' => 2.0,
            'gis.queue_backup_writes' => false,
            'gis.florida_mls_codes' => ['stellar', 'demo_mls'],
        ]);

        Storage::fake('gis_backup');
        Cache::flush();
        Bus::fake();

        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);
    }

    public function test_gis_failover_serves_pinellas_when_primary_fails(): void
    {
        Http::fake([
            'services9.arcgis.com/*' => Http::response('bad gateway', 502),
            'egis.pinellas.gov/*' => Http::response(self::pinellasSampleFc(), 200, ['Content-Type' => 'application/json']),
        ]);

        $bbox = '-82.83,27.95,-82.79,27.98';
        $response = $this->getJson('/api/v1/gis?bbox='.$bbox, [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertOk();
        $response->assertJsonPath('meta.source_tier', 'pinellas');
        $response->assertJsonPath('meta.source_used', 'pinellas_enterprise_parcels');
        $response->assertJsonPath('type', 'FeatureCollection');
        $this->assertNotEmpty($response->json('features'));
        Http::assertSentCount(2);
    }

    public function test_gis_second_request_hits_edge_cache_without_new_http(): void
    {
        Http::fake([
            'services9.arcgis.com/*' => Http::response('bad gateway', 502),
            'egis.pinellas.gov/*' => Http::response(self::pinellasSampleFc(), 200, ['Content-Type' => 'application/json']),
        ]);

        $bbox = '-82.83,27.95,-82.79,27.98';
        $this->getJson('/api/v1/gis?bbox='.$bbox, [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk();

        $this->getJson('/api/v1/gis?bbox='.$bbox, [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk()->assertJsonPath('meta.cache_hit', 'laravel_cache');

        Http::assertSentCount(2);
    }

    public function test_generation_bump_forces_refetch_even_when_edge_warm(): void
    {
        Http::fake([
            'services9.arcgis.com/*' => Http::response('bad gateway', 502),
            'egis.pinellas.gov/*' => Http::response(self::pinellasSampleFc(), 200, ['Content-Type' => 'application/json']),
        ]);

        $bbox = '-82.83,27.95,-82.79,27.98';
        $this->getJson('/api/v1/gis?bbox='.$bbox, [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk();

        GisSourceState::query()->where('source_key', 'pinellas_enterprise_parcels')->increment('generation');

        $this->getJson('/api/v1/gis?bbox='.$bbox, [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk();

        Http::assertSentCount(4);
    }

    public function test_gis_degrades_when_no_sources_succeed(): void
    {
        Http::fake([
            'services9.arcgis.com/*' => Http::response('timeout', 504),
            'egis.pinellas.gov/*' => Http::response('error', 500),
            'maps.hillsboroughcounty.org/*' => Http::response('error', 500),
        ]);

        $bbox = '-82.83,27.95,-82.79,27.98';
        $response = $this->getJson('/api/v1/gis?bbox='.$bbox, [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertOk();
        $response->assertJsonPath('meta.degraded', true);
        $response->assertJsonPath('meta.source_tier', 'degraded_osm_hint');
        $this->assertSame([], $response->json('features'));
        $this->assertArrayHasKey('leaflet_fallback', $response->json('meta'));
    }

    public function test_mls_scoped_gis_rejects_unknown_mls(): void
    {
        $response = $this->getJson('/api/v1/mls/unknown-board/gis?bbox=-82.83,27.95,-82.79,27.98', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertNotFound();
    }

    public function test_mls_scoped_gis_accepts_configured_florida_mls(): void
    {
        Http::fake([
            'services9.arcgis.com/*' => Http::response('bad gateway', 502),
            'egis.pinellas.gov/*' => Http::response(self::pinellasSampleFc(), 200, ['Content-Type' => 'application/json']),
        ]);

        $this->getJson('/api/v1/mls/stellar/gis?bbox=-82.83,27.95,-82.79,27.98', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk()->assertJsonPath('meta.mls_code', 'stellar');
    }

    public function test_teaser_mode_hides_non_teaser_fields(): void
    {
        Http::fake([
            'services9.arcgis.com/*' => Http::response('bad gateway', 502),
            'egis.pinellas.gov/*' => Http::response(self::pinellasSampleFc(), 200, ['Content-Type' => 'application/json']),
        ]);

        $response = $this->getJson('/api/v1/gis?bbox=-82.83,27.95,-82.79,27.98', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertOk();
        $response->assertJsonPath('meta.teaser', true);
        $props = $response->json('features.0.properties');
        $this->assertIsArray($props);
        $this->assertArrayNotHasKey('OWNER1', $props);
        $this->assertArrayHasKey('PARCELID', $props);
    }

    public function test_queue_backup_dispatches_when_enabled(): void
    {
        config(['gis.queue_backup_writes' => true]);
        Bus::fake();

        Http::fake([
            'services9.arcgis.com/*' => Http::response('bad gateway', 502),
            'egis.pinellas.gov/*' => Http::response(self::pinellasSampleFc(), 200, ['Content-Type' => 'application/json']),
        ]);

        $this->getJson('/api/v1/gis?bbox=-82.83,27.95,-82.79,27.98', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk();

        Bus::assertDispatched(PersistGisGeoJsonBackupJob::class);
    }

    public function test_oversized_bbox_returns_422(): void
    {
        config(['gis.max_bbox_span_degrees' => 0.05]);

        $response = $this->getJson('/api/v1/gis?bbox=-82.83,27.95,-82.50,27.98', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertStatus(422);
    }
}
