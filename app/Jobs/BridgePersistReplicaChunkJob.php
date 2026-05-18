<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeRateLimitGuard;
use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Revenue impact: bounded row batches keep worker RAM under limit while replication pages
 * stream to Postgres; cursor advancement runs only on the final chunk of each page.
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
        public ?BridgeReplicaCursorPatch $cursorPatch = null,
        public bool $dispatchIncrementalAfter = false,
        public ?string $nextFetchMode = null,
        public int $nextIncrementalSkip = 0,
        public int $nextChainDepth = 0,
    ) {}

    public function handle(BridgeSyncService $sync, BridgeRateLimitGuard $rateLimitGuard): void
    {
        $sync->persistChunk($this->dataset, $this->rows, $this->cursorPatch);

        if ($this->cursorPatch === null && ! $this->dispatchIncrementalAfter && $this->nextFetchMode === null) {
            return;
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
