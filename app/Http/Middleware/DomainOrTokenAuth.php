<?php

namespace App\Http\Middleware;

use App\Models\Domain;
use Closure;
use Illuminate\Http\Request;
use Laravel\Sanctum\PersonalAccessToken;
use Symfony\Component\HttpFoundation\Response;

class DomainOrTokenAuth
{
    /**
     * Revenue impact: this gate is the primary unauthorized-access prevention control
     * for Stellar MLS data; it converts anonymous traffic into either verified domains
     * (lead capture) or tokenized server-to-server integrations (full IDX throughput).
     */
    public function handle(Request $request, Closure $next): Response
    {
        $token = $request->bearerToken();
        if (is_string($token) && $token !== '') {
            return $this->handlePersonalAccessToken($request, $next, $token);
        }

        return $this->handleDomainIdentification($request, $next);
    }

    private function handlePersonalAccessToken(Request $request, Closure $next, string $plainTextToken): Response
    {
        $accessToken = PersonalAccessToken::findToken($plainTextToken);

        if (! $accessToken) {
            abort(403, 'Invalid API token.');
        }

        $canAccess = $accessToken->can('idx:access') || $accessToken->can('idx:full');
        if (! $canAccess) {
            abort(403, 'Token is missing required IDX abilities.');
        }

        $request->attributes->set('bridge.auth', 'token');
        $request->attributes->set('bridge.token_name', $accessToken->name);
        $request->attributes->set('bridge.user_id', $accessToken->tokenable_id);
        $request->attributes->set('bridge.full_access', $accessToken->can('idx:full'));

        return $next($request);
    }

    private function handleDomainIdentification(Request $request, Closure $next): Response
    {
        $slug = $this->resolveDomainSlug($request);
        if ($slug === null) {
            abort(401, 'Missing domain identification.');
        }

        /** @var Domain|null $domain */
        $domain = Domain::query()
            ->active()
            ->whereRaw('LOWER(domain_slug) = ?', [mb_strtolower($slug)])
            ->first();

        if (! $domain) {
            abort(403, 'Domain is not registered or inactive.');
        }

        $request->attributes->set('bridge.auth', 'domain');
        $request->attributes->set('bridge.domain', $domain);
        $request->attributes->set('bridge.domain_slug', $domain->domain_slug);
        $request->attributes->set('bridge.token_name', null);
        $request->attributes->set('bridge.user_id', null);
        $request->attributes->set('bridge.full_access', false);

        return $next($request);
    }

    private function resolveDomainSlug(Request $request): ?string
    {
        $header = $request->headers->get('X-Domain-Slug');
        if (is_string($header) && trim($header) !== '') {
            return trim($header);
        }

        $queryDomain = $request->query('domain');
        if (is_string($queryDomain) && trim($queryDomain) !== '') {
            return trim($queryDomain);
        }

        $referer = $request->headers->get('referer');
        if (! is_string($referer) || $referer === '') {
            return null;
        }

        $host = parse_url($referer, PHP_URL_HOST);
        if (! is_string($host) || $host === '') {
            return null;
        }

        return mb_strtolower($host);
    }
}
