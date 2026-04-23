<?php

namespace App\Ghl\Sync\Jobs;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

/**
 * Revenue Impact: Agency bulk installs unlock many sub-accounts → multiplies addressable lead volume per sale.
 */
class BulkLocationTokenJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public function __construct(
        public int $agencyOAuthTokenId,
    ) {
        $this->onQueue(config('ghl.sync.queues.maintenance'));
    }

    public function handle(): void
    {
        $token = GhlOAuthToken::query()->find($this->agencyOAuthTokenId);
        if (! $token || $token->user_type !== 'Company') {
            return;
        }

        // Future: iterate installed locations / oauth.installedLocations and mint location tokens.
    }
}
