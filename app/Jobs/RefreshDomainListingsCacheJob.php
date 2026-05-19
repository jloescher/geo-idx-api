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
use Illuminate\Support\Facades\Log;
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
        Log::info('RefreshDomainListingsCacheJob started', [
            'domain_slug' => $this->domainSlug,
            'mirror_count' => DB::table('listings')->count(),
        ]);

        $domain = Domain::query()
            ->active()
            ->where(DB::raw('LOWER(domain_slug)'), '=', Str::lower($this->domainSlug))
            ->first();

        if (! $domain instanceof Domain) {
            return;
        }

        $allowed = $domain->getAllowedMlsDatasets();

        // FORCE cache population — critical for geo-web teaser listings + lead gating
        if ($allowed !== null && in_array('bridge_stellar', $allowed, true)) {
            Log::info('Force-populating listings_cache from mirror for Stellar domain', [
                'domain_slug' => $domain->domain_slug,
                'mirror_rows' => DB::table('listings')->count(),
            ]);

            $this->populateListingsCacheFromMirror($domain, $feeds);
        }

        Log::info('listings_cache population complete', [
            'domain_slug' => $domain->domain_slug,
            'cache_rows' => DB::table('listings_cache')->count(),
        ]);
    }

    private function populateListingsCacheFromMirror(Domain $domain, MlsFeedResolver $feeds): void
    {
        $feedCodes = $feeds->enabledFeedsForDomain($domain);
        if ($feedCodes === []) {
            $feedCodes = $feeds->catalogFeedCodes();
        }

        foreach ($feedCodes as $internalFeedCode) {
            DB::table('listings_cache')
                ->where('domain_slug', $domain->domain_slug)
                ->where('feed_code', $internalFeedCode)
                ->delete();

            RefreshListingsCache::dispatchSync($domain->domain_slug, $internalFeedCode);
        }
    }
}
