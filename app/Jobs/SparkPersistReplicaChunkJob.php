<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeSyncTelemetry;
use App\Services\Replication\ReplicaPageStore;
use App\Services\Spark\SparkSyncService;
use Illuminate\Bus\Batchable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\Log;

class SparkPersistReplicaChunkJob implements ShouldQueue
{
    use Batchable;
    use Queueable;

    public int $timeout = 600;

    public function __construct(
        public string $dataset,
        public int $pageId,
        public int $chunkIndex,
        public int $chunkTotal,
    ) {
        $this->onQueue((string) config('spark.sync_persist_queue', 'spark-sync-persist'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['spark-replication', 'dataset:'.$this->dataset, 'persist-chunk'];
    }

    public function handle(
        SparkSyncService $sync,
        BridgeSyncTelemetry $telemetry,
        ReplicaPageStore $pageStore,
    ): void {
        $rows = $pageStore->rowsForChunk($this->pageId, $this->chunkIndex, $this->chunkTotal);
        $stats = $sync->persistChunk($this->dataset, $rows);

        $telemetry->recordPagePersisted(
            dataset: $this->dataset,
            stats: $stats,
            chunkIndex: $this->chunkIndex,
            chunkTotal: $this->chunkTotal,
        );

        $peakMb = round(memory_get_peak_usage(true) / 1024 / 1024, 2);
        Log::info('replication.persist.memory', [
            'provider' => 'spark',
            'dataset' => $this->dataset,
            'chunk_index' => $this->chunkIndex,
            'chunk_total' => $this->chunkTotal,
            'row_count' => count($rows),
            'peak_mb' => $peakMb,
        ]);

        unset($rows);
        gc_collect_cycles();
    }
}
