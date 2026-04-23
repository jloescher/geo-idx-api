<?php

namespace App\Ghl\Widgets\Middleware;

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
            foreach ($row->allowedOrigins() as $a) {
                $base = rtrim((string) $a, '/');
                if ($base !== '' && str_starts_with(rtrim($origin, '/'), $base)) {
                    $response->headers->set('Access-Control-Allow-Origin', rtrim($origin, '/'));
                    $response->headers->set('Vary', 'Origin');
                    break;
                }
            }
        }

        return $response;
    }
}
