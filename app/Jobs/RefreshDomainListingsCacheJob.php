<?php

namespace App\Jobs;

use App\Console\Commands\MlsRefreshCache;
use App\Jobs\Mls\RefreshListingsCache;
use App\Models\Domain;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;

/**
 * Revenue impact: legacy single-domain dispatch now fans out per enabled feed so multi-MLS sites refresh fully.
 *
 * Compliance: MLS GRID IDX + Stellar Data Access Agreement — each feed uses its own credential boundary.
 *
 * @deprecated Prefer {@see RefreshListingsCache} dispatched from {@see MlsRefreshCache}.
 */
class RefreshDomainListingsCacheJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    /**
     * Revenue impact: scheduled refresh keeps teaser pages snappy without on-demand cold
     * Bridge hits that would otherwise spike latency during paid ad bursts.
     *
     * Compliance: refresh uses the same feed allowlist as live traffic; no cross-feed credential reuse.
     */
    public function __construct(
        public string $domainSlug,
    ) {}

    public function handle(MlsFeedResolver $feeds): void
    {
        $domain = Domain::query()
            ->active()
            ->where(DB::raw('LOWER(domain_slug)'), '=', Str::lower($this->domainSlug))
            ->first();

        if (! $domain instanceof Domain) {
            return;
        }

        foreach ($feeds->enabledFeedsForDomain($domain) as $internalFeedCode) {
            RefreshListingsCache::dispatch($domain->domain_slug, $internalFeedCode);
        }
    }
}
