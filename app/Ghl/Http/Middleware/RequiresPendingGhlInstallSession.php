<?php

namespace App\Ghl\Http\Middleware;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Revenue Impact: Forces MLS URL registration before widgets go live → compliance + trust → higher close rate.
 */
class RequiresPendingGhlInstallSession
{
    public function handle(Request $request, Closure $next): Response
    {
        $id = session('ghl_pending_oauth_token_id');
        if (! $id || ! GhlOAuthToken::query()->whereKey($id)->exists()) {
            return redirect()->route('leadconnector.install')
                ->with('error', 'Complete OAuth install first.');
        }

        return $next($request);
    }
}
