<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Services\Bridge\BridgeHttpService;
use App\Services\Bridge\BridgeImageUrlRewriter;
use App\Services\Bridge\BridgeProxyAuditLogger;
use App\Services\Bridge\BridgeTeaser;
use App\Services\Bridge\ListingsCacheService;
use Illuminate\Http\Request;
use Illuminate\Http\Response;
use Symfony\Component\HttpFoundation\Response as SymfonyResponse;

class BridgeProxyController extends Controller
{
    public function __construct(
        private readonly BridgeHttpService $bridge,
        private readonly ListingsCacheService $listingsCache,
        private readonly BridgeProxyAuditLogger $audit,
        private readonly BridgeImageUrlRewriter $imageUrlRewriter,
    ) {}

    /**
     * Revenue impact: collection caching + teaser gating reduce Bridge costs while
     * preserving fast first paint for organic SEO traffic (more leads per dollar).
     */
    public function listings(Request $request): SymfonyResponse
    {
        $url = $this->bridge->webUrl('listings');

        $domainSlug = $request->attributes->get('bridge.domain_slug');
        $useCache = is_string($domainSlug) && $domainSlug !== '' && ! $request->filled('filters');

        if ($useCache) {
            $body = $this->listingsCache->rememberListingsCollection($domainSlug, function () use ($request, $url): array {
                $response = $this->bridge->getJsonFromUrl($url, $request);

                return [
                    'body' => $response->body(),
                    'etag' => $response->header('ETag'),
                ];
            });
        } else {
            $response = $this->bridge->getJsonFromUrl($url, $request);
            if (! $response->successful()) {
                return $this->auditAndPassthrough($request, 'listings.collection', $response);
            }
            $body = $response->body();
        }

        return $this->finalizeJson($request, 'listings.collection', $body);
    }

