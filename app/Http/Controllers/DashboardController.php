<?php

namespace App\Http\Controllers;

use App\Billing\SubscriptionCatalog;
use App\Models\Domain;
use Illuminate\Contracts\View\View;
use Illuminate\Database\QueryException;
use Illuminate\Http\Request;
use Laravel\Cashier\Subscription;

class DashboardController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
    ) {}

    /**
     * Revenue Impact: Dashboard clarity improves activation and lowers churn by
     * showing exactly what subscribers can use next (widgets vs API).
     */
    public function __invoke(Request $request): View
    {
        $user = $request->user();

        /** @var Subscription|null $subscription */
        $subscription = $user?->subscription('default');
        $subscription?->loadMissing('items');

        $priceId = $subscription?->items->first()?->stripe_price;
        $planKey = $user !== null ? ($this->catalog->planKeyForUser($user) ?? '') : '';
        $catalogPlan = $planKey !== '' ? ($this->catalog->plans()[$planKey] ?? null) : null;
        $hasApiAccess = $user !== null && $this->catalog->userMayCreateIdxProxyApiTokens($user);
        $hasWidgetAccess = in_array($planKey, ['pro', 'smart', 'ultra', 'mega'], true);
        $apiRequestLimit = match ($planKey) {
            'ultra' => 2_000_000,
            'mega' => null,
            default => null,
        };
        $apiRequestCount = $hasApiAccess
            ? (int) ($subscription?->items?->first(fn ($item): bool => is_int($item->quantity) && $item->quantity >= 0)?->quantity ?? 0)
            : null;
        $apiOverageRate = (string) config('billing.overage.label', '$0.001 per additional API call');
        $apiOverageCount = $apiRequestLimit !== null && is_int($apiRequestCount)
            ? max(0, $apiRequestCount - $apiRequestLimit)
            : 0;
        $apiTokens = $hasApiAccess
            ? $user->tokens()->latest('id')->limit(8)->get()
            : collect();
        try {
            $activeDomains = Domain::query()
                ->where('is_active', true)
                ->orderBy('domain_slug')
                ->limit(8)
                ->get(['id', 'domain_slug']);
        } catch (QueryException) {
            $activeDomains = collect();
        }
        $leadsThisMonth = app(SubscriptionCatalog::class)->teaserLeadsThisMonth();
        $widgetInstalledCount = $hasWidgetAccess ? min(4, max(1, $activeDomains->count())) : 0;
        $trialEndsAt = $subscription?->trial_ends_at;
        $trialDays = (int) config('billing.trial_days', 14);
        $trialProgressPercent = null;

        if ($trialEndsAt !== null) {
            $elapsed = $trialEndsAt->copy()->subDays($trialDays)->diffInSeconds(now(), false);
            $duration = max(1, $trialDays * 24 * 60 * 60);
            $trialProgressPercent = min(100, max(0, (int) round(($elapsed / $duration) * 100)));
        }

        return view('dashboard.index', [
            'subscription' => $subscription,
            'plan' => is_array($catalogPlan) ? $catalogPlan : [],
            'planKey' => $planKey,
            'priceId' => $priceId,
            'hasApiAccess' => $hasApiAccess,
            'hasWidgetAccess' => $hasWidgetAccess,
            'apiRequestCount' => $apiRequestCount,
            'apiRequestLimit' => $apiRequestLimit,
            'apiOverageRate' => $apiOverageRate,
            'apiOverageCount' => $apiOverageCount,
            'apiTokens' => $apiTokens,
            'activeDomains' => $activeDomains,
            'leadsThisMonth' => $leadsThisMonth,
            'widgetInstalledCount' => $widgetInstalledCount,
            'trialProgressPercent' => $trialProgressPercent,
            'trialEndsAt' => $trialEndsAt,
            'apiPublicUrl' => rtrim((string) config('idx_urls.api_public_url'), '/'),
            'appUrl' => rtrim((string) config('app.url'), '/'),
        ]);
    }
}
