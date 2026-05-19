<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeSyncService;
use App\Services\Mls\MlsDatasetRegistry;
use App\Services\Replication\ReplicaPageStore;
use App\Services\Replication\ReplicationFreshness;
use App\Services\Spark\SparkSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\DB;

/**
 * Revenue impact: catch-up kickoff only when mirror is stale keeps workers busy without
 * wasting MLS quota on 15-minute polls during initial replication.
 */
class MlsReplicationKickoffJob implements ShouldQueue
{
    use Queueable;

    public int $timeout = 60;

    public function __construct()
    {
        $this->onQueue('default');
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['mls-replication', 'kickoff'];
    }

    public function handle(
        MlsDatasetRegistry $registry,
        ReplicationFreshness $freshness,
        ReplicaPageStore $pageStore,
        BridgeSyncService $bridgeSync,
        SparkSyncService $sparkSync,
    ): void {
        if (DB::connection()->getDriverName() !== 'pgsql') {
            return;
        }

        foreach ($registry->datasets() as $slug => $def) {
            $provider = (string) ($def['provider'] ?? 'bridge');
            if ($freshness->mode($slug, $provider) !== ReplicationFreshness::MODE_CATCH_UP) {
                continue;
            }

            if ($pageStore->hasActivePage($slug, $provider)) {
                continue;
            }

            $sync = $provider === 'spark' ? $sparkSync : $bridgeSync;
            $cursor = $sync->cursorForDataset($slug);
            $fetchQueue = (string) ($def['fetch_queue'] ?? 'default');

            if ($sync->shouldRunReplication($cursor)) {
                $job = $provider === 'spark'
                    ? new SparkSyncFetchPageJob($slug, 'replication', 0, 0)
                    : new BridgeSyncFetchPageJob($slug, 'replication', 0, 0);
                dispatch($job)->onQueue($fetchQueue);

                continue;
            }

            if ($sync->shouldRunIncremental($cursor)) {
                $job = $provider === 'spark'
                    ? new SparkSyncFetchPageJob($slug, 'incremental', 0, 0)
                    : new BridgeSyncFetchPageJob($slug, 'incremental', 0, 0);
                dispatch($job)->onQueue($fetchQueue);
            }
        }
    }
}
