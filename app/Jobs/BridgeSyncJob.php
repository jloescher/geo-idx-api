<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Bridge\BridgeSyncService;
use App\Services\Bridge\BridgeSyncTelemetry;
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
        $this->onQueue((string) config('bridge.sync_fetch_queue', 'bridge-sync-fetch'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['bridge-replication', 'kickoff'];
    }

    public function handle(
        BridgeSyncService $sync,
        BridgeSyncTelemetry $telemetry,
        BridgeReplicaPageStore $pageStore,
    ): void {
        if (DB::connection()->getDriverName() !== 'pgsql') {
            return;
        }

        $datasets = config('bridge.datasets', ['stellar']);
        $list = is_array($datasets) ? array_values(array_filter(array_map(trim(...), $datasets))) : ['stellar'];
        $queue = (string) config('bridge.sync_fetch_queue', 'bridge-sync-fetch');
        $persistQueue = (string) config('bridge.sync_persist_queue', 'bridge-sync-persist');

        $telemetry->recordKickoff($list, $queue, $persistQueue);

        foreach ($list as $dataset) {
            if (! is_string($dataset) || $dataset === '') {
                continue;
            }

            $cursor = $sync->cursorForDataset($dataset);

            if ($pageStore->hasActivePage($dataset)) {
                continue;
            }

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
