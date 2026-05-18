<?php

namespace Tests\Unit\Services\Mls;

use App\Services\Mls\MlsClientFactory;
use Tests\TestCase;

class SparkClientLiveHostTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.datasets' => ['stellar'],
            'spark.datasets' => ['beaches'],
            'spark.live_reso_base_url' => 'https://sparkapi.com/v1/Reso/OData',
            'spark.replication_reso_base_url' => 'https://replication.sparkapi.com/Reso/OData',
        ]);
    }

    public function test_spark_client_reso_urls_use_live_host(): void
    {
        $client = app(MlsClientFactory::class)->sparkClientForFeed('spark_beaches');

        $this->assertSame(
            ['https://sparkapi.com/v1/Reso/OData/Property'],
            $client->resoCollectionUrls('Property')
        );

        $this->assertSame(
            ["https://sparkapi.com/v1/Reso/OData/Property('LISTING-1')"],
            $client->resoEntityUrls('Property', 'LISTING-1')
        );
    }
}
