<?php

declare(strict_types=1);

namespace App\Services\Spark;

use App\Jobs\SparkSyncFetchPageJob;

final class SparkSyncFetchScheduler
{
    public function __construct(
        private readonly SparkRateLimitGuard $rateLimitGuard,
    ) {}

    public function dispatchNext(string $dataset, string $mode, int $incrementalSkip = 0, int $chainDepth = 0): void
    {
        $queue = (string) config('spark.sync_fetch_queue', 'spark-sync-fetch');
        $delayMs = $this->rateLimitGuard->delayMillisecondsForNextFetch();

        SparkSyncFetchPageJob::dispatch($dataset, $mode, $incrementalSkip, $chainDepth)
            ->onQueue($queue)
            ->delay(now()->addMilliseconds($delayMs));
    }

    public function dispatchIncremental(string $dataset): void
    {
        $this->dispatchNext($dataset, 'incremental', 0, 0);
    }
}
