<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Spark\SparkSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\DB;

class SparkSyncJob implements ShouldQueue
{
    use Queueable;

    public int $timeout = 60;

    public function __construct()
    {
        $this->onQueue((string) config('spark.sync_fetch_queue', 'spark-sync-fetch'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['spark-replication', 'kickoff'];
    }

    public function handle(SparkSyncService $sync, BridgeReplicaPageStore $pageStore): void
    {
        if (DB::connection()->getDriverName() !== 'pgsql') {
            return;
        }

        $datasets = config('spark.datasets', ['beaches']);
        $list = is_array($datasets) ? array_values(array_filter(array_map(trim(...), $datasets))) : ['beaches'];
        $queue = (string) config('spark.sync_fetch_queue', 'spark-sync-fetch');

        foreach ($list as $dataset) {
            if (! is_string($dataset) || $dataset === '') {
                continue;
            }

            $cursor = $sync->cursorForDataset($dataset);

            if ($pageStore->hasActivePage($dataset, 'spark')) {
                continue;
            }

            if ($sync->shouldRunReplication($cursor)) {
                SparkSyncFetchPageJob::dispatch($dataset, 'replication', 0, 0)->onQueue($queue);

                continue;
            }

            if ($sync->shouldRunIncremental($cursor)) {
                SparkSyncFetchPageJob::dispatch($dataset, 'incremental', 0, 0)->onQueue($queue);
            }
        }
    }
}
