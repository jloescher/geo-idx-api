<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Spark\SparkSyncFetchScheduler;
use App\Services\Spark\SparkSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

class SparkPersistReplicaFinalizeJob implements ShouldQueue
{
    use Queueable;

    public int $timeout = 60;

    public function __construct(
        public string $dataset,
        public ?int $replicaPageId = null,
        public ?BridgeReplicaCursorPatch $cursorPatch = null,
        public bool $dispatchIncrementalAfter = false,
        public ?string $nextFetchMode = null,
        public int $nextChainDepth = 0,
    ) {
        $this->onQueue((string) config('spark.sync_persist_queue', 'spark-sync-persist'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['spark-replication', 'dataset:'.$this->dataset, 'persist-finalize'];
    }

    public function handle(
        SparkSyncService $sync,
        SparkSyncFetchScheduler $scheduler,
        BridgeReplicaPageStore $pageStore,
    ): void {
        if ($this->cursorPatch !== null) {
            $sync->applyCursorPatch($this->dataset, $this->cursorPatch);
        }

        if ($this->replicaPageId !== null) {
            $pageStore->markCompleted($this->replicaPageId);
            $pageStore->deletePage($this->replicaPageId);
        }

        if ($this->dispatchIncrementalAfter) {
            $scheduler->dispatchIncremental($this->dataset);

            return;
        }

        if ($this->nextFetchMode !== null) {
            $scheduler->dispatchNext(
                $this->dataset,
                $this->nextFetchMode,
                0,
                $this->nextChainDepth,
            );
        }
    }
}
