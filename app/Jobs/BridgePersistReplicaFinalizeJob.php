<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeRateLimitGuard;
use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Applies cursor state and chains the next fetch when a Bridge page returns zero rows.
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
    ) {}

    public function handle(BridgeSyncService $sync, BridgeRateLimitGuard $rateLimitGuard): void
    {
        if ($this->cursorPatch !== null) {
            $sync->applyCursorPatch($this->dataset, $this->cursorPatch);
        }

        $queue = (string) config('bridge.sync_queue', 'bridge-sync');
        $delay = $rateLimitGuard->delaySecondsForNextFetch();

        if ($this->dispatchIncrementalAfter) {
            BridgeSyncFetchPageJob::dispatch($this->dataset, 'incremental', 0, 0)
                ->onQueue($queue)
                ->delay(now()->addSeconds($delay));

            return;
        }

        if ($this->nextFetchMode !== null) {
            BridgeSyncFetchPageJob::dispatch(
                $this->dataset,
                $this->nextFetchMode,
                $this->nextIncrementalSkip,
                $this->nextChainDepth,
            )
                ->onQueue($queue)
                ->delay(now()->addSeconds($delay));
        }
    }
}
