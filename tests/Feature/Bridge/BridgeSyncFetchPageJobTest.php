<?php

namespace Tests\Feature\Bridge;

use App\Jobs\BridgePersistReplicaChunkJob;
use App\Jobs\BridgePersistReplicaFinalizeJob;
use App\Jobs\BridgeSyncFetchPageJob;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeSyncFetchScheduler;
use App\Services\Bridge\BridgeSyncService;
use Illuminate\Bus\PendingBatch;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Bus;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class BridgeSyncFetchPageJobTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.server_token' => 'test-token',
            'bridge.sync_fetch_queue' => 'bridge-sync-fetch',
            'bridge.sync_persist_queue' => 'bridge-sync-persist',
            'bridge.sync_include_media' => false,
            'bridge.sync_replication_top' => 2000,
            'bridge.sync_persist_job_chunk_size' => 100,
        ]);
    }

    public function test_replication_fetch_dispatches_parallel_persist_batch_not_fetch(): void
    {
        $replicationUrl = 'https://bridge.test/OData/stellar/Property/replication';
        $nextUrl = 'https://bridge.test/OData/stellar/Property/replication?page=2';

        Http::fake([
            $replicationUrl.'*' => Http::response([
                'value' => [
                    $this->sampleListingRow('STELLAR-100'),
                ],
            ], 200, [
                'Link' => '<'.$nextUrl.'>; rel="next"',
            ]),
        ]);

        Bus::fake();

        (new BridgeSyncFetchPageJob('stellar', 'replication', 0, 0))
            ->handle(app(BridgeSyncService::class));

        Bus::assertBatched(function (PendingBatch $batch): bool {
            if ($batch->jobs->count() !== 1) {
                return false;
            }

            $job = $batch->jobs->first();

            return $job instanceof BridgePersistReplicaChunkJob
                && $job->dataset === 'stellar'
                && count($job->rows) === 1
                && $job->queue === 'bridge-sync-persist';
        });
    }

    public function test_finalize_after_persist_batch_schedules_next_replication_fetch(): void
    {
        $replicationUrl = 'https://bridge.test/OData/stellar/Property/replication';
        $nextUrl = 'https://bridge.test/OData/stellar/Property/replication?page=2';

        Http::fake([
            $replicationUrl.'*' => Http::response([
                'value' => [
                    $this->sampleListingRow('STELLAR-200'),
                ],
            ], 200, [
                'Link' => '<'.$nextUrl.'>; rel="next"',
            ]),
        ]);

        Bus::fake();

        (new BridgeSyncFetchPageJob('stellar', 'replication', 0, 0))
            ->handle(app(BridgeSyncService::class));

        $chunkJob = null;
        Bus::assertBatched(function (PendingBatch $batch) use (&$chunkJob): bool {
            $chunkJob = $batch->jobs->first();

            return $chunkJob instanceof BridgePersistReplicaChunkJob;
        });
        $this->assertInstanceOf(BridgePersistReplicaChunkJob::class, $chunkJob);

        $chunkJob->handle(app(BridgeSyncService::class));

        $finalize = new BridgePersistReplicaFinalizeJob(
            dataset: 'stellar',
            cursorPatch: new BridgeReplicaCursorPatch(
                applyReplicationState: true,
                replicationNextUrl: $nextUrl,
                replicationInProgress: true,
            ),
            nextFetchMode: 'replication',
            nextChainDepth: 1,
        );

        Bus::fake();

        $finalize->handle(
            app(BridgeSyncService::class),
            app(BridgeSyncFetchScheduler::class),
        );

        Bus::assertDispatched(BridgeSyncFetchPageJob::class, function (BridgeSyncFetchPageJob $fetch): bool {
            return $fetch->dataset === 'stellar'
                && $fetch->mode === 'replication'
                && $fetch->chainDepth === 1;
        });

        $cursor = ListingSyncCursor::query()->where('dataset_slug', 'stellar')->first();
        $this->assertNotNull($cursor);
        $this->assertSame($nextUrl, $cursor->replication_next_url);
        $this->assertTrue($cursor->replication_in_progress);
    }

    /**
     * @return array<string, mixed>
     */
    private function sampleListingRow(string $listingKey): array
    {
        return [
            'ListingKey' => $listingKey,
            'ListingId' => 'MLS-1',
            'StandardStatus' => 'Active',
            'BridgeModificationTimestamp' => '2026-05-01T12:00:00Z',
            'ModificationTimestamp' => '2026-05-01T12:00:00Z',
            'ListPrice' => 500000,
            'City' => 'Tampa',
            'StateOrProvince' => 'FL',
            'Latitude' => 27.95,
            'Longitude' => -82.45,
        ];
    }
}
