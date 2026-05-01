<?php

namespace App\Console\Commands;

use App\Services\AgentPortal\AlertSchedulerService;
use Illuminate\Console\Attributes\Description;
use Illuminate\Console\Attributes\Signature;
use Illuminate\Console\Command;

#[Signature('agent:process-due-alerts')]
#[Description('Process all alerts that are due to run and schedule their next run')]
class ProcessDueAlertsCommand extends Command
{
    public function handle(AlertSchedulerService $scheduler): int
    {
        $count = $scheduler->processDueAlerts();
        $this->info("Processed {$count} due alert(s).");

        return self::SUCCESS;
    }
}
