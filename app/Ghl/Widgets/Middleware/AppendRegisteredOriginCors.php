<?php

namespace App\Ghl\Widgets\Middleware;

use App\Ghl\Widgets\Support\OriginMatcher;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Revenue Impact: CORS on registered origins only → MLS-safe embeds → partner trust → more installs.
 */
class AppendRegisteredOriginCors
{
    public function handle(Request $request, Closure $next): Response
    {
        $response = $next($request);
        $row = $request->attributes->get('ghl_registered_url');
        $origin = (string) $request->headers->get('Origin', '');
        if ($row && $origin !== '') {
            $validatedOrigin = OriginMatcher::allowedOrigin($origin, $row->allowedOrigins())
                ?? $request->attributes->get('validated_widget_origin');

            if (is_string($validatedOrigin) && $validatedOrigin !== '') {
                $response->headers->set('Access-Control-Allow-Origin', $validatedOrigin);
                $response->headers->set('Vary', 'Origin');
            }
        }

        return $response;
    }
}
