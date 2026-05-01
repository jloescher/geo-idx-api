<?php

namespace Tests\Feature;

use App\Filament\Pages\AgentAlertsWorkspace;
use App\Filament\Pages\AgentAutomationsWorkspace;
use App\Filament\Pages\AgentContactsWorkspace;
use App\Filament\Pages\AgentDashboardWorkspace;
use App\Filament\Pages\AgentSearchWorkspace;
use App\Models\User;
use App\Services\AgentPortal\FeatureFlagService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class FilamentAgentPortalTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();
    }

    public function test_guest_cannot_view_agent_portal_overview(): void
    {
        $response = $this->get('https://localhost/filament-dashboard/agent-portal-overview');

        $response->assertRedirect();
    }

    public function test_authenticated_user_can_view_agent_portal_overview(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-portal-overview');

        $response->assertOk();
        $response->assertSee('MLS');
        $response->assertSee('dataset access');
        $response->assertSee('Workspace shortcuts', false);
        $response->assertSee('data-agent-subfeature-badge="seo_landing_pages"', false);
        $response->assertSee('data-agent-subfeature-badge="widgets"', false);
        $response->assertSee('data-agent-subfeature-badge="multi_mls"', false);
        $response->assertSee('Getting started', false);
        $response->assertSee('data-agent-onboarding-checklist', false);
    }

    public function test_agent_portal_overview_hides_checklist_after_dismiss_endpoint(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/dismiss')->assertOk();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-portal-overview');

        $response->assertOk();
        $response->assertDontSee('data-agent-onboarding-checklist', false);
        $response->assertDontSee('Getting started', false);
    }

    public function test_agent_portal_overview_shows_checklist_again_after_restore_endpoint(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/dismiss')->assertOk();
        $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/restore')->assertOk();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-portal-overview');

        $response->assertOk();
        $response->assertSee('Getting started', false);
        $response->assertSee('data-agent-onboarding-checklist', false);
    }

    public function test_agent_settings_workspace_shows_restore_when_checklist_dismissed(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/dismiss')->assertOk();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-settings-workspace');

        $response->assertOk();
        $response->assertSee('Show checklist on overview', false);
    }

    public function test_agent_settings_workspace_hides_restore_panel_when_checklist_visible(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-settings-workspace');

        $response->assertOk();
        $response->assertDontSee('Show checklist on overview', false);
    }

    public function test_authenticated_user_can_view_agent_search_workspace(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-search-workspace');

        $response->assertOk();
        $response->assertSee('Map + filters', false);
        $response->assertSee('Auto-fit map to results', false);
        $response->assertSee('Search as you pan (debounced)', false);
        $response->assertSee('OpenStreetMap', false);
        $response->assertSee('My location', false);
        $response->assertSee('Zoom to drawn shapes', false);
        $response->assertSee('Fullscreen map', false);
    }

    public function test_authenticated_user_can_view_agent_dashboard_upcoming_alerts_panel(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-dashboard-workspace');

        $response->assertOk();
        $response->assertSee('Upcoming alerts', false);
        $response->assertSee('idxAgentUpcomingAlerts', false);
        $response->assertSee('Manage in Email alerts', false);
        $this->assertTrue(AgentDashboardWorkspace::shouldRegisterNavigation());
    }

    public function test_authenticated_user_can_view_agent_contacts_workspace_detail_tabs(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-contacts-workspace');

        $response->assertOk();
        $response->assertSee('Email activity', false);
        $response->assertSee('Site activity', false);
        $response->assertSee('Full timeline', false);
        $this->assertTrue(AgentContactsWorkspace::shouldRegisterNavigation());
    }

    public function test_authenticated_user_can_view_agent_alerts_workspace_type_tabs(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-alerts-workspace');

        $response->assertOk();
        $response->assertSee('All types', false);
        $response->assertSee('Market', false);
        $response->assertSee('Home value', false);
        $response->assertSee('View runs', false);
        $this->assertTrue(AgentAlertsWorkspace::shouldRegisterNavigation());
    }

    public function test_disabled_search_module_blocks_agent_search_workspace(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'search', false);

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-search-workspace');

        $response->assertForbidden();
    }

    public function test_automations_workspace_is_blocked_when_feature_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-automations-workspace');

        $response->assertForbidden();
    }

    public function test_automations_workspace_can_be_viewed_when_feature_enabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'automations', true);

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-automations-workspace');

        $response->assertOk();
        $response->assertSee('Automations');
    }

    public function test_overview_marks_disabled_automations_shortcut(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-portal-overview');

        $response->assertOk();
        $response->assertSee('data-agent-workspace-shortcut="agent-automations-workspace"', false);
        $response->assertSee('text-amber-600 dark:text-amber-400');
    }

    public function test_search_navigation_not_registered_when_module_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        app(FeatureFlagService::class)->setFlag($user, 'search', false);

        $this->assertFalse(AgentSearchWorkspace::shouldRegisterNavigation());
    }

    public function test_search_navigation_registers_when_module_enabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        app(FeatureFlagService::class)->setFlag($user, 'search', true);

        $this->assertTrue(AgentSearchWorkspace::shouldRegisterNavigation());
    }

    public function test_automations_navigation_not_registered_by_default(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $this->assertFalse(AgentAutomationsWorkspace::shouldRegisterNavigation());
    }

    public function test_agent_feed_access_json_returns_scopes(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/feed-access');

        $response->assertOk();
        $response->assertJsonStructure([
            'scopes' => [
                '*' => ['mls_code', 'dataset_code', 'feed_id', 'status', 'permissions_json', 'can_search', 'can_alerts'],
            ],
        ]);
        $this->assertNotEmpty($response->json('scopes'));
        $this->assertTrue($response->json('scopes.0.can_search'));
        $this->assertTrue($response->json('scopes.0.can_alerts'));
    }

    public function test_marketing_workspace_shows_widget_builder_when_widgets_enabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'search', true);
        app(FeatureFlagService::class)->setFlag($user, 'marketing', true);
        app(FeatureFlagService::class)->setFlag($user, 'widgets', true);
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-marketing-workspace');

        $response->assertOk();
        $response->assertSee('Widget embed code generator', false);
        $response->assertSee('SEO landing template', false);
        $response->assertSee('Home value quick estimate', false);
    }

    public function test_marketing_workspace_hides_widget_builder_when_widgets_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'search', true);
        app(FeatureFlagService::class)->setFlag($user, 'marketing', true);
        app(FeatureFlagService::class)->setFlag($user, 'widgets', false);

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-marketing-workspace');

        $response->assertOk();
        $response->assertDontSee('Widget embed code generator', false);
        $response->assertSee('Widget embed code generation is turned off', false);
    }

    public function test_marketing_workspace_hides_seo_controls_when_seo_feature_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'search', true);
        app(FeatureFlagService::class)->setFlag($user, 'marketing', true);
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', false);

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard/agent-marketing-workspace');

        $response->assertOk();
        $response->assertDontSee('value="seo_landing"', false);
        $response->assertDontSee('SEO templates only', false);
        $response->assertDontSee('>SEO templates</p>', false);
        $response->assertSee('seo_landing_pages', false);
    }
}
