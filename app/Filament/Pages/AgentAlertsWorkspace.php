<?php

namespace App\Filament\Pages;

use App\Filament\Concerns\AuthorizesAgentPortalAccess;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;

class AgentAlertsWorkspace extends Page
{
    use AuthorizesAgentPortalAccess;

    protected string $view = 'filament.pages.agent-alerts-workspace';

    protected static ?string $title = 'Email alerts';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-bell-alert';

    protected static ?string $navigationLabel = 'Email alerts';

    protected static string|\UnitEnum|null $navigationGroup = 'Agent Portal';

    protected static ?int $navigationSort = 40;

    protected Width|string|null $maxContentWidth = Width::Full;

    protected static function requiredAgentModule(): ?string
    {
        return 'alerts';
    }
}
