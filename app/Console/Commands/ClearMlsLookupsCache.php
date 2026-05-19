<?php

namespace App\Console\Commands;

use App\Services\Mls\MlsFeedResolver;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;

class ClearMlsLookupsCache extends Command
{
    protected $signature = 'mls:clear-lookups-cache
                            {--dataset= : Only clear cached lookups for this dataset (e.g. stellar)}
                            {--all : Clear all cached lookup responses}';

    /** @var list<string> */
    protected $aliases = ['bridge:clear-lookups-cache'];

    protected $description = 'Clear cached MLS Lookup API responses from mls_search_cache';

    public function handle(MlsFeedResolver $feeds): int
    {
        if (! $this->option('all') && ! $this->option('dataset')) {
            $this->error('Specify --all or --dataset=<dataset>');

            return self::INVALID;
        }

        $table = (string) config('mls.search_cache_table', 'mls_search_cache');

        if ($this->option('all')) {
            $deleted = DB::table($table)
                ->where('partition_key', 'like', 'lookups:%')
                ->delete();

            $this->info("Deleted {$deleted} cached lookup response(s).");

            return self::SUCCESS;
        }

        $dataset = (string) $this->option('dataset');
        $catalogKey = $feeds->normalizeWireDatasetToCatalogKey($dataset);
        $deleted = DB::table($table)
            ->where('partition_key', 'lookups:'.$catalogKey)
            ->delete();

        $this->info("Deleted {$deleted} cached lookup response(s) for dataset={$dataset} (partition lookups:{$catalogKey}).");

        return self::SUCCESS;
    }
}
