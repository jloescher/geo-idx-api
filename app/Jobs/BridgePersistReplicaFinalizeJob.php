<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Bridge\BridgeSyncFetchScheduler;
use App\Services\Bridge\BridgeSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Applies cursor state after a persist batch completes and schedules the next rate-limited fetch.
 */
class BridgePersistReplicaFinalizeJob implements ShouldQueue
{
    use Queueable;

    public int $timeout = 60;

    public function __construct(
        public string $dataset,
        public ?int $replicaPageId = null,
        public ?BridgeReplicaCursorPatch $cursorPatch = null,
        public bool $dispatchIncrementalAfter = false,
        public ?string $nextFetchMode = null,
        public int $nextIncrementalSkip = 0,
        public int $nextChainDepth = 0,
    ) {
        $this->onQueue((string) config('bridge.sync_persist_queue', 'bridge-sync-persist'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['bridge-replication', 'dataset:'.$this->dataset, 'persist-finalize'];
    }

    public function handle(
        BridgeSyncService $sync,
        BridgeSyncFetchScheduler $scheduler,
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
                $this->nextIncrementalSkip,
                $this->nextChainDepth,
            );
        }
    }
}
