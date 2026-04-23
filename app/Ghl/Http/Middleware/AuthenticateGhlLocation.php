<?php

namespace App\Ghl\Http\Middleware;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Revenue Impact: Authenticated GHL API calls unlock teaser stats → paywall CTAs → Stripe upgrades.
 */
class AuthenticateGhlLocation
{
    public function handle(Request $request, Closure $next): Response
    {
        $bearer = $request->bearerToken();
        if (! $bearer) {
            return response()->json(['message' => 'Missing bearer token'], 401);
        }

        $hash = hash('sha256', $bearer);
        $token = GhlOAuthToken::query()
            ->where('access_token_hash', $hash)
            ->where('status', 'active')
            ->first();

        if (! $token) {
            return response()->json(['message' => 'Invalid token'], 401);
        }

        if ($token->expires_at && $token->expires_at->isPast()) {
            return response()->json(['message' => 'Token expired; refresh OAuth'], 401);
        }

        $request->attributes->set('ghl_oauth_token', $token);

        return $next($request);
    }
}
