<?php

namespace App\Console\Commands;

use App\Jobs\Mls\RefreshListingsCache;
use App\Models\Domain;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Console\Command;

/**
 * Revenue impact: predictable 15-minute MLS cache churn keeps IDX conversion paths fast while capping upstream spend.
 *
 * Compliance: MLS GRID IDX + Stellar Data Access Agreement — refresh respects domain feed allowlists only.
 */
class MlsRefreshCache extends Command
{
    protected $signature = 'mls:refresh-cache {--domain= : Restrict to a single domain slug}';

    protected $description = 'Queue MLS Active/Pending listings cache refresh jobs per domain and internal feed code';

    public function handle(MlsFeedResolver $feeds): int
    {
        $query = Domain::query()->active()->orderBy('domain_slug');
        $slug = $this->option('domain');
        if (is_string($slug) && trim($slug) !== '') {
            $query->where('domain_slug', trim($slug));
        }

        $dispatched = 0;
        foreach ($query->cursor() as $domain) {
            foreach ($feeds->enabledFeedsForDomain($domain) as $internalFeedCode) {
                RefreshListingsCache::dispatch($domain->domain_slug, $internalFeedCode);
                $dispatched++;
            }
        }

        $this->info("Dispatched {$dispatched} MLS listings cache refresh job(s).");

        return self::SUCCESS;
    }
}
