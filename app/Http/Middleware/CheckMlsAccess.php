<?php

namespace App\Http\Middleware;

use App\Services\Mls\MlsFeedResolver;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Revenue impact: MLS feed allowlists protect paid data from cross-market scraping and quota bleed.
 *
 * Compliance: MLS GRID IDX + participant agreements require domain-scoped feed selection before any MLS API call.
 */
final readonly class CheckMlsAccess
{
    public function __construct(
        private MlsFeedResolver $feeds,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        if ($this->shouldBypass($request)) {
            return $next($request);
        }

        $feedCode = $this->feeds->resolveFeedCode($request);
        $request->attributes->set('mls.feed_code', $feedCode);
        $request->attributes->set('mls.feed_definition', $this->feeds->feedDefinition($feedCode));

        return $next($request);
    }

    private function shouldBypass(Request $request): bool
    {
        $path = $request->path();

        if ($path === 'api/v1/gis') {
            return true;
        }

        if (preg_match('#^api/v1/mls/[^/]+/gis$#', $path) === 1) {
            return true;
        }

        return false;
    }
}
