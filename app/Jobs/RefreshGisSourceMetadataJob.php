<?php

namespace App\Jobs;

use App\Services\GisSourceMetadataService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

class RefreshGisSourceMetadataJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    /**
     * Revenue impact: queued probes keep Octane workers free during FGIO metadata
     * checks while still invalidating stale parcel blobs after county publishes.
     */
    public function handle(GisSourceMetadataService $metadata): void
    {
        $metadata->probeAllSources();
    }
}
