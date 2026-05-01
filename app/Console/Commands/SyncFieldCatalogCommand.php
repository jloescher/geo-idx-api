<?php

namespace App\Console\Commands;

use App\Services\AgentPortal\Contracts\LookupProviderInterface;
use App\Services\AgentPortal\FieldCatalogService;
use Illuminate\Console\Attributes\Description;
use Illuminate\Console\Attributes\Signature;
use Illuminate\Console\Command;

#[Signature('agent:sync-field-catalog {--mls= : Specific MLS code to sync} {--dataset= : Specific dataset code to sync}')]
#[Description('Sync MLS field catalog from Bridge lookups into the database')]
class SyncFieldCatalogCommand extends Command
{
    public function handle(FieldCatalogService $catalogService, LookupProviderInterface $lookups): int
    {
        $mls = (string) $this->option('mls');
        $dataset = (string) $this->option('dataset');

        if ($mls !== '' && $dataset !== '') {
            $this->info("Syncing field catalog for {$mls}/{$dataset}...");
            $count = $catalogService->syncFieldCatalog($mls, $dataset);
            $this->info("Synced {$count} field entries.");

            return self::SUCCESS;
        }

        $this->warn('Specify --mls and --dataset to sync a specific MLS dataset.');

        return self::SUCCESS;
    }
}
