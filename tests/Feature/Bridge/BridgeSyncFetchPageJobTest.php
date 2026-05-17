<?php

namespace Tests\Feature\Bridge;

use App\Jobs\BridgePersistReplicaPageJob;
use App\Jobs\BridgeSyncFetchPageJob;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\BridgeRateLimitGuard;
use App\Services\Bridge\BridgeSyncService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Queue;
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
            'bridge.sync_queue' => 'bridge-sync',
            'bridge.sync_include_media' => false,
            'bridge.sync_replication_top' => 2000,
        ]);
    }

    public function test_replication_fetch_dispatches_persist_job_and_chains_fetch_after_persist_not_in_fetch_handler(): void
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

        Queue::fake();

        $job = new BridgeSyncFetchPageJob('stellar', 'replication', 0, 0);
        $job->handle(app(BridgeSyncService::class));

        Queue::assertPushed(BridgePersistReplicaPageJob::class, function (BridgePersistReplicaPageJob $persist): bool {
            return $persist->dataset === 'stellar'
                && count($persist->rows) === 1
                && $persist->nextFetchMode === 'replication'
                && $persist->nextChainDepth === 1
                && $persist->cursorPatch?->replicationNextUrl === 'https://bridge.test/OData/stellar/Property/replication?page=2';
        });

        Queue::assertNotPushed(BridgeSyncFetchPageJob::class);
    }

    public function test_persist_job_chains_next_replication_fetch_after_cursor_is_written(): void
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

        Queue::fake();

        (new BridgeSyncFetchPageJob('stellar', 'replication', 0, 0))
            ->handle(app(BridgeSyncService::class));

        $persistJob = null;
        Queue::assertPushed(BridgePersistReplicaPageJob::class, function (BridgePersistReplicaPageJob $persist) use (&$persistJob): bool {
            $persistJob = $persist;

            return true;
        });
        $this->assertInstanceOf(BridgePersistReplicaPageJob::class, $persistJob);

        Queue::fake();

        $persistJob->handle(
            app(BridgeSyncService::class),
            app(BridgeRateLimitGuard::class),
        );

        Queue::assertPushed(BridgeSyncFetchPageJob::class, function (BridgeSyncFetchPageJob $fetch): bool {
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
