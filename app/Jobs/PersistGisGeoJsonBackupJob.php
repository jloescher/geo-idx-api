<?php

namespace App\Jobs;

use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Storage;

class PersistGisGeoJsonBackupJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    /**
     * Revenue impact: async NVMe writes keep request latency inside OTP funnel
     * budgets while still giving ops a filesystem trail for support escalations.
     */
    public function __construct(
        public string $queryHash,
        public string $jsonBody,
    ) {}

    public function handle(): void
    {
        Storage::disk('gis_backup')->put($this->queryHash.'.json', $this->jsonBody);
    }
}
