<?php

namespace App\Ghl\Sync\Jobs;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\OAuth\Services\TokenRefreshService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

/**
 * Revenue Impact: Proactive refresh prevents CRM sync outages → continuous lead delivery → retention.
 */
class RefreshGhlTokensJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public function __construct(
        public int $ghlOAuthTokenId,
    ) {
        $this->onQueue(config('ghl.sync.queues.maintenance'));
    }

    public function handle(TokenRefreshService $refresh): void
    {
        $token = GhlOAuthToken::query()->find($this->ghlOAuthTokenId);
        if (! $token || $token->status !== 'active') {
            return;
        }

        $refresh->refresh($token);
    }
}
