<?php

namespace Tests\Feature\Bridge;

use App\Jobs\BridgePersistReplicaPageJob;
use App\Jobs\BridgeSyncFetchPageJob;
use App\Models\Listing;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\BridgeRateLimitGuard;
use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeSyncService;
use Carbon\CarbonImmutable;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

class BridgePersistReplicaPageJobTest extends TestCase
{
    use RefreshDatabase;

    public function test_persist_chunk_upserts_active_listing_and_updates_cursor_on_last_page(): void
    {
        config(['bridge.sync_queue' => 'bridge-sync']);

        $maxTs = CarbonImmutable::parse('2026-05-01T12:00:00Z');

        $job = new BridgePersistReplicaPageJob(
            dataset: 'stellar',
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
            cursorPatch: new BridgeReplicaCursorPatch(
                applyReplicationState: true,
                replicationNextUrl: null,
                replicationInProgress: false,
                maxBridgeTs: $maxTs,
            ),
        );

        $job->handle(
            app(BridgeSyncService::class),
            app(BridgeRateLimitGuard::class),
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

        $cursor = ListingSyncCursor::query()->where('dataset_slug', 'stellar')->first();
        $this->assertNotNull($cursor);
        $this->assertNull($cursor->replication_next_url);
        $this->assertFalse($cursor->replication_in_progress);
        $this->assertNotNull($cursor->last_bridge_modification_timestamp);
    }

    public function test_persist_job_dispatches_incremental_fetch_after_replication_completes(): void
    {
        config(['bridge.sync_queue' => 'bridge-sync']);
        Queue::fake();

        $maxTs = CarbonImmutable::parse('2026-05-02T08:00:00Z');

        $job = new BridgePersistReplicaPageJob(
            dataset: 'stellar',
            rows: [
                [
                    'ListingKey' => 'STELLAR-400',
                    'StandardStatus' => 'Active',
                    'BridgeModificationTimestamp' => $maxTs->toIso8601String(),
                    'ListPrice' => 300000,
                ],
            ],
            cursorPatch: new BridgeReplicaCursorPatch(
                applyReplicationState: true,
                replicationNextUrl: null,
                replicationInProgress: false,
                maxBridgeTs: $maxTs,
            ),
            dispatchIncrementalAfter: true,
        );

        $job->handle(
            app(BridgeSyncService::class),
            app(BridgeRateLimitGuard::class),
        );

        Queue::assertPushed(BridgeSyncFetchPageJob::class, function (BridgeSyncFetchPageJob $fetch): bool {
            return $fetch->dataset === 'stellar' && $fetch->mode === 'incremental';
        });

        $this->assertSame(1, Listing::query()->where('listing_key', 'STELLAR-400')->count());
    }
}
