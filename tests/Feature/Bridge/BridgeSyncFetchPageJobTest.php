<?php

namespace Tests\Feature\Bridge;

use App\Jobs\BridgePersistReplicaChunkJob;
use App\Jobs\BridgeSyncFetchPageJob;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\BridgeRateLimitGuard;
use App\Services\Bridge\BridgeSyncService;
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
            'bridge.sync_queue' => 'bridge-sync',
            'bridge.sync_include_media' => false,
            'bridge.sync_replication_top' => 2000,
            'bridge.sync_persist_job_chunk_size' => 100,
        ]);
    }

    public function test_replication_fetch_dispatches_chained_persist_chunks_not_fetch(): void
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

        Bus::assertChained([
            function (BridgePersistReplicaChunkJob $persist): bool {
                return $persist->dataset === 'stellar'
                    && count($persist->rows) === 1
                    && $persist->nextFetchMode === 'replication'
                    && $persist->nextChainDepth === 1
                    && $persist->cursorPatch?->replicationNextUrl === 'https://bridge.test/OData/stellar/Property/replication?page=2';
            },
        ]);
    }

    public function test_last_persist_chunk_chains_next_replication_fetch_after_cursor_is_written(): void
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

        $persistJob = null;
        Bus::assertChained([
            function (BridgePersistReplicaChunkJob $persist) use (&$persistJob): bool {
                $persistJob = $persist;

                return true;
            },
        ]);
        $this->assertInstanceOf(BridgePersistReplicaChunkJob::class, $persistJob);

        Bus::fake();

        $persistJob->handle(
            app(BridgeSyncService::class),
            app(BridgeRateLimitGuard::class),
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
