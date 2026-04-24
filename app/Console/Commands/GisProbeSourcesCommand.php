<?php

namespace App\Console\Commands;

use App\Jobs\RefreshGisSourceMetadataJob;
use App\Services\GisSourceMetadataService;
use Illuminate\Console\Command;

class GisProbeSourcesCommand extends Command
{
    protected $signature = 'gis:probe-sources {--queued : Dispatch the queued job instead of running synchronously}';

    protected $description = 'Probe ArcGIS layer metadata for all GIS sources and bump generation when fingerprints change';

    public function handle(GisSourceMetadataService $metadata): int
    {
        if ($this->option('queued')) {
            RefreshGisSourceMetadataJob::dispatch()->onQueue((string) config('gis.queue'));
            $this->info('Dispatched '.RefreshGisSourceMetadataJob::class);

            return self::SUCCESS;
        }

        $metadata->probeAllSources();
        $this->info('GIS source metadata probe complete.');

        return self::SUCCESS;
    }
}
