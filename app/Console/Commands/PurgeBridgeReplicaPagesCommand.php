<?php

namespace App\Console\Commands;

use App\Services\Bridge\BridgeReplicaPageStore;
use Illuminate\Console\Command;

class PurgeBridgeReplicaPagesCommand extends Command
{
    protected $signature = 'bridge:purge-replica-pages';

    protected $description = 'Delete old completed, failed, and abandoned Bridge replica staging pages';

    public function handle(BridgeReplicaPageStore $store): int
    {
        $deleted = $store->purgeEligibleRows();
        $this->info("Purged {$deleted} bridge_replica_pages row(s).");

        return self::SUCCESS;
    }
}
