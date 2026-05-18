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
            'spark.replication_reso_base_url' => 'https://replication.sparkapi.com/Reso/OData',
            'spark.live_reso_base_url' => 'https://sparkapi.com/v1/Reso/OData',
            'spark.access_token' => 'test-spark-token',
        ]);
    }

    public function test_replication_property_collection_url_uses_replication_host(): void
    {
        $service = app(SparkHttpService::class);

        $this->assertSame(
            'https://replication.sparkapi.com/Reso/OData/Property',
            $service->replicationPropertyCollectionUrl()
        );
    }

    public function test_replication_property_entity_url_encodes_listing_key(): void
    {
        $service = app(SparkHttpService::class);

        $this->assertSame(
            "https://replication.sparkapi.com/Reso/OData/Property('20240712154755555836000000')",
            $service->replicationPropertyEntityUrl('20240712154755555836000000')
        );
    }

    public function test_live_property_collection_url_uses_sparkapi_host(): void
    {
        $service = app(SparkHttpService::class);

        $this->assertSame(
            'https://sparkapi.com/v1/Reso/OData/Property',
            $service->livePropertyCollectionUrl()
        );
    }

    public function test_live_property_entity_url_encodes_listing_key(): void
    {
        $service = app(SparkHttpService::class);

        $this->assertSame(
            "https://sparkapi.com/v1/Reso/OData/Property('20240712154755555836000000')",
            $service->livePropertyEntityUrl('20240712154755555836000000')
        );
    }
}
