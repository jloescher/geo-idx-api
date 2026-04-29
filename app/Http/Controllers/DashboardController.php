<?php

namespace App\Http\Controllers;

use App\Billing\SubscriptionCatalog;
use App\Ghl\Sync\Models\QuantyraLead;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Models\Domain;
use App\Services\SubscriberLeadScopeService;
use Illuminate\Contracts\View\View;
use Illuminate\Database\QueryException;
use Illuminate\Http\Request;
use Laravel\Cashier\Subscription;

class DashboardController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
        private readonly SubscriberLeadScopeService $leadScope,
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
                ->where('user_id', $user?->id)
                ->where('is_active', true)
                ->orderBy('domain_slug')
                ->limit(8)
                ->get([
                    'id',
                    'domain_slug',
                    'verification_status',
                    'verification_method',
                    'txt_verification_name',
                    'txt_verification_value',
                ]);
        } catch (QueryException) {
            $activeDomains = collect();
        }
        $widgetPreviewApiKey = (string) $request->query('widget_site_key', (string) $request->query('widget_api_key', ''));
        $leadsThisMonth = null;
        $leadsMetricAvailable = false;
        $widgetInstalledCount = 0;

        try {
            if ($widgetPreviewApiKey === '' && $user !== null && $hasWidgetAccess) {
                $widgetPreviewApiKey = $user->ensureWidgetEmbedSiteKey();
            }
            if ($widgetPreviewApiKey === '') {
                $widgetPreviewApiKey = (string) (GhlRegisteredUrl::query()
                    ->where('widget_access_enabled', true)
                    ->whereNotNull('widget_api_key')
                    ->where('widget_api_key', '!=', '')
                    ->orderByDesc('updated_at')
                    ->orderByDesc('id')
                    ->value('widget_api_key') ?? '');
            }
            if ($widgetPreviewApiKey === '') {
                $widgetPreviewApiKey = (string) (GhlRegisteredUrl::query()
                    ->whereNotNull('widget_api_key')
                    ->where('widget_api_key', '!=', '')
                    ->orderByDesc('updated_at')
                    ->orderByDesc('id')
                    ->value('widget_api_key') ?? '');
            }

            if ($widgetPreviewApiKey !== '') {
                $previewRow = GhlRegisteredUrl::query()
                    ->where('widget_api_key', $widgetPreviewApiKey)
                    ->where('widget_access_enabled', true)
                    ->first();

                if ($previewRow !== null) {
                    $leadsThisMonth = QuantyraLead::query()
                        ->where('ghl_location_id', $previewRow->ghl_location_id)
                        ->where('created_at', '>=', now()->startOfMonth())
                        ->count();
                    $leadsMetricAvailable = true;
                }
            }

            $widgetInstalledCount = GhlRegisteredUrl::query()
                ->where('quantyra_user_id', $user?->id)
                ->where('widget_access_enabled', true)
                ->count();
        } catch (QueryException) {
            $leadsThisMonth = null;
            $leadsMetricAvailable = false;
            $widgetInstalledCount = 0;
        }
        $trialEndsAt = null;
        $trialProgressPercent = null;
        $mlsVerified = $user !== null && (string) $user->mls_membership_status === 'active';
        $hasVerifiedDomain = $activeDomains
            ->whereIn('verification_status', ['verified', 'verified_ghl'])
            ->isNotEmpty();

        $onboardingSteps = [
            [
                'key' => 'mls',
                'label' => 'Verify Stellar MLS membership (MLS ID + email)',
                'done' => $mlsVerified,
            ],
            [
                'key' => 'subscription',
                'label' => 'Start paid subscription (no trial)',
                'done' => $subscription?->valid() ?? false,
            ],
            [
                'key' => 'domains',
                'label' => 'Register and verify at least one domain',
                'done' => $hasVerifiedDomain,
            ],
            [
                'key' => 'widgets',
                'label' => 'Enable widget access by TXT verification or LeadConnector domain attachment',
                'done' => $hasVerifiedDomain || $widgetInstalledCount > 0,
            ],
            [
                'key' => 'api',
                'label' => 'Generate API token (Ultra/Mega)',
                'done' => ! $hasApiAccess || $apiTokens->isNotEmpty(),
            ],
        ];
        $onboardingCompletedCount = collect($onboardingSteps)->where('done', true)->count();
        $domainLimit = $user !== null ? $this->catalog->domainLimitForUser($user) : 0;
        $domainLimitReached = is_int($domainLimit) && $activeDomains->count() >= $domainLimit;
        $extraDomainAddonPriceId = (string) config('billing.addons.extra_domain.stripe_price_monthly', '');
        $canPurchaseExtraDomainSlots = $planKey === 'pro' && $extraDomainAddonPriceId !== '';

        $appUrl = rtrim((string) config('app.url'), '/');
        $widgetLoaderBaseUrl = rtrim((string) config('idx_urls.api_public_url'), '/');

        $widgetPaletteForm = [
            'primary' => '#2563EB',
            'secondary' => '#1E40AF',
            'accent' => '#10B981',
            'text' => '#0f172a',
            'background' => '#ffffff',
            'theme' => 'light',
        ];
        if ($user !== null && is_array($user->widget_palette)) {
            $widgetPaletteForm = array_merge($widgetPaletteForm, $user->widget_palette);
        }

        $activePanel = (string) $request->query('panel', 'dashboard');
        if (! in_array($activePanel, ['dashboard', 'onboarding', 'widgets', 'leads', 'domains', 'api', 'billing', 'settings'], true)) {
            $activePanel = 'dashboard';
        }

        $leadsEligible = $user !== null
            && in_array($planKey, ['pro', 'smart', 'ultra', 'mega'], true)
            && (string) $user->mls_membership_status === 'active'
            && $subscription?->valid() === true;

        $totalLeads = 0;
        $hotLeads24h = 0;
        $conversionRate = 0.0;
        if ($user !== null) {
            $locationIds = $this->leadScope->locationIdsForUser($user);
            $query = QuantyraLead::query()->whereIn('ghl_location_id', $locationIds);
            $totalLeads = (clone $query)->count();
            $hotLeads24h = (clone $query)->where('created_at', '>=', now()->subDay())->count();
            $convertedLeads = (clone $query)->where('payload->status', 'converted')->count();
            $conversionRate = $totalLeads > 0 ? round(($convertedLeads / $totalLeads) * 100, 1) : 0.0;
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
            'leadsMetricAvailable' => $leadsMetricAvailable,
            'widgetInstalledCount' => $widgetInstalledCount,
            'trialProgressPercent' => $trialProgressPercent,
            'trialEndsAt' => $trialEndsAt,
            'onboardingSteps' => $onboardingSteps,
            'onboardingCompletedCount' => $onboardingCompletedCount,
            'domainLimit' => $domainLimit,
            'domainLimitReached' => $domainLimitReached,
            'widgetPreviewApiKey' => $widgetPreviewApiKey,
            'canPurchaseExtraDomainSlots' => $canPurchaseExtraDomainSlots,
            'apiPublicUrl' => rtrim((string) config('idx_urls.api_public_url'), '/'),
            'appUrl' => $appUrl,
            'widgetLoaderBaseUrl' => $widgetLoaderBaseUrl,
            'widgetPaletteForm' => $widgetPaletteForm,
            'activePanel' => $activePanel,
            'leadsEligible' => $leadsEligible,
            'totalLeads' => $totalLeads,
            'hotLeads24h' => $hotLeads24h,
            'conversionRate' => $conversionRate,
        ]);
    }
}
