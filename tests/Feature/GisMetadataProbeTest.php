<?php

namespace Tests\Feature;

use App\Models\GisCache;
use App\Models\GisSourceState;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class GisMetadataProbeTest extends TestCase
{
    use RefreshDatabase;

    private static function layerJson(string $version = '12'): string
    {
        return json_encode([
            'currentVersion' => (float) $version,
            'editingInfo' => ['lastEditDate' => 1_700_000_000_000],
            'serviceItemId' => 'abc',
            'name' => 'TestLayer',
        ], JSON_THROW_ON_ERROR);
    }

    public function test_probe_updates_fingerprint_and_bumps_generation_on_publish_change(): void
    {
        $primaryUrl = 'https://services9.arcgis.com/Gh9awoU677aKree0/arcgis/rest/services/Florida_Statewide_Cadastral/FeatureServer/0?f=json';
        $pinellasUrl = 'https://egis.pinellas.gov/gis/rest/services/PublicWebGIS/Parcels/MapServer/1?f=json';
        $hillsUrl = 'https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_Parcels/FeatureServer/0?f=json';

        Http::fake([
            $primaryUrl => Http::sequence()
                ->push(self::layerJson('12'), 200)
                ->push(self::layerJson('13'), 200),
            $pinellasUrl => Http::response(self::layerJson('10'), 200),
            $hillsUrl => Http::response(self::layerJson('5'), 200),
        ]);

        Artisan::call('gis:probe-sources');

        $primary = GisSourceState::query()->where('source_key', 'florida_statewide_cadastral')->firstOrFail();
        $this->assertNotNull($primary->fingerprint);
        $this->assertSame(0, (int) $primary->generation);

        Artisan::call('gis:probe-sources');

        $primary->refresh();
        $this->assertSame(1, (int) $primary->generation);
    }

    public function test_clear_cache_deletes_rows_and_bumps_generation(): void
    {
        GisCache::query()->create([
            'query_hash' => 'abc123',
            'geojson' => '{"type":"FeatureCollection","features":[]}',
            'county' => 'pinellas',
            'expires_at' => now()->addDay(),
            'source_used' => 'pinellas_enterprise_parcels',
            'source_generation' => 0,
        ]);

        $before = (int) GisSourceState::query()->where('source_key', 'pinellas_enterprise_parcels')->value('generation');

        Artisan::call('gis:clear-cache', [
            '--source' => 'pinellas_enterprise_parcels',
        ]);

        $this->assertDatabaseMissing('gis_cache', ['query_hash' => 'abc123']);
        $after = (int) GisSourceState::query()->where('source_key', 'pinellas_enterprise_parcels')->value('generation');
        $this->assertSame($before + 1, $after);
    }
}
