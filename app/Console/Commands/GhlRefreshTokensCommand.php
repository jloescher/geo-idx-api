<?php

namespace App\Console\Commands;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Sync\Jobs\RefreshGhlTokensJob;
use Illuminate\Console\Command;

class GhlRefreshTokensCommand extends Command
{
    protected $signature = 'ghl:refresh-tokens';

    protected $description = 'Queue GHL OAuth token refresh for tokens expiring within 2 hours';

    public function handle(): int
    {
        $threshold = now()->addHours(2);

        GhlOAuthToken::query()
            ->where('status', 'active')
            ->where('expires_at', '<=', $threshold)
            ->each(function (GhlOAuthToken $token): void {
                RefreshGhlTokensJob::dispatch($token->id);
            });

        $this->info('Queued refresh jobs for expiring GHL tokens.');

        return self::SUCCESS;
    }
}
