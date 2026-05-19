<?php

namespace App\Console\Commands;

use App\Services\Replication\ReplicaPageStore;
use Illuminate\Console\Command;

class PurgeReplicaPagesCommand extends Command
{
    protected $signature = 'mls:purge-replica-pages';

    /** @var list<string> */
    protected $aliases = ['bridge:purge-replica-pages'];

    protected $description = 'Delete old completed, failed, and abandoned MLS replica staging pages';

    public function handle(ReplicaPageStore $store): int
    {
        $deleted = $store->purgeEligibleRows();
        $this->info("Purged {$deleted} replica_pages row(s).");

        return self::SUCCESS;
    }
}
