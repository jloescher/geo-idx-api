<?php

namespace Tests\Feature;

use App\Ghl\Sync\Models\QuantyraLead;
use App\Models\AgentAlert;
use App\Models\AgentPortalSetting;
use App\Models\AgentSearch;
use App\Models\AgentShareLink;
use App\Models\User;
use App\Services\AgentPortal\AgentOnboardingChecklistService;
use App\Services\AgentPortal\FeatureFlagService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AgentOnboardingChecklistTest extends TestCase
{
    use RefreshDatabase;

    public function test_feed_step_done_when_user_has_assigned_mls_dataset(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $steps = app(AgentOnboardingChecklistService::class)->checklistForUser($user);
        $feed = collect($steps)->firstWhere('id', 'mls_feed_access');

        $this->assertNotNull($feed);
        $this->assertTrue($feed['done']);
    }

    public function test_feed_step_done_when_account_uses_default_dataset_fallback(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => [],
        ]);

        $steps = app(AgentOnboardingChecklistService::class)->checklistForUser($user);
        $feed = collect($steps)->firstWhere('id', 'mls_feed_access');

        $this->assertNotNull($feed);
        $this->assertTrue($feed['done'], 'Empty assigned_mls_datasets still resolves to default MLS scope via User::assignedDatasets().');
    }

    public function test_search_step_blocked_when_search_module_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        app(FeatureFlagService::class)->setFlag($user, 'search', false);

        $steps = app(AgentOnboardingChecklistService::class)->checklistForUser($user);
        $searchStep = collect($steps)->firstWhere('id', 'saved_search');

        $this->assertTrue($searchStep['blocked']);
        $this->assertSame('Enable in Settings', $searchStep['cta_label']);
    }

    public function test_contact_step_done_when_scoped_lead_exists(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Onboard Lead', 'status' => 'new'],
        ]);

        $steps = app(AgentOnboardingChecklistService::class)->checklistForUser($user);
        $contactStep = collect($steps)->firstWhere('id', 'first_contact');

        $this->assertTrue($contactStep['done']);
    }

    public function test_share_link_step_done_when_link_exists(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'onboard-share-test',
            'attribution_json' => [],
        ]);

        $steps = app(AgentOnboardingChecklistService::class)->checklistForUser($user);
        $shareStep = collect($steps)->firstWhere('id', 'share_link');

        $this->assertTrue($shareStep['done']);
    }

    public function test_alert_step_done_when_active_alert_exists(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Onboard alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);

        $steps = app(AgentOnboardingChecklistService::class)->checklistForUser($user);
        $alertStep = collect($steps)->firstWhere('id', 'active_alert');

        $this->assertTrue($alertStep['done']);
    }

    public function test_saved_search_step_done_when_search_exists(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Checklist search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $steps = app(AgentOnboardingChecklistService::class)->checklistForUser($user);
        $searchStep = collect($steps)->firstWhere('id', 'saved_search');

        $this->assertTrue($searchStep['done']);
    }

    public function test_user_can_dismiss_agent_onboarding_checklist_via_settings_endpoint(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/dismiss');

        $response->assertOk();
        $response->assertJsonPath('data.hide_agent_onboarding_checklist', true);

        $row = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $this->assertNotNull($row);
        $this->assertTrue((bool) ($row->settings_json['hide_agent_onboarding_checklist'] ?? false));

        $show = $this->actingAs($user)->getJson('https://localhost/agent/settings');
        $show->assertOk();
        $show->assertJsonPath('data.hide_agent_onboarding_checklist', true);
    }

    public function test_dismiss_onboarding_checklist_preserves_other_settings_keys(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        AgentPortalSetting::query()->create([
            'user_id' => $user->id,
            'settings_json' => [
                'timezone' => 'America/Denver',
                'theme_density' => 'comfortable',
            ],
        ]);

        $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/dismiss')->assertOk();

        $row = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $this->assertSame('America/Denver', $row->settings_json['timezone'] ?? null);
        $this->assertTrue((bool) ($row->settings_json['hide_agent_onboarding_checklist'] ?? false));
    }

    public function test_user_can_restore_agent_onboarding_checklist_via_settings_endpoint(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/dismiss')->assertOk();

        $response = $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/restore');

        $response->assertOk();
        $response->assertJsonPath('data.hide_agent_onboarding_checklist', false);

        $row = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $this->assertNotNull($row);
        $this->assertFalse((bool) ($row->settings_json['hide_agent_onboarding_checklist'] ?? false));
    }
}
