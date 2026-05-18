<?php

namespace Tests\Unit\Bridge;

use App\Events\Bridge\BridgeReplicationPageFetched;
use App\Events\Bridge\BridgeReplicationPagePersisted;
use App\Services\Bridge\BridgeReplicaPersistStats;
use App\Services\Bridge\BridgeSyncTelemetry;
use Illuminate\Support\Facades\Event;
use Tests\TestCase;

class BridgeSyncTelemetryTest extends TestCase
{
    public function test_status_counts_from_rows_groups_standard_status(): void
    {
        $rows = [
            ['StandardStatus' => 'Active'],
            ['StandardStatus' => 'Pending'],
            ['StandardStatus' => 'Active'],
            ['StandardStatus' => 'Closed'],
        ];

        $counts = BridgeSyncTelemetry::statusCountsFromRows($rows);

        $this->assertSame([
            'active' => 2,
            'closed' => 1,
            'pending' => 1,
        ], $counts);
    }

    public function test_sanitize_bridge_url_strips_access_token(): void
    {
        $sanitized = BridgeSyncTelemetry::sanitizeBridgeUrl(
            'https://bridge.test/OData/stellar/Property/replication?access_token=secret&$top=10',
        );

        $this->assertStringNotContainsString('secret', $sanitized);
        $this->assertStringContainsString('top=10', $sanitized);
    }

    public function test_record_page_fetched_dispatches_event_with_payload(): void
    {
        Event::fake([BridgeReplicationPageFetched::class]);

        $telemetry = new BridgeSyncTelemetry;
        $telemetry->recordPageFetched(
            dataset: 'stellar',
            mode: 'replication',
            bridgeUrl: 'https://bridge.test/OData/stellar/Property/replication',
            odataQuery: ['$filter' => "(StandardStatus eq 'Active')", '$top' => 2000],
            httpStatus: 200,
            listingsDownloaded: 3,
            statusCounts: ['active' => 3],
            replicationStarting: true,
            hasNextPage: true,
            chainDepth: 0,
        );

        Event::assertDispatched(BridgeReplicationPageFetched::class, function (BridgeReplicationPageFetched $event): bool {
            return $event->dataset === 'stellar'
                && $event->mode === 'replication'
                && $event->listingsDownloaded === 3
                && $event->statusCounts['active'] === 3
                && $event->odataQuery['$filter'] === "(StandardStatus eq 'Active')";
        });
    }

    public function test_record_page_persisted_dispatches_event_with_stats(): void
    {
        Event::fake([BridgeReplicationPagePersisted::class]);

        $telemetry = new BridgeSyncTelemetry;
        $stats = new BridgeReplicaPersistStats(
            rowsReceived: 10,
            upserted: 8,
            deleted: 2,
            skipped: 0,
            durationMs: 50,
        );

        $telemetry->recordPagePersisted('stellar', $stats, chunkIndex: 1, chunkTotal: 2);

        Event::assertDispatched(BridgeReplicationPagePersisted::class, function (BridgeReplicationPagePersisted $event): bool {
            return $event->dataset === 'stellar'
                && $event->stats->upserted === 8
                && $event->stats->deleted === 2
                && $event->chunkIndex === 1
                && $event->chunkTotal === 2;
        });
    }
}
