<?php

namespace Tests\Feature\Spark;

use App\Enums\ListingMirrorProvider;
use App\Models\Listing;
use App\Services\Mls\ListingMirrorWriter;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class ListingMirrorWriterTest extends TestCase
{
    use RefreshDatabase;

    public function test_persists_active_listing_from_spark_sample_fixture(): void
    {
        $fixturePath = base_path('docs/spark/beaches_50_listings.json');
        $this->assertFileExists($fixturePath);

        $payload = json_decode((string) file_get_contents($fixturePath), true, 512, JSON_THROW_ON_ERROR);
        $row = $payload['value'][0] ?? null;
        $this->assertIsArray($row);

        $writer = app(ListingMirrorWriter::class);
        $stats = $writer->hydrateReplicaBatch('beaches', [$row], ListingMirrorProvider::Spark);

        $this->assertSame(1, $stats->upserted);

        $listing = Listing::query()
            ->where('dataset_slug', 'beaches')
            ->where('listing_key', $row['ListingKey'])
            ->first();

        $this->assertNotNull($listing);
        $this->assertSame('active', strtolower((string) $listing->standard_status));
        $this->assertNull($listing->bridge_modification_timestamp);
        $this->assertNotNull($listing->modification_timestamp);
        $this->assertEquals(1.5, (float) $listing->bathrooms_total_decimal);
        $this->assertNotNull($listing->latitude);
        $this->assertNotNull($listing->longitude);
        $this->assertIsArray($listing->raw_data);
        $this->assertArrayHasKey('Media', $listing->raw_data);
    }
}
