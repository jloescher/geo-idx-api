<?php

namespace Tests\Feature\Spark;

use App\Jobs\SparkPersistReplicaChunkJob;
use App\Jobs\SparkSyncFetchPageJob;
use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Bridge\BridgeSyncTelemetry;
use App\Services\Spark\SparkSyncService;
use Illuminate\Bus\PendingBatch;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Bus;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class SparkSyncFetchPageJobTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'spark.replication_reso_base_url' => 'https://replication.sparkapi.com/Reso/OData',
            'spark.live_reso_base_url' => 'https://sparkapi.com/v1/Reso/OData',
            'spark.access_token' => 'test-spark-token',
            'spark.sync_fetch_queue' => 'spark-sync-fetch',
            'spark.sync_persist_queue' => 'spark-sync-persist',
            'spark.sync_replication_top' => 1000,
            'spark.sync_expand' => 'Media,Unit,Room,OpenHouse',
            'spark.sync_persist_job_chunk_size' => 50,
        ]);
    }

    public function test_replication_first_page_requests_active_pending_filter_and_expand(): void
    {
        $collectionUrl = 'https://replication.sparkapi.com/Reso/OData/Property';

        Http::fake([
            $collectionUrl.'*' => Http::response(['value' => []], 200),
        ]);

        Bus::fake();

        (new SparkSyncFetchPageJob('beaches', 'replication', 0, 0))->handle(
            app(SparkSyncService::class),
            app(BridgeSyncTelemetry::class),
            app(BridgeReplicaPageStore::class),
        );

        Http::assertSent(function (Request $request) use ($collectionUrl): bool {
            if (! str_starts_with($request->url(), $collectionUrl)) {
                return false;
            }

            $filter = $request->data()['$filter'] ?? '';
            $expand = $request->data()['$expand'] ?? '';

            return str_contains($filter, "StandardStatus eq 'Active'")
                && str_contains($filter, "StandardStatus eq 'Pending'")
                && str_contains($expand, 'Media')
                && ! array_key_exists('$select', $request->data());
        });
    }

    public function test_replication_fetch_stages_page_with_spark_provider(): void
    {
        $collectionUrl = 'https://replication.sparkapi.com/Reso/OData/Property';
        $listingKey = '20240712154755555836000000';

        Http::fake([
            $collectionUrl.'*' => Http::response([
                'value' => [
                    [
                        'ListingKey' => $listingKey,
                        'StandardStatus' => 'Active',
                        'ModificationTimestamp' => '2024-07-13T00:56:59Z',
                    ],
                ],
            ], 200),
        ]);

        Bus::fake();

        (new SparkSyncFetchPageJob('beaches', 'replication', 0, 0))->handle(
            app(SparkSyncService::class),
            app(BridgeSyncTelemetry::class),
            app(BridgeReplicaPageStore::class),
        );

        $this->assertDatabaseCount('bridge_replica_pages', 1);
        $this->assertTrue(app(BridgeReplicaPageStore::class)->hasActivePage('beaches', 'spark'));

        Bus::assertBatched(function (PendingBatch $batch): bool {
            if ($batch->jobs->count() !== 1) {
                return false;
            }

            $job = $batch->jobs->first();

            return $job instanceof SparkPersistReplicaChunkJob
                && $job->dataset === 'beaches'
                && $job->pageId > 0
                && $job->queue === 'spark-sync-persist';
        });
    }
}
