<?php

namespace App\Filament\Concerns;

use App\Models\User;
use App\Services\AgentPortal\FeatureFlagService;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Gate;

trait AuthorizesAgentPortalAccess
{
    public static function shouldRegisterNavigation(): bool
    {
        return static::canAccess();
    }

    public static function canAccess(): bool
    {
        if (! Auth::check() || ! Gate::allows('viewAgentPortal')) {
            return false;
        }

        /** @var User|null $user */
        $user = Auth::user();
        $module = static::requiredAgentModule();

        if ($user instanceof User && is_string($module) && $module !== '') {
            return app(FeatureFlagService::class)->isEnabled($user, $module);
        }

        return true;
    }

    protected static function requiredAgentModule(): ?string
    {
        return null;
    }
}
