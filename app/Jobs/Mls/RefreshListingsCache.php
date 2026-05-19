<?php

namespace App\Jobs\Mls;

use App\Models\Domain;
use App\Services\Bridge\ListingsCacheService;
use App\Services\Mls\MlsActivePendingListingsFetcher;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Http\Request;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;

/**
 * Revenue impact: queued per-feed Active+Pending sync maximizes listing coverage for geo-web without duplicate MLS toll.
 *
 * Compliance: MLS GRID IDX + Stellar Data Access Agreement — cache stores only Active/Pending RESO snapshots per feed.
 */
class RefreshListingsCache implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public function __construct(
        public string $domainSlug,
        public string $internalFeedCode,
    ) {}

    public function handle(
        ListingsCacheService $cache,
        MlsActivePendingListingsFetcher $fetcher,
        MlsFeedResolver $feeds,
    ): void {
        $domain = Domain::query()
            ->active()
            ->where(DB::raw('LOWER(domain_slug)'), '=', Str::lower($this->domainSlug))
            ->first();

        if (! $domain instanceof Domain) {
            return;
        }

        $enabled = $feeds->enabledFeedsForDomain($domain);
        if (! in_array($this->internalFeedCode, $enabled, true)) {
            return;
        }

        // Do not pass Web-style `limit` — Bridge OData rejects it alongside $top (see BridgeProxyController).
        $incoming = Request::create('/api/v1/listings', 'GET');
        $incoming->attributes->set('bridge.domain', $domain);
        $incoming->attributes->set('bridge.domain_slug', $domain->domain_slug);
        $incoming->attributes->set('mls.feed_code', $this->internalFeedCode);

        $cache->rememberListingsCollection($domain->domain_slug, $this->internalFeedCode, function () use ($fetcher, $incoming): array {
            $payload = $fetcher->fetchMergedCollectionForCache($incoming);

            return [
                'body' => $payload['body'],
                'etag' => $payload['etag'],
            ];
        });
    }
}
