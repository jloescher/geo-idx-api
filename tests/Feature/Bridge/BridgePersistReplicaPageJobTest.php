<?php

namespace Tests\Feature\Bridge;

use App\Jobs\BridgePersistReplicaChunkJob;
use App\Jobs\BridgePersistReplicaFinalizeJob;
use App\Jobs\BridgeSyncFetchPageJob;
use App\Models\Listing;
use App\Services\Bridge\BridgeSyncFetchScheduler;
use App\Services\Bridge\BridgeSyncService;
use App\Services\Bridge\BridgeSyncTelemetry;
use App\Services\Replication\ReplicaPageStore;
use App\Services\Replication\ReplicationCursorPatch;
use Carbon\CarbonImmutable;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

class BridgePersistReplicaPageJobTest extends TestCase
{
    use RefreshDatabase;

    public function test_persist_chunk_upserts_active_listing_from_staged_page(): void
    {
        $maxTs = CarbonImmutable::parse('2026-05-01T12:00:00Z');
        $store = app(ReplicaPageStore::class);

        $pageId = $store->storePage(
            datasetSlug: 'stellar',
            mode: 'replication',
            rows: [
                [
                    'ListingKey' => 'STELLAR-300',
                    'ListingId' => 'MLS-300',
                    'StandardStatus' => 'Active',
                    'BridgeModificationTimestamp' => $maxTs->toIso8601String(),
                    'ModificationTimestamp' => $maxTs->toIso8601String(),
                    'ListPrice' => 425000,
                    'City' => 'Orlando',
                    'StateOrProvince' => 'FL',
                    'Latitude' => 28.54,
                    'Longitude' => -81.38,
                ],
                [
                    'ListingKey' => 'STELLAR-301',
                    'ListingId' => 'MLS-301',
                    'StandardStatus' => 'Closed',
                    'BridgeModificationTimestamp' => $maxTs->toIso8601String(),
                ],
            ],
            bridgeUrl: 'https://bridge.test/page',
            odataQuery: ['$top' => 2000],
        );

        $job = new BridgePersistReplicaChunkJob(
            dataset: 'stellar',
            pageId: $pageId,
            chunkIndex: 1,
            chunkTotal: 1,
        );

        $job->handle(
            app(BridgeSyncService::class),
            app(BridgeSyncTelemetry::class),
            $store,
        );

        $this->assertDatabaseHas('listings', [
            'dataset_slug' => 'stellar',
            'listing_key' => 'STELLAR-300',
            'standard_status' => 'Active',
        ]);

        $this->assertDatabaseMissing('listings', [
            'dataset_slug' => 'stellar',
            'listing_key' => 'STELLAR-301',
        ]);
    }

    public function test_finalize_job_dispatches_incremental_fetch_and_purges_staging_page(): void
    {
        config(['bridge.sync_fetch_queue' => 'bridge-sync-fetch']);
        Queue::fake();

        $maxTs = CarbonImmutable::parse('2026-05-02T08:00:00Z');
        $store = app(ReplicaPageStore::class);

        $pageId = $store->storePage(
            datasetSlug: 'stellar',
            mode: 'replication',
            rows: [
                [
                    'ListingKey' => 'STELLAR-400',
                    'StandardStatus' => 'Active',
                    'BridgeModificationTimestamp' => $maxTs->toIso8601String(),
                    'ModificationTimestamp' => $maxTs->toIso8601String(),
                    'ListPrice' => 300000,
                ],
            ],
            bridgeUrl: null,
            odataQuery: null,
        );

        (new BridgePersistReplicaChunkJob('stellar', $pageId, 1, 1))->handle(
            app(BridgeSyncService::class),
            app(BridgeSyncTelemetry::class),
            $store,
        );

        $finalize = new BridgePersistReplicaFinalizeJob(
            dataset: 'stellar',
            replicaPageId: $pageId,
            cursorPatch: new ReplicationCursorPatch(
                applyReplicationState: true,
                replicationNextUrl: null,
                replicationInProgress: false,
                maxBridgeTs: $maxTs,
            ),
            dispatchIncrementalAfter: true,
        );

        $finalize->handle(
            app(BridgeSyncService::class),
            app(BridgeSyncFetchScheduler::class),
            $store,
        );

        Queue::assertPushedOn('bridge-sync-fetch', BridgeSyncFetchPageJob::class, function (BridgeSyncFetchPageJob $fetch): bool {
            return $fetch->dataset === 'stellar' && $fetch->mode === 'incremental';
        });

        $this->assertTrue(Listing::query()->where('listing_key', 'STELLAR-400')->exists());
        $this->assertDatabaseMissing('replica_pages', ['id' => $pageId]);
    }
}
