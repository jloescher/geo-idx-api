<?php

namespace App\Filament\Pages;

use App\Filament\Concerns\AuthorizesAgentPortalAccess;
use App\Models\AgentPortalSetting;
use App\Models\User;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;
use Illuminate\Support\Facades\Auth;

class AgentSettingsWorkspace extends Page
{
    use AuthorizesAgentPortalAccess;

    protected string $view = 'filament.pages.agent-settings-workspace';

    protected static ?string $title = 'Settings';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-cog-6-tooth';

    protected static ?string $navigationLabel = 'Settings';

    protected static string|\UnitEnum|null $navigationGroup = 'Agent Portal';

    protected static ?int $navigationSort = 90;

    protected Width|string|null $maxContentWidth = Width::Full;

    protected static function requiredAgentModule(): ?string
    {
        return 'settings';
    }

    /**
     * @return array<string, mixed>
     */
    protected function getViewData(): array
    {
        /** @var User|null $user */
        $user = Auth::user();

        return [
            'onboardingChecklistDismissed' => $user instanceof User && $this->hideAgentOnboardingChecklist($user),
        ];
    }

    private function hideAgentOnboardingChecklist(User $user): bool
    {
        $row = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $json = is_array($row?->settings_json) ? $row->settings_json : [];

        return (bool) ($json['hide_agent_onboarding_checklist'] ?? false);
    }
}
