<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeHttpService;
use App\Services\Bridge\ListingsCacheService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Http\Request;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

class RefreshDomainListingsCacheJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    /**
     * Revenue impact: scheduled refresh keeps teaser pages snappy without on-demand cold
     * Bridge hits that would otherwise spike latency during paid ad bursts.
     */
    public function __construct(
        public string $domainSlug,
    ) {}

    public function handle(BridgeHttpService $bridge, ListingsCacheService $cache): void
    {
        $incoming = Request::create('/api/v1/listings', 'GET', [
            'limit' => 200,
        ]);

        $url = $bridge->webUrl('listings');

        $cache->rememberListingsCollection($this->domainSlug, function () use ($bridge, $incoming, $url): array {
            $response = $bridge->getJsonFromUrl($url, $incoming);

            return [
                'body' => $response->body(),
                'etag' => $response->header('ETag'),
            ];
        });
    }
}
