<?php

namespace App\Filament\Pages;

use App\Filament\Concerns\AuthorizesAgentPortalAccess;
use App\Models\User;
use App\Services\AgentPortal\FeatureFlagService;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;
use Illuminate\Support\Facades\Auth;

class AgentMarketingWorkspace extends Page
{
    use AuthorizesAgentPortalAccess;

    protected string $view = 'filament.pages.agent-marketing-workspace';

    protected static ?string $title = 'Marketing';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-megaphone';

    protected static ?string $navigationLabel = 'Marketing';

    protected static string|\UnitEnum|null $navigationGroup = 'Agent Portal';

    protected static ?int $navigationSort = 60;

    protected Width|string|null $maxContentWidth = Width::Full;

    protected static function requiredAgentModule(): ?string
    {
        return 'marketing';
    }

    /**
     * @return array{marketingFeatureFlags: array{widgets: bool, seo_landing_pages: bool}, compsDatasetCodes: list<string>}
     */
    protected function getViewData(): array
    {
        /** @var User|null $user */
        $user = Auth::user();
        $flags = app(FeatureFlagService::class);

        $defaults = [
            'marketingFeatureFlags' => [
                'widgets' => false,
                'seo_landing_pages' => false,
            ],
            'compsDatasetCodes' => [(string) config('bridge.dataset', 'stellar')],
        ];

        if (! $user instanceof User) {
            return $defaults;
        }

        $datasets = [];
        foreach (app(SubscriberFeedAccessService::class)->detailedFeedScopesForUser($user) as $scope) {
            if (($scope['can_search'] ?? false) !== true) {
                continue;
            }
            $code = trim((string) ($scope['dataset_code'] ?? ''));
            if ($code !== '' && ! in_array($code, $datasets, true)) {
                $datasets[] = $code;
            }
        }
        sort($datasets);
        if ($datasets === []) {
            $datasets = [(string) config('bridge.dataset', 'stellar')];
        }

        return [
            'marketingFeatureFlags' => [
                'widgets' => $flags->isEnabled($user, 'widgets'),
                'seo_landing_pages' => $flags->isEnabled($user, 'seo_landing_pages'),
            ],
            'compsDatasetCodes' => $datasets,
        ];
    }
}
