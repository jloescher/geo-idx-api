<?php

namespace Tests\Unit\Bridge;

use App\Services\Bridge\BridgeReplicaPersistStats;
use Tests\TestCase;

class BridgeReplicaPersistStatsTest extends TestCase
{
    public function test_to_array_exposes_persist_counters(): void
    {
        $stats = new BridgeReplicaPersistStats(
            rowsReceived: 100,
            upserted: 90,
            deleted: 8,
            skipped: 2,
            durationMs: 120,
        );

        $this->assertSame([
            'rows_received' => 100,
            'upserted' => 90,
            'deleted' => 8,
            'skipped' => 2,
            'duration_ms' => 120,
        ], $stats->toArray());
    }
}
