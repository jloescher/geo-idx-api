<?php

namespace App\Filament\Pages;

use App\Filament\Concerns\AuthorizesAgentPortalAccess;
use App\Services\AgentPortal\SearchFieldRegistry;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;

class AgentSearchWorkspace extends Page
{
    use AuthorizesAgentPortalAccess;

    protected string $view = 'filament.pages.agent-search-workspace';

    protected static ?string $title = 'Search workspace';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-map';

    protected static ?string $navigationLabel = 'Search';

    protected static string|\UnitEnum|null $navigationGroup = 'Agent Portal';

    protected static ?int $navigationSort = 20;

    protected Width|string|null $maxContentWidth = Width::Full;

    protected static function requiredAgentModule(): ?string
    {
        return 'search';
    }

    /**
     * @return array<string, mixed>
     */
    protected function getViewData(): array
    {
        return [
            'coreFields' => app(SearchFieldRegistry::class)->coreFields(),
        ];
    }
}
