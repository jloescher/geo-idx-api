<?php

namespace App\Ghl\Widgets\Middleware;

use App\Billing\SubscriptionCatalog;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Models\Domain;
use App\Models\User;
use App\Widgets\DirectSiteWidgetContext;
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
        if ($row !== null && $row->widget_access_enabled) {
            $request->attributes->set('ghl_registered_url', $row);

            return $next($request);
        }

        $user = User::query()->where('widget_embed_site_key', $apiKey)->first();
        if ($user !== null) {
            $catalog = app(SubscriptionCatalog::class);
            $hasPaidSubscription = (bool) $user->subscription('default')?->valid();
            $mlsActive = (string) $user->mls_membership_status === 'active';
            $hasVerifiedDomain = Domain::query()
                ->where('user_id', $user->id)
                ->where('is_active', true)
                ->whereIn('verification_status', ['verified', 'verified_ghl'])
                ->exists();
            if (! $hasPaidSubscription || ! $mlsActive || ! $hasVerifiedDomain || ! in_array($catalog->planKeyForUser($user) ?? '', ['pro', 'smart', 'ultra', 'mega'], true)) {
                return response('Subscriber access requirements not met for widget runtime', 403);
            }

            $request->attributes->set('ghl_registered_url', new DirectSiteWidgetContext($user));

            return $next($request);
        }

        return response('Invalid widget API key', 401);
    }
}
