<?php

namespace Tests\Unit\Services\Spark;

use App\Services\Spark\SparkHttpService;
use Tests\TestCase;

class SparkHttpServiceTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        config([
            'spark.reso_base_url' => 'https://replication.sparkapi.com/Reso/OData',
            'spark.access_token' => 'test-spark-token',
        ]);
    }

    public function test_property_collection_url_uses_configured_reso_base(): void
    {
        $service = app(SparkHttpService::class);

        $this->assertSame(
            'https://replication.sparkapi.com/Reso/OData/Property',
            $service->propertyCollectionUrl()
        );
    }

    public function test_property_entity_url_encodes_listing_key(): void
    {
        $service = app(SparkHttpService::class);

        $this->assertSame(
            "https://replication.sparkapi.com/Reso/OData/Property('20240712154755555836000000')",
            $service->propertyEntityUrl('20240712154755555836000000')
        );
    }
}
