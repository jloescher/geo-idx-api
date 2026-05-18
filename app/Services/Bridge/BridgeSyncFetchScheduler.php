<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use App\Jobs\BridgeSyncFetchPageJob;

/**
 * Sole dispatcher for rate-limited Bridge HTTP fetch jobs (replication / incremental pages).
 */
final class BridgeSyncFetchScheduler
{
    public function __construct(
        private readonly BridgeRateLimitGuard $rateLimitGuard,
    ) {}

    public function dispatchNext(string $dataset, string $mode, int $incrementalSkip = 0, int $chainDepth = 0): void
    {
        $queue = (string) config('bridge.sync_fetch_queue', 'bridge-sync-fetch');
        $delay = $this->rateLimitGuard->delaySecondsForNextFetch();

        BridgeSyncFetchPageJob::dispatch($dataset, $mode, $incrementalSkip, $chainDepth)
            ->onQueue($queue)
            ->delay(now()->addSeconds($delay));
    }

    public function dispatchIncremental(string $dataset): void
    {
        $this->dispatchNext($dataset, 'incremental', 0, 0);
    }
}
