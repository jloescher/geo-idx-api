<?php

namespace App\Ghl\Widgets\Middleware;

use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Revenue Impact: API keys gate MLS widget traffic → monetizable authenticated usage only.
 */
class ValidateWidgetApiKey
{
    public function handle(Request $request, Closure $next): Response
    {
        $apiKey = $request->route('apiKey')
            ?? $request->query('api_key')
            ?? $request->input('api_key')
            ?? $request->header('X-Quantyra-Widget-Key');

        if (! $apiKey) {
            return response('Missing widget API key', 401);
        }

        $row = GhlRegisteredUrl::query()->where('widget_api_key', $apiKey)->first();
        if (! $row || ! $row->widget_access_enabled) {
            return response('Invalid widget API key', 401);
        }

        $request->attributes->set('ghl_registered_url', $row);

        return $next($request);
    }
}
