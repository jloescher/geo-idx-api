<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Bridge\BridgeSyncService;
use App\Services\Bridge\BridgeSyncTelemetry;
use Illuminate\Bus\Batchable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Persists one chunk of a staged Bridge page to Postgres — no Bridge HTTP or fetch chaining.
 */
class BridgePersistReplicaChunkJob implements ShouldQueue
{
    use Batchable;
    use Queueable;

    public int $timeout = 300;

    public function __construct(
        public string $dataset,
        public int $pageId,
        public int $chunkIndex,
        public int $chunkTotal,
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

    public function handle(
        BridgeSyncService $sync,
        BridgeSyncTelemetry $telemetry,
        BridgeReplicaPageStore $pageStore,
    ): void {
        $rows = $pageStore->rowsForChunk($this->pageId, $this->chunkIndex, $this->chunkTotal);
        $stats = $sync->persistChunk($this->dataset, $rows);

        $telemetry->recordPagePersisted(
            dataset: $this->dataset,
            stats: $stats,
            chunkIndex: $this->chunkIndex,
            chunkTotal: $this->chunkTotal,
        );
    }
}
