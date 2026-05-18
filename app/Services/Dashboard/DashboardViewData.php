<?php

namespace App\Services\Dashboard;

use App\Models\Domain;
use App\Models\User;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Database\QueryException;
use Illuminate\Http\Request;

final class DashboardViewData
{
    private const PANELS = ['dashboard', 'onboarding', 'domains', 'api'];

    public function __construct(
        private readonly MlsFeedResolver $mlsFeeds,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function forRequest(Request $request): array
    {
        $user = $request->user();

        $apiTokens = $user !== null
            ? $user->tokens()->latest('id')->limit(8)->get()
            : collect();

        try {
            $activeDomains = Domain::query()
                ->where('user_id', $user?->id)
                ->where('is_active', true)
                ->orderBy('domain_slug')
                ->limit(50)
                ->get([
                    'id',
                    'domain_slug',
                    'mls_dataset',
                    'allowed_mls_datasets',
                    'verification_status',
                    'verification_method',
                    'txt_verification_name',
                    'txt_verification_value',
                ]);
        } catch (QueryException) {
            $activeDomains = collect();
        }

        $verifiedDomainCount = $activeDomains
            ->whereIn('verification_status', ['verified', 'verified_ghl'])
            ->count();

        $hasVerifiedDomain = $activeDomains
            ->whereIn('verification_status', ['verified', 'verified_ghl'])
            ->isNotEmpty();

        $verifiedDomains = $activeDomains->whereIn('verification_status', ['verified', 'verified_ghl']);
        $allVerifiedDomainsHaveMlsScope = $verifiedDomains->isNotEmpty() && $verifiedDomains->every(function (Domain $domain): bool {
            $allowed = $domain->getAllowedMlsDatasets();

            return $allowed !== null && $allowed !== [];
        });

        $onboardingSteps = [
            [
                'key' => 'domains',
                'label' => 'Register and verify at least one domain (DNS TXT)',
                'done' => $hasVerifiedDomain,
            ],
            [
                'key' => 'mls_scope',
                'label' => 'Confirm MLS feed access on each verified domain (Domains tab)',
                'done' => ! $hasVerifiedDomain || $allVerifiedDomainsHaveMlsScope,
            ],
            [
                'key' => 'api',
                'label' => 'Create an API token and call the API with Authorization + X-Domain-Slug',
                'done' => $apiTokens->isNotEmpty(),
            ],
        ];
        $onboardingCompletedCount = collect($onboardingSteps)->where('done', true)->count();

        $appUrl = rtrim((string) config('app.url'), '/');

        $activePanel = (string) $request->query('panel', 'dashboard');
        if (! in_array($activePanel, self::PANELS, true)) {
            $activePanel = 'dashboard';
        }

        return [
            'hasApiAccess' => true,
            'apiTokens' => $apiTokens,
            'activeDomains' => $activeDomains,
            'verifiedDomainCount' => $verifiedDomainCount,
            'onboardingSteps' => $onboardingSteps,
            'onboardingCompletedCount' => $onboardingCompletedCount,
            'domainLimit' => null,
            'domainLimitReached' => false,
            'canPurchaseExtraDomainSlots' => false,
            'canInviteUsers' => $user instanceof User && $user->isAdmin(),
            'mlsCatalogFeedCodes' => $this->mlsFeeds->catalogFeedCodes(),
            'apiPublicUrl' => rtrim((string) config('idx_urls.api_public_url'), '/'),
            'appUrl' => $appUrl,
            'activePanel' => $activePanel,
        ];
    }
}
