<?php

namespace App\Services\AgentPortal;

use App\Models\AgentPortalSetting;
use App\Models\User;

final class FeatureFlagService
{
    private const MODULE_FLAGS = [
        'dashboard',
        'contacts',
        'search',
        'alerts',
        'automations',
        'marketing',
        'settings',
        'widgets',
        'seo_landing_pages',
        'multi_mls',
    ];

    private const GLOBAL_DEFAULTS = [
        'dashboard' => true,
        'contacts' => true,
        'search' => true,
        'alerts' => true,
        'automations' => false,
        'marketing' => true,
        'settings' => true,
        'widgets' => false,
        'seo_landing_pages' => false,
        'multi_mls' => false,
    ];

    /**
     * Check whether a feature module is enabled for the given user.
     */
    public function isEnabled(User $user, string $module): bool
    {
        $flags = $this->getFlagsForUser($user);

        return (bool) ($flags[$module] ?? self::GLOBAL_DEFAULTS[$module] ?? false);
    }

    /**
     * Get all feature flags for a user, merged with global defaults.
     *
     * @return array<string, bool>
     */
    public function getFlagsForUser(User $user): array
    {
        $setting = AgentPortalSetting::query()
            ->where('user_id', $user->id)
            ->first();

        $userFlags = [];
        if ($setting instanceof AgentPortalSetting && is_array($setting->settings_json)) {
            $userFlags = (array) ($setting->settings_json['feature_flags'] ?? []);
        }

        $merged = self::GLOBAL_DEFAULTS;
        foreach ($userFlags as $key => $value) {
            if (in_array($key, self::MODULE_FLAGS, true)) {
                $merged[$key] = (bool) $value;
            }
        }

        return $merged;
    }

    /**
     * Set a feature flag for a user.
     */
    public function setFlag(User $user, string $module, bool $enabled): void
    {
        if (! in_array($module, self::MODULE_FLAGS, true)) {
            return;
        }

        $setting = AgentPortalSetting::query()
            ->where('user_id', $user->id)
            ->firstOrNew(['user_id' => $user->id]);

        $settings = is_array($setting->settings_json) ? $setting->settings_json : [];
        $settings['feature_flags'] ?? $settings['feature_flags'] = [];
        $settings['feature_flags'][$module] = $enabled;

        $setting->settings_json = $settings;
        $setting->save();
    }

    /**
     * @return list<string>
     */
    public function getAvailableModules(): array
    {
        return self::MODULE_FLAGS;
    }

    /**
     * @return array<string, bool>
     */
    public function getGlobalDefaults(): array
    {
        return self::GLOBAL_DEFAULTS;
    }
}
