<?php

namespace App\Console\Commands;

use App\Models\GisCache;
use App\Models\GisSourceState;
use Illuminate\Console\Command;

class GisClearCacheCommand extends Command
{
    protected $signature = 'gis:clear-cache
                            {--source= : Only invalidate rows for this source_used / source_key (e.g. pinellas_enterprise_parcels)}
                            {--all : Truncate all GIS cache rows and bump every source generation}';

    protected $description = 'Invalidate GIS origin cache (Postgres) and bump source generations so edge caches go stale';

    public function handle(): int
    {
        if (! $this->option('all') && ! $this->option('source')) {
            $this->error('Specify --all or --source=<source_key>');

            return self::INVALID;
        }

        if ($this->option('all')) {
            GisCache::query()->delete();
            GisSourceState::query()->increment('generation');
            $this->info('Truncated gis_cache and bumped all gis_source_states.generation (Laravel edge entries invalidate on next read via generation check).');

            return self::SUCCESS;
        }

        $source = (string) $this->option('source');
        GisCache::query()->where('source_used', $source)->delete();
        GisSourceState::query()->where('source_key', $source)->increment('generation');
        $this->info("Deleted gis_cache for source_used={$source} and bumped generation for {$source}.");

        return self::SUCCESS;
    }
}
