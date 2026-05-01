<?php

namespace App\Filament\Pages;

use App\Filament\Concerns\AuthorizesAgentPortalAccess;
use App\Models\AgentPortalSetting;
use App\Models\User;
use App\Services\AgentPortal\AgentOnboardingChecklistService;
use App\Services\AgentPortal\FeatureFlagService;
use App\Services\AgentPortal\SearchFieldRegistry;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;
use Illuminate\Support\Facades\Auth;

class AgentPortalOverview extends Page
{
    use AuthorizesAgentPortalAccess;

    protected string $view = 'filament.pages.agent-portal-overview';

    protected static ?string $title = 'Agent overview';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-chart-bar-square';

    protected static ?string $navigationLabel = 'Overview';

    protected static string|\UnitEnum|null $navigationGroup = 'Agent Portal';

    protected static ?int $navigationSort = 10;

    protected Width|string|null $maxContentWidth = Width::Full;

    protected static function requiredAgentModule(): ?string
    {
        return 'dashboard';
    }

    /**
     * @return array<string, mixed>
     */
    protected function getViewData(): array
    {
        /** @var User|null $user */
        $user = Auth::user();
        $scopes = $user instanceof User
            ? app(SubscriberFeedAccessService::class)->resolvedScopesForUser($user)
            : [];

        return [
            'feedScopes' => $scopes,
            'registrySample' => app(SearchFieldRegistry::class)->coreFields(),
            'workspaceShortcuts' => $user instanceof User
                ? $this->workspaceShortcuts($user)
                : [],
            'onboardingChecklist' => $user instanceof User
                ? app(AgentOnboardingChecklistService::class)->checklistForUser($user)
                : [],
            'showOnboardingChecklist' => $user instanceof User && ! $this->hideAgentOnboardingChecklist($user),
        ];
    }

    private function hideAgentOnboardingChecklist(User $user): bool
    {
        $row = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $json = is_array($row?->settings_json) ? $row->settings_json : [];

        return (bool) ($json['hide_agent_onboarding_checklist'] ?? false);
    }

    /**
     * @return list<array{label: string, url: string, slug: string, module: string, enabled: bool, badges?: list<array{key: string, label: string, enabled: bool}>}>
     */
    protected function workspaceShortcuts(User $user): array
    {
        $flags = app(FeatureFlagService::class);
        $panel = 'dashboard';

        /** @var list<array{class-string<Page>, string}> */
        $pairs = [
            [AgentDashboardWorkspace::class, 'dashboard'],
            [AgentSearchWorkspace::class, 'search'],
            [AgentContactsWorkspace::class, 'contacts'],
            [AgentAlertsWorkspace::class, 'alerts'],
            [AgentAutomationsWorkspace::class, 'automations'],
            [AgentMarketingWorkspace::class, 'marketing'],
            [AgentSettingsWorkspace::class, 'settings'],
        ];

        $out = [];
        foreach ($pairs as [$class, $module]) {
            $row = [
                'label' => $class::getNavigationLabel(),
                'url' => $class::getUrl(panel: $panel),
                'slug' => $class::getDefaultSlug(),
                'module' => $module,
                'enabled' => $flags->isEnabled($user, $module),
            ];

            if ($module === 'marketing') {
                $row['badges'] = [
                    ['key' => 'seo_landing_pages', 'label' => 'SEO landings', 'enabled' => $flags->isEnabled($user, 'seo_landing_pages')],
                    ['key' => 'widgets', 'label' => 'Widgets embed', 'enabled' => $flags->isEnabled($user, 'widgets')],
                ];
            } elseif ($module === 'search') {
                $row['badges'] = [
                    ['key' => 'multi_mls', 'label' => 'Multi-MLS', 'enabled' => $flags->isEnabled($user, 'multi_mls')],
                ];
            }

            $out[] = $row;
        }

        return $out;
    }
}
