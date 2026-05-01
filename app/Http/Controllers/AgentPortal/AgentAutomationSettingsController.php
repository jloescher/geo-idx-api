<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Http\Requests\AgentPortal\AgentAutomationSettingsRequest;
use App\Http\Requests\AgentPortal\AgentIntegrationLifecycleRequest;
use App\Models\AgentAutomationSetting;
use App\Models\User;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AgentAutomationSettingsController extends Controller
{
    /**
     * @return array<string, mixed>
     */
    private function defaultSettings(): array
    {
        return [
            'nurture_enabled' => false,
            'nurture_mode' => 'off',
            'dry_run' => true,
            'eligibility_tags' => [],
            'integration_health' => 'disconnected',
            'last_sync_at' => null,
            'notes' => null,
            'integrations' => [],
        ];
    }

    public function show(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $settings = AgentAutomationSetting::query()
            ->where('user_id', $user->id)
            ->first();

        return response()->json([
            'data' => $settings?->settings_json ?? $this->defaultSettings(),
        ]);
    }

    public function update(AgentAutomationSettingsRequest $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $settings = AgentAutomationSetting::query()->updateOrCreate(
            ['user_id' => $user->id],
            ['settings_json' => $request->validated()],
        );

        return response()->json([
            'data' => $settings->settings_json ?? [],
        ]);
    }

    public function connect(AgentIntegrationLifecycleRequest $request): JsonResponse
    {
        return $this->applyIntegrationAction($request, 'connected');
    }

    public function reconnect(AgentIntegrationLifecycleRequest $request): JsonResponse
    {
        return $this->applyIntegrationAction($request, 'connected');
    }

    public function disconnect(AgentIntegrationLifecycleRequest $request): JsonResponse
    {
        return $this->applyIntegrationAction($request, 'disconnected');
    }

    private function applyIntegrationAction(AgentIntegrationLifecycleRequest $request, string $status): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $provider = (string) $request->validated()['provider'];
        $settings = AgentAutomationSetting::query()
            ->firstOrCreate(
                ['user_id' => $user->id],
                ['settings_json' => $this->defaultSettings()],
            );

        $payload = is_array($settings->settings_json) ? $settings->settings_json : $this->defaultSettings();
        $integrations = isset($payload['integrations']) && is_array($payload['integrations'])
            ? $payload['integrations']
            : [];
        $integrations[$provider] = [
            'status' => $status,
            'updated_at' => now()->toIso8601String(),
        ];
        $payload['integrations'] = $integrations;
        $payload['integration_health'] = $status === 'connected' ? 'connected' : 'disconnected';
        $payload['last_sync_at'] = now()->toIso8601String();

        $settings->settings_json = $payload;
        $settings->save();

        return response()->json(['data' => $payload]);
    }
}
