<?php

namespace App\Console\Commands;

use App\Jobs\MlsReplicationKickoffJob;
use Illuminate\Console\Command;

class MlsReplicationKickoffCommand extends Command
{
    protected $signature = 'mls:replication-kickoff';

    protected $description = 'Dispatch replication/incremental fetch jobs for all catch-up MLS datasets';

    public function handle(): int
    {
        MlsReplicationKickoffJob::dispatch();
        $this->info('Dispatched MLS replication kickoff job.');

        return self::SUCCESS;
    }
}
