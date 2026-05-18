<?php

namespace Tests\Feature\Bridge;

use App\Events\Bridge\BridgeReplicationPageFetched;
use App\Jobs\BridgePersistReplicaChunkJob;
use App\Jobs\BridgePersistReplicaFinalizeJob;
use App\Jobs\BridgeSyncFetchPageJob;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Bridge\BridgeSyncFetchScheduler;
use App\Services\Bridge\BridgeSyncService;
use App\Services\Bridge\BridgeSyncTelemetry;
use Carbon\CarbonImmutable;
use Illuminate\Bus\PendingBatch;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Bus;
use Illuminate\Support\Facades\Event;
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

    public function test_replication_first_page_requests_active_pending_filter_and_media(): void
    {
        $replicationUrl = 'https://bridge.test/OData/stellar/Property/replication';

        Http::fake([
            $replicationUrl.'*' => Http::response(['value' => []], 200),
        ]);

        Bus::fake();

        $this->runFetchJob('stellar', 'replication');

        Http::assertSent(function (Request $request) use ($replicationUrl): bool {
            if (! str_starts_with($request->url(), $replicationUrl)) {
                return false;
            }

            $filter = $request->data()['$filter'] ?? '';

            return str_contains($filter, "StandardStatus eq 'Active'")
                && str_contains($filter, "StandardStatus eq 'Pending'")
                && str_contains($request->data()['$select'] ?? '', 'Media')
                && ! array_key_exists('$unselect', $request->data());
        });
    }

    public function test_replication_fetch_stages_page_and_dispatches_persist_batch(): void
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

        $this->runFetchJob('stellar', 'replication');

        $this->assertDatabaseCount('bridge_replica_pages', 1);

        Bus::assertBatched(function (PendingBatch $batch): bool {
            if ($batch->jobs->count() !== 1) {
                return false;
            }

            $job = $batch->jobs->first();

            return $job instanceof BridgePersistReplicaChunkJob
                && $job->dataset === 'stellar'
                && $job->pageId > 0
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

        $this->runFetchJob('stellar', 'replication');

        $chunkJob = null;
        Bus::assertBatched(function (PendingBatch $batch) use (&$chunkJob): bool {
            $chunkJob = $batch->jobs->first();

            return $chunkJob instanceof BridgePersistReplicaChunkJob;
        });
        $this->assertInstanceOf(BridgePersistReplicaChunkJob::class, $chunkJob);

        $store = app(BridgeReplicaPageStore::class);
        $chunkJob->handle(
            app(BridgeSyncService::class),
            app(BridgeSyncTelemetry::class),
            $store,
        );

        $finalize = new BridgePersistReplicaFinalizeJob(
            dataset: 'stellar',
            replicaPageId: $chunkJob->pageId,
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
            $store,
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
        $this->assertDatabaseMissing('bridge_replica_pages', ['id' => $chunkJob->pageId]);
    }

    public function test_incremental_fetch_uses_modification_timestamp_gt_filter(): void
    {
        $propertyUrl = 'https://bridge.test/OData/stellar/Property';
        $cursorTs = CarbonImmutable::parse('2026-05-10T15:30:00Z');

        ListingSyncCursor::query()->create([
            'dataset_slug' => 'stellar',
            'replication_in_progress' => false,
            'last_bridge_modification_timestamp' => $cursorTs,
        ]);

        Http::fake([
            $propertyUrl.'*' => Http::response(['value' => []], 200),
        ]);

        Bus::fake();

        $this->runFetchJob('stellar', 'incremental');

        Http::assertSent(function (Request $request) use ($cursorTs): bool {
            $filter = $request->data()['$filter'] ?? '';

            return str_contains($filter, 'ModificationTimestamp gt datetime')
                && str_contains($filter, $cursorTs->utc()->format('Y-m-d\TH:i:s\Z'));
        });
    }

    public function test_incremental_fetch_skipped_while_replication_in_progress(): void
    {
        ListingSyncCursor::query()->create([
            'dataset_slug' => 'stellar',
            'replication_in_progress' => true,
            'last_bridge_modification_timestamp' => now(),
        ]);

        Http::fake();
        Bus::fake();

        $this->runFetchJob('stellar', 'incremental');

        Http::assertNothingSent();
    }

    public function test_replication_fetch_emits_page_fetched_telemetry(): void
    {
        $replicationUrl = 'https://bridge.test/OData/stellar/Property/replication';

        Http::fake([
            $replicationUrl.'*' => Http::response([
                'value' => [
                    $this->sampleListingRow('STELLAR-100'),
                    array_merge($this->sampleListingRow('STELLAR-101'), ['StandardStatus' => 'Pending']),
                ],
            ], 200),
        ]);

        Bus::fake();
        Event::fake([BridgeReplicationPageFetched::class]);

        $this->runFetchJob('stellar', 'replication');

        Event::assertDispatched(BridgeReplicationPageFetched::class, function (BridgeReplicationPageFetched $event): bool {
            return $event->listingsDownloaded === 2
                && $event->statusCounts['active'] === 1
                && $event->statusCounts['pending'] === 1
                && isset($event->odataQuery['$filter']);
        });
    }

    private function runFetchJob(string $dataset, string $mode): void
    {
        (new BridgeSyncFetchPageJob($dataset, $mode, 0, 0))->handle(
            app(BridgeSyncService::class),
            app(BridgeSyncTelemetry::class),
            app(BridgeReplicaPageStore::class),
        );
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
