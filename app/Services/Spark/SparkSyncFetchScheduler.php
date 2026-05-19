<?php

declare(strict_types=1);

namespace App\Services\Spark;

use App\Jobs\SparkSyncFetchPageJob;
use App\Services\Replication\ReplicationFreshness;

/**
 * Revenue impact: steady-state 15-minute idle spacing caps Spark replication egress after catch-up.
 */
final class SparkSyncFetchScheduler
{
    public function __construct(
        private readonly SparkRateLimitGuard $rateLimitGuard,
        private readonly ReplicationFreshness $freshness,
    ) {}

    public function dispatchNext(string $dataset, string $mode, int $incrementalSkip = 0, int $chainDepth = 0, bool $idlePoll = false): void
    {
        $queue = (string) config('spark.sync_fetch_queue', 'spark-sync-fetch');
        $delayMs = $this->resolveDelayMs($dataset, $chainDepth, $idlePoll);

        SparkSyncFetchPageJob::dispatch($dataset, $mode, $incrementalSkip, $chainDepth)
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

        if ($idlePoll && $this->freshness->mode($dataset, 'spark') === ReplicationFreshness::MODE_STEADY) {
            return max($rateLimitMs, $this->freshness->steadyPollMinutes() * 60_000);
        }

        return $rateLimitMs;
    }
}
