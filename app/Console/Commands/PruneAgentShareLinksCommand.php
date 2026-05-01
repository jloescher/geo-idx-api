<?php

namespace App\Console\Commands;

use App\Models\AgentShareLink;
use Illuminate\Console\Attributes\Description;
use Illuminate\Console\Attributes\Signature;
use Illuminate\Console\Command;

#[Signature('agent:prune-share-links {--days= : Prune inactive links older than N days (defaults to config)} {--dry-run : List candidate rows without deleting}')]
#[Description('Prune old inactive agent share links')]
class PruneAgentShareLinksCommand extends Command
{
    /**
     * Execute the console command.
     */
    public function handle(): int
    {
        $daysOption = $this->option('days');
        $days = is_numeric($daysOption) ? (int) $daysOption : (int) config('agent_portal.share_links.prune_days', 90);
        if ($days <= 0) {
            $this->error('The --days option must be greater than zero.');

            return self::FAILURE;
        }

        $cutoff = now()->subDays($days);
        $query = AgentShareLink::query()
            ->whereNotNull('expires_at', 'and')
            ->where('expires_at', '<=', $cutoff);

        if ($this->option('dry-run')) {
            $count = (int) $query->count();
            $this->info(sprintf('Dry run: would prune %d inactive share link(s) with expires_at on or before %s (%d day window).', $count, $cutoff->toIso8601String(), $days));

            return self::SUCCESS;
        }

        $deleted = $query->delete();

        $this->info(sprintf('Pruned %d inactive share link(s) older than %d days.', $deleted, $days));

        return self::SUCCESS;
    }
}
