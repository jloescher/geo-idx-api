<?php

namespace App\Jobs;

use App\Enums\MlsProvider;
use App\Models\Domain;
use App\Services\Bridge\ListingsCacheService;
use App\Services\Mls\MlsClientFactory;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Http\Request;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;

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

    public function handle(ListingsCacheService $cache, MlsClientFactory $factory, MlsFeedResolver $feeds): void
    {
        $domain = Domain::query()
            ->active()
            ->where(DB::raw('LOWER(domain_slug)'), '=', Str::lower($this->domainSlug))
            ->first();

        if (! $domain instanceof Domain) {
            return;
        }

        $feedCode = $domain->resolveDefaultFeedCode();
        $restriction = $domain->getAllowedMlsDatasets();
        $catalog = $feeds->catalogFeedCodes();
        if ($restriction !== null) {
            $intersect = array_values(array_intersect($restriction, $catalog));
            if ($intersect !== []) {
                $preferred = $domain->getMlsDataset();
                if (is_string($preferred) && trim($preferred) !== '' && in_array(trim($preferred), $intersect, true)) {
                    $feedCode = trim($preferred);
                } else {
                    $feedCode = $intersect[0];
                }
            }
        }

        $incoming = Request::create('/api/v1/listings', 'GET', [
            'limit' => 200,
        ]);
        $incoming->attributes->set('bridge.domain', $domain);
        $incoming->attributes->set('bridge.domain_slug', $domain->domain_slug);

        $def = $feeds->feedDefinition($feedCode);
        $provider = (($def['provider'] ?? '') === MlsProvider::Spark->value)
            ? MlsProvider::Spark
            : MlsProvider::Bridge;

        $cache->rememberListingsCollection($domain->domain_slug, $feedCode, function () use ($factory, $feedCode, $provider, $incoming): array {
            if ($provider === MlsProvider::Spark) {
                $response = $factory->sparkClientForFeed($feedCode)->getActivePendingPropertyCollection($incoming);

                return [
                    'body' => $response->body(),
                    'etag' => $response->header('ETag'),
                ];
            }

            $client = $factory->bridgeClientForFeed($feedCode);
            $url = $client->webUrl('listings');
            $response = $client->getJsonFromUrl($url, $incoming);

            return [
                'body' => $response->body(),
                'etag' => $response->header('ETag'),
            ];
        });
    }
}