    /**
     * Revenue impact: detail views are high-intent; serving them quickly increases
     * contact form completion versus abandonment on slow MLS round trips.
     */
    public function listing(Request $request, string $listingId): SymfonyResponse
    {
        $url = $this->bridge->webUrl('listings/'.$listingId);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'listings.detail', $response);
        }

        return $this->finalizeJson($request, 'listings.detail', $response->body());
    }

    public function agents(Request $request): SymfonyResponse
    {
        return $this->proxyWeb($request, 'agents.collection', 'agents');
    }

    public function agent(Request $request, string $agentId): SymfonyResponse
    {
        return $this->proxyWeb($request, 'agents.detail', 'agents/'.$agentId);
    }

    public function offices(Request $request): SymfonyResponse
    {
        return $this->proxyWeb($request, 'offices.collection', 'offices');
    }

    public function office(Request $request, string $officeId): SymfonyResponse
    {
        return $this->proxyWeb($request, 'offices.detail', 'offices/'.$officeId);
    }

    public function openHouses(Request $request): SymfonyResponse
    {
        return $this->proxyWeb($request, 'openhouses.collection', 'openhouses');
    }

    public function openHouse(Request $request, string $openhouseId): SymfonyResponse
    {
        return $this->proxyWeb($request, 'openhouses.detail', 'openhouses/'.$openhouseId);
    }

    /**
     * Revenue impact: RESO Property is the canonical listing payload for IDX compliance;
     * proxying it avoids leaking raw Bridge hosts to unapproved embedders.
     */
    public function properties(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.property.collection', 'Property');
    }

    public function property(Request $request, string $listingKey): SymfonyResponse
    {
        $url = $this->bridge->resoEntityUrl('Property', $listingKey);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.property.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.property.detail', $response->body());
    }

    public function members(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.member.collection', 'Member');
    }

    public function member(Request $request, string $memberKey): SymfonyResponse
    {
        $url = $this->bridge->resoEntityUrl('Member', $memberKey);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.member.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.member.detail', $response->body());
    }

    public function resoOffices(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.office.collection', 'Office');
    }

    public function resoOffice(Request $request, string $officeKey): SymfonyResponse
    {
        $url = $this->bridge->resoEntityUrl('Office', $officeKey);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.office.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.office.detail', $response->body());
    }

    public function resoOpenHouses(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.openhouse.collection', 'OpenHouse');
    }

    public function resoOpenHouse(Request $request, string $openHouseKey): SymfonyResponse
    {
        $url = $this->bridge->resoEntityUrl('OpenHouse', $openHouseKey);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, 'reso.openhouse.detail', $response);
        }

        return $this->finalizeJson($request, 'reso.openhouse.detail', $response->body());
    }

    public function lookup(Request $request): SymfonyResponse
    {
        return $this->proxyResoCollection($request, 'reso.lookup.collection', 'Lookup');
    }

    public function pubParcels(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.collection', 'pub/parcels');
    }

    public function pubParcel(Request $request, string $parcelId): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.detail', 'pub/parcels/'.$parcelId);
    }

    public function pubParcelAssessments(Request $request, string $parcelId): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.assessments', 'pub/parcels/'.$parcelId.'/assessments');
    }

    public function pubParcelTransactions(Request $request, string $parcelId): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.parcels.transactions', 'pub/parcels/'.$parcelId.'/transactions');
    }

    public function pubAssessments(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.assessments.collection', 'pub/assessments');
    }

    public function pubTransactions(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'pub.transactions.collection', 'pub/transactions');
    }

    public function zestimates(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'zestimates.collection', 'zestimates_v2/zestimates');
    }

    public function zgeconMarketReport(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'zgecon.marketreport', 'zgecon/marketreport');
    }

    public function zgeconMarketReportReplication(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'zgecon.marketreport.replication', 'zgecon/marketreport/replication');
    }

    public function zgeconRegion(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'zgecon.region', 'zgecon/region');
    }

    public function zgeconCut(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'zgecon.cut', 'zgecon/cut');
    }

    public function zgeconType(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'zgecon.type', 'zgecon/type');
    }

    public function reviews(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'reviews.collection', 'reviews/Reviews');
    }

    public function reviewees(Request $request): SymfonyResponse
    {
        return $this->proxyDoc($request, 'reviews.reviewees', 'reviews/Reviewees');
    }

    private function proxyWeb(Request $request, string $auditType, string $path): SymfonyResponse
    {
        $url = $this->bridge->webUrl($path);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, $auditType, $response);
        }

        return $this->finalizeJson($request, $auditType, $response->body());
    }

    private function proxyResoCollection(Request $request, string $auditType, string $resource): SymfonyResponse
    {
        $url = $this->bridge->resoCollectionUrl($resource);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, $auditType, $response);
        }

        return $this->finalizeJson($request, $auditType, $response->body());
    }

    private function proxyDoc(Request $request, string $auditType, string $path): SymfonyResponse
    {
        $url = $this->bridge->docPath($path);
        $response = $this->bridge->getJsonFromUrl($url, $request);

        if (! $response->successful()) {
            return $this->auditAndPassthrough($request, $auditType, $response);
        }

        return $this->finalizeJson($request, $auditType, $response->body());
    }

    private function finalizeJson(Request $request, string $auditType, string $body): Response
    {
        $fullAccess = (bool) $request->attributes->get('bridge.full_access', false);

        // Revenue impact: teaser gating is the primary monetization lever for unauthenticated
        // domain traffic; idx:full tokens bypass it to keep geo-web server render latency low.
        $shouldTeaser = ! $fullAccess;
        if ($shouldTeaser) {
            try {
                $body = BridgeTeaser::apply($body, 3);
            } catch (\Throwable) {
                // If Bridge returns non-JSON, passthrough unchanged.
            }
        }

        // Revenue impact: idx-images URLs keep Cloudflare hot on one stable origin shape,
        // shrinking global TTFB and protecting Bridge credentials from browser DevTools leaks.
        try {
            $body = $this->imageUrlRewriter->rewriteJson($body);
        } catch (\Throwable) {
            // passthrough
        }

        $listingCount = BridgeTeaser::countListings($body);

        $this->audit->log(
            $request,
            $auditType,
            $listingCount,
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return response($body, 200, [
            'Content-Type' => 'application/json',
        ]);
    }

    private function auditAndPassthrough(Request $request, string $auditType, \Illuminate\Http\Client\Response $response): Response
    {
        $this->audit->log(
            $request,
            $auditType,
            null,
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        return response($response->body(), $response->status(), array_filter([
            'Content-Type' => $response->header('Content-Type') ?: 'application/json',
        ]));
    }
}
