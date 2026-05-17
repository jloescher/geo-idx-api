<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\DB;

/**
 * Revenue impact: periodic kickoff dispatches rate-limited fetch jobs without blocking workers
 * on multi-page replication (see BridgeSyncFetchPageJob).
 */
class BridgeSyncJob implements ShouldQueue
{
    use Queueable;

    public int $timeout = 60;

    public function __construct()
    {
        $this->onQueue((string) config('bridge.sync_queue', 'bridge-sync'));
    }

    public function handle(BridgeSyncService $sync): void
    {
        if (DB::connection()->getDriverName() !== 'pgsql') {
            return;
        }

        $datasets = config('bridge.datasets', ['stellar']);
        $list = is_array($datasets) ? array_values(array_filter(array_map(trim(...), $datasets))) : ['stellar'];
        $queue = (string) config('bridge.sync_queue', 'bridge-sync');

        foreach ($list as $dataset) {
            if (! is_string($dataset) || $dataset === '') {
                continue;
            }

            $cursor = $sync->cursorForDataset($dataset);

            if ($sync->shouldRunReplication($cursor)) {
                BridgeSyncFetchPageJob::dispatch($dataset, 'replication', 0, 0)->onQueue($queue);

                continue;
            }

            if ($sync->shouldRunIncremental($cursor)) {
                BridgeSyncFetchPageJob::dispatch($dataset, 'incremental', 0, 0)->onQueue($queue);
            }
        }
    }
}
