<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaCursorPatch;
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
        public ?BridgeReplicaCursorPatch $cursorPatch = null,
        public bool $dispatchIncrementalAfter = false,
        public ?string $nextFetchMode = null,
        public int $nextIncrementalSkip = 0,
        public int $nextChainDepth = 0,
    ) {
        $this->onQueue((string) config('bridge.sync_persist_queue', 'bridge-sync-persist'));
    }

    public function handle(BridgeSyncService $sync, BridgeSyncFetchScheduler $scheduler): void
    {
        if ($this->cursorPatch !== null) {
            $sync->applyCursorPatch($this->dataset, $this->cursorPatch);
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
