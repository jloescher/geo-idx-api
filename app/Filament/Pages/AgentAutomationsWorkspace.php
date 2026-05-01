<?php

namespace App\Filament\Pages;

use App\Filament\Concerns\AuthorizesAgentPortalAccess;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;

class AgentAutomationsWorkspace extends Page
{
    use AuthorizesAgentPortalAccess;

    protected string $view = 'filament.pages.agent-automations-workspace';

    protected static ?string $title = 'Automations';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-cog-6-tooth';

    protected static ?string $navigationLabel = 'Automations';

    protected static string|\UnitEnum|null $navigationGroup = 'Agent Portal';

    protected static ?int $navigationSort = 50;

    protected Width|string|null $maxContentWidth = Width::Full;

    protected static function requiredAgentModule(): ?string
    {
        return 'automations';
    }
}
