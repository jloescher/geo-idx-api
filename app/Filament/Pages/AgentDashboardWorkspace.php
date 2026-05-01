<?php

namespace App\Filament\Pages;

use App\Filament\Concerns\AuthorizesAgentPortalAccess;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;

class AgentDashboardWorkspace extends Page
{
    use AuthorizesAgentPortalAccess;

    protected string $view = 'filament.pages.agent-dashboard-workspace';

    protected static ?string $title = 'Agent Dashboard';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-home-modern';

    protected static ?string $navigationLabel = 'Dashboard';

    protected static string|\UnitEnum|null $navigationGroup = 'Agent Portal';

    protected static ?int $navigationSort = 10;

    protected Width|string|null $maxContentWidth = Width::Full;

    protected static function requiredAgentModule(): ?string
    {
        return 'dashboard';
    }
}
