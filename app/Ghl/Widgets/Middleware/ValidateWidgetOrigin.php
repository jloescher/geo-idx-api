<?php

namespace App\Ghl\Widgets\Middleware;

use App\Ghl\Widgets\Support\OriginMatcher;
use App\Ghl\Widgets\Support\TrustedWidgetPreviewOrigins;
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
        $referer = (string) $request->headers->get('Referer', '');

        if ($origin === '' && $referer === '') {
            return response('Missing Origin', 403);
        }

        $validatedOrigin = OriginMatcher::allowedOrigin($origin, $row->allowedOrigins())
            ?? OriginMatcher::allowedOrigin($referer, $row->allowedOrigins())
            ?? TrustedWidgetPreviewOrigins::originIfTrusted($origin)
            ?? TrustedWidgetPreviewOrigins::originIfTrusted($referer);

        if ($validatedOrigin === null) {
            return response('Origin not registered for MLS compliance', 403);
        }

        $request->attributes->set('validated_widget_origin', $validatedOrigin);

        return $next($request);
    }
}
