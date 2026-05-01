<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Http\Requests\AgentPortal\AgentPortalSettingsRequest;
use App\Models\AgentPortalSetting;
use App\Models\User;
use App\Services\AgentPortal\FeatureFlagService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AgentPortalSettingsController extends Controller
{
    /**
     * @return array<string, mixed>
     */
    private function defaults(): array
    {
        return [
            'notification_email_enabled' => true,
            'notification_sms_enabled' => false,
            'weekly_digest_enabled' => true,
            'alert_default_cadence' => 'daily',
            'timezone' => 'America/New_York',
            'theme_density' => 'compact',
            'onboarding_tips_enabled' => true,
            'hide_agent_onboarding_checklist' => false,
            'feature_flags' => [],
        ];
    }

    public function show(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $settings = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $stored = is_array($settings?->settings_json) ? $settings->settings_json : [];

        return response()->json([
            'data' => array_merge($this->defaults(), $stored),
        ]);
    }

    public function update(AgentPortalSettingsRequest $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $existing = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $current = is_array($existing?->settings_json) ? $existing->settings_json : [];
        $merged = array_merge($this->defaults(), $current, $request->validated());

        $settings = AgentPortalSetting::query()->updateOrCreate(
            ['user_id' => $user->id],
            ['settings_json' => $merged],
        );

        $stored = is_array($settings->settings_json) ? $settings->settings_json : [];

        return response()->json([
            'data' => array_merge($this->defaults(), $stored),
        ]);
    }

    public function dismissOnboardingChecklist(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $existing = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $current = is_array($existing?->settings_json) ? $existing->settings_json : [];
        $merged = array_merge($this->defaults(), $current, [
            'hide_agent_onboarding_checklist' => true,
        ]);

        AgentPortalSetting::query()->updateOrCreate(
            ['user_id' => $user->id],
            ['settings_json' => $merged],
        );

        return response()->json([
            'data' => [
                'hide_agent_onboarding_checklist' => true,
            ],
        ]);
    }

    public function restoreOnboardingChecklist(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $existing = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $current = is_array($existing?->settings_json) ? $existing->settings_json : [];
        $merged = array_merge($this->defaults(), $current, [
            'hide_agent_onboarding_checklist' => false,
        ]);

        AgentPortalSetting::query()->updateOrCreate(
            ['user_id' => $user->id],
            ['settings_json' => $merged],
        );

        return response()->json([
            'data' => [
                'hide_agent_onboarding_checklist' => false,
            ],
        ]);
    }

    public function featureFlags(Request $request, FeatureFlagService $flags): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        return response()->json([
            'data' => [
                'flags' => $flags->getFlagsForUser($user),
                'available_modules' => $flags->getAvailableModules(),
                'global_defaults' => $flags->getGlobalDefaults(),
            ],
        ]);
    }

    public function updateFeatureFlags(Request $request, FeatureFlagService $flags): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $payload = $request->validate([
            'flags' => ['required', 'array'],
            'flags.*' => ['boolean'],
        ]);

        foreach ($payload['flags'] as $module => $enabled) {
            $flags->setFlag($user, (string) $module, (bool) $enabled);
        }

        return response()->json([
            'data' => [
                'flags' => $flags->getFlagsForUser($user),
            ],
        ]);
    }
}
