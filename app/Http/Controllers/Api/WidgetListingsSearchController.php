<?php

namespace App\Http\Controllers\Api;

use App\Ghl\Widgets\Support\WidgetEmbedDomainResolver;
use App\Http\Controllers\Controller;
use App\Http\Requests\Search\SearchRequest;
use App\Services\Bridge\BridgeSearchAction;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;

/**
 * Browser-callable MLS search for embed widgets (domain-scoped, teaser-gated like Bridge proxy).
 */
class WidgetListingsSearchController extends Controller
{
    public function __construct(
        private readonly BridgeSearchAction $bridgeSearchAction,
    ) {}

    public function __invoke(Request $request): JsonResponse
    {
        $domain = WidgetEmbedDomainResolver::resolveActiveDomain($request);
        if ($domain === null) {
            return response()->json([
                'message' => 'No approved MLS domain matches this embed origin. Add the site hostname under My Approved Domains.',
            ], 422);
        }

        $validated = Validator::make(
            $request->all(),
            (new SearchRequest)->rules(),
        )->validate();

        $bridgeRequest = Request::create('/', 'POST', $validated);
        $bridgeRequest->query->replace($request->query->all());
        $bridgeRequest->attributes->set('bridge.domain', $domain);
        $bridgeRequest->attributes->set('bridge.domain_slug', $domain->domain_slug);
        $bridgeRequest->attributes->set('bridge.auth', 'domain');
        $bridgeRequest->attributes->set('bridge.full_access', false);
        $bridgeRequest->attributes->set('bridge.token_name', null);
        $bridgeRequest->attributes->set('bridge.user_id', null);

        $searchRequest = SearchRequest::createFrom($bridgeRequest);
        $searchRequest->setContainer(app())->setRedirector(app('redirect'));
        $searchRequest->setUserResolver($request->getUserResolver());

        if ($request->route() !== null) {
            $searchRequest->setRouteResolver(static fn () => $request->route());
        }

        $searchRequest->validateResolved();

        return ($this->bridgeSearchAction)($searchRequest);
    }
}
