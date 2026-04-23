<?php

namespace App\Ghl\Widgets\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Revenue Impact: Origin allowlist prevents MLS data leakage to unregistered domains → Stellar compliance + liability shield.
 */
class ValidateWidgetOrigin
{
    public function handle(Request $request, Closure $next): Response
    {
        $row = $request->attributes->get('ghl_registered_url');
        if (! $row) {
            return response('Widget context missing', 500);
        }

        $origin = (string) $request->headers->get('Origin', '');
        if ($origin === '') {
            // Same-origin server fetches may omit Origin; allow Referer match for SSR edge cases.
            $origin = (string) $request->headers->get('Referer', '');
        }

        if ($origin === '') {
            return response('Missing Origin', 403);
        }

        $normalized = rtrim($origin, '/');
        $allowed = $row->allowedOrigins();
        $ok = false;
        foreach ($allowed as $a) {
            $b = rtrim((string) $a, '/');
            if ($b !== '' && str_starts_with($normalized, $b)) {
                $ok = true;
                break;
            }
        }

        if (! $ok) {
            return response('Origin not registered for MLS compliance', 403);
        }

        return $next($request);
    }
}
