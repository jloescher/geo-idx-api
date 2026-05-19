<?php

namespace App\Services\Dashboard;

use App\Models\Domain;
use App\Models\User;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Database\QueryException;
use Illuminate\Http\Request;

final class DashboardViewData
{
    private const PANELS = ['setup', 'api'];

    private const LEGACY_PANEL_MAP = [
        'dashboard' => 'setup',
        'onboarding' => 'setup',
        'domains' => 'setup',
    ];

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

        $verifiedDomains = $activeDomains->whereIn('verification_status', ['verified', 'verified_ghl']);
        $verifiedDomainCount = $verifiedDomains->count();
        $hasVerifiedDomain = $verifiedDomains->isNotEmpty();

        $primaryVerifiedDomain = $verifiedDomains->first();
        $primaryVerifiedDomainSlug = $primaryVerifiedDomain?->domain_slug;

        $pendingDomains = $activeDomains->whereNotIn('verification_status', ['verified', 'verified_ghl']);

        $setupPhase = match (true) {
            $hasVerifiedDomain => 'ready',
            $activeDomains->isNotEmpty() => 'verify',
            default => 'register',
        };

        $tokenNames = $apiTokens->pluck('name')->map(fn (string $name): string => mb_strtolower(trim($name)))->all();
        $hasProductionToken = in_array('production', $tokenNames, true);
        $hasStagingToken = in_array('staging', $tokenNames, true);
        $canCreateStagingToken = $hasVerifiedDomain && ! $hasStagingToken;

        $appUrl = rtrim((string) config('app.url'), '/');

        $requestedPanel = (string) $request->query('panel', 'setup');
        $activePanel = $this->resolvePanel($requestedPanel);

        return [
            'hasApiAccess' => true,
            'apiTokens' => $apiTokens,
            'activeDomains' => $activeDomains,
            'verifiedDomainCount' => $verifiedDomainCount,
            'hasVerifiedDomain' => $hasVerifiedDomain,
            'primaryVerifiedDomainSlug' => $primaryVerifiedDomainSlug,
            'pendingDomains' => $pendingDomains,
            'setupPhase' => $setupPhase,
            'hasProductionToken' => $hasProductionToken,
            'hasStagingToken' => $hasStagingToken,
            'canCreateStagingToken' => $canCreateStagingToken,
            'domainLimit' => null,
            'domainLimitReached' => false,
            'canPurchaseExtraDomainSlots' => false,
            'canInviteUsers' => $user instanceof User && $user->isAdmin(),
            'mlsCatalogFeedCodes' => $this->mlsFeeds->catalogFeedCodes(),
            'mlsFeedLabels' => $this->mlsFeedLabels(),
            'apiPublicUrl' => rtrim((string) config('idx_urls.api_public_url'), '/'),
            'appUrl' => $appUrl,
            'activePanel' => $activePanel,
        ];
    }

    private function resolvePanel(string $requested): string
    {
        if (isset(self::LEGACY_PANEL_MAP[$requested])) {
            return self::LEGACY_PANEL_MAP[$requested];
        }

        if (in_array($requested, self::PANELS, true)) {
            return $requested;
        }

        return 'setup';
    }

    /**
     * @return array<string, string>
     */
    private function mlsFeedLabels(): array
    {
        $labels = [];
        foreach ($this->mlsFeeds->catalogFeedCodes() as $code) {
            $labels[$code] = $this->mlsFeeds->feedLabel($code);
        }

        return $labels;
    }
}
