<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use App\Jobs\BridgeSyncFetchPageJob;
use App\Services\Replication\ReplicationFreshness;

/**
 * Sole dispatcher for rate-limited Bridge HTTP fetch jobs (replication / incremental pages).
 *
 * Revenue impact: steady-state 15-minute idle spacing caps Bridge egress after mirror is current.
 */
final class BridgeSyncFetchScheduler
{
    public function __construct(
        private readonly BridgeRateLimitGuard $rateLimitGuard,
        private readonly ReplicationFreshness $freshness,
    ) {}

    public function dispatchNext(string $dataset, string $mode, int $incrementalSkip = 0, int $chainDepth = 0, bool $idlePoll = false): void
    {
        $queue = (string) config('bridge.sync_fetch_queue', 'bridge-sync-fetch');
        $delayMs = $this->resolveDelayMs($dataset, $chainDepth, $idlePoll);

        BridgeSyncFetchPageJob::dispatch($dataset, $mode, $incrementalSkip, $chainDepth)
            ->onQueue($queue)
            ->delay(now()->addMilliseconds($delayMs));
    }

    public function dispatchIncremental(string $dataset, bool $idlePoll = false): void
    {
        $this->dispatchNext($dataset, 'incremental', 0, 0, $idlePoll);
    }

    private function resolveDelayMs(string $dataset, int $chainDepth, bool $idlePoll): int
    {
        $rateLimitMs = $this->rateLimitGuard->delayMillisecondsForNextFetch();

        if ($chainDepth > 0) {
            return $rateLimitMs;
        }

        if ($idlePoll && $this->freshness->mode($dataset, 'bridge') === ReplicationFreshness::MODE_STEADY) {
            return max($rateLimitMs, $this->freshness->steadyPollMinutes() * 60_000);
        }

        return $rateLimitMs;
    }
}
