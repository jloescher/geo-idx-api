<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

/**
 * Revenue impact: 15‑minute ingestion keeps Postgres mirror monetization-ready without exceeding
 * Bridge $top pagination limits (follow docs/bridge_interactive/reso_web_api.md).
 */
class BridgeSyncJob implements ShouldQueue
{
    use Dispatchable;
    use InteractsWithQueue;
    use Queueable;
    use SerializesModels;

    public int $timeout = 600;

    public function handle(BridgeSyncService $sync): void
    {
        $sync->syncAllDatasets();
    }
}
