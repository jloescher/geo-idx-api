<?php

namespace App\Console\Commands;

use App\Support\BridgeSyncQueueNames;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;

class BridgeMigrateLegacyQueueJobsCommand extends Command
{
    protected $signature = 'bridge:migrate-legacy-queue-jobs {--dry-run : Report counts without updating rows}';

    protected $description = 'Move database queue jobs from legacy bridge-sync to bridge-sync-fetch';

    public function handle(): int
    {
        $target = BridgeSyncQueueNames::fetchQueue();
        $migrated = 0;

        foreach (BridgeSyncQueueNames::legacyFetchQueueNames() as $legacy) {
            $count = DB::table('jobs')->where('queue', $legacy)->count();
            if ($count === 0) {
                continue;
            }

            if ($this->option('dry-run')) {
                $this->line("Would move {$count} job(s) from {$legacy} → {$target}.");

                continue;
            }

            $migrated += DB::table('jobs')
                ->where('queue', $legacy)
                ->update(['queue' => $target]);
        }

        if ($this->option('dry-run')) {
            return self::SUCCESS;
        }

        $this->info("Migrated {$migrated} job(s) to queue {$target}.");

        return self::SUCCESS;
    }
}
