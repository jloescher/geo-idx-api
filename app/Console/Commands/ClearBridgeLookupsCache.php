<?php

namespace App\Console\Commands;

use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;

class ClearBridgeLookupsCache extends Command
{
    protected $signature = 'bridge:clear-lookups-cache
                            {--dataset= : Only clear cached lookups for this dataset (e.g. stellar)}
                            {--all : Clear all cached lookup responses}';

    protected $description = 'Clear cached Bridge Lookup API responses from bridge_search_cache';

    public function handle(): int
    {
        if (! $this->option('all') && ! $this->option('dataset')) {
            $this->error('Specify --all or --dataset=<dataset>');

            return self::INVALID;
        }

        if ($this->option('all')) {
            $deleted = DB::table('bridge_search_cache')
                ->where('partition_key', 'like', 'lookups:%')
                ->delete();

            $this->info("Deleted {$deleted} cached lookup response(s).");

            return self::SUCCESS;
        }

        $dataset = (string) $this->option('dataset');
        $deleted = DB::table('bridge_search_cache')
            ->where('partition_key', 'lookups:'.$dataset)
            ->delete();

        $this->info("Deleted {$deleted} cached lookup response(s) for dataset={$dataset}.");

        return self::SUCCESS;
    }
}
