<?php

namespace App\Http\Middleware;

use App\Models\Domain;
use App\Models\User;
use Closure;
use Illuminate\Http\Request;
use Laravel\Sanctum\PersonalAccessToken;
use Symfony\Component\HttpFoundation\Response;

class DomainOrTokenAuth
{
    /**
     * Authenticates Bridge/GIS proxy traffic via verified domain hostname, or via
     * Sanctum bearer token together with X-Domain-Slug (or ?domain=) for a domain owned by the token user.
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

        $user = User::query()->find($accessToken->tokenable_id);
        if ($user === null) {
            abort(403, 'Invalid API token.');
        }

        $slug = $this->resolveDomainSlugForTokenRequest($request);
        if ($slug === null) {
            abort(400, 'Missing domain identification. Send X-Domain-Slug (or ?domain=) matching a verified domain on your account.');
        }

        /** @var Domain|null $domain */
        $domain = Domain::query()
            ->active()
            ->where('user_id', $user->id)
            ->whereRaw('LOWER(domain_slug) = ?', [mb_strtolower($slug)])
            ->first();

        if ($domain === null) {
            abort(403, 'Domain is not registered, inactive, or not owned by this token.');
        }

        if (! in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true)) {
            abort(403, 'Domain must be TXT-verified before API token access is allowed.');
        }

        $request->attributes->set('bridge.auth', 'token');
        $request->attributes->set('bridge.token_name', $accessToken->name);
        $request->attributes->set('bridge.user_id', $accessToken->tokenable_id);
        $request->attributes->set('bridge.domain', $domain);
        $request->attributes->set('bridge.domain_slug', $domain->domain_slug);
        $request->attributes->set('bridge.full_access', true);

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
        $request->attributes->set('bridge.full_access', true);

        return $next($request);
    }

    private function resolveDomainSlugForTokenRequest(Request $request): ?string
    {
        $header = $request->headers->get('X-Domain-Slug');
        if (is_string($header) && trim($header) !== '') {
            return trim($header);
        }

        $queryDomain = $request->query('domain');
        if (is_string($queryDomain) && trim($queryDomain) !== '') {
            return trim($queryDomain);
        }

        return null;
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
