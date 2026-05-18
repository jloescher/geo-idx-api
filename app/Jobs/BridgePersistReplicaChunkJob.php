<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeSyncService;
use App\Services\Bridge\BridgeSyncTelemetry;
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
        public ?int $chunkIndex = null,
        public ?int $chunkTotal = null,
    ) {
        $this->onQueue((string) config('bridge.sync_persist_queue', 'bridge-sync-persist'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['bridge-replication', 'dataset:'.$this->dataset, 'persist-chunk'];
    }

    public function handle(BridgeSyncService $sync, BridgeSyncTelemetry $telemetry): void
    {
        $stats = $sync->persistChunk($this->dataset, $this->rows);

        $telemetry->recordPagePersisted(
            dataset: $this->dataset,
            stats: $stats,
            chunkIndex: $this->chunkIndex,
            chunkTotal: $this->chunkTotal,
        );
    }
}
