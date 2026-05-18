<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Persists one chunk of a Bridge page to Postgres — no Bridge HTTP or fetch chaining.
 */
class BridgePersistReplicaChunkJob implements ShouldQueue
{
    use Queueable;

    public int $timeout = 300;

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    public function __construct(
        public string $dataset,
        public array $rows,
    ) {
        $this->onQueue((string) config('bridge.sync_persist_queue', 'bridge-sync-persist'));
    }

    public function handle(BridgeSyncService $sync): void
    {
        $sync->persistChunk($this->dataset, $this->rows);
    }
}
