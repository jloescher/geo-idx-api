<?php

namespace Tests\Feature\Dashboard;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Sync\Models\QuantyraLead;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class DashboardMetricsTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        $this->withoutVite();
    }

    public function test_dashboard_bootstraps_preview_site_key_from_latest_enabled_ghl_registered_url(): void
    {
        $oauth = $this->createToken('co-prev', 'loc-prev');
        GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-prev',
            'ghl_company_id' => 'co-prev',
            'primary_url' => 'https://preview-client.example',
            'widget_api_key' => 'qh_latestkey1234567890123456789012xy',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $html = (string) $this->get('http://localhost/dashboard')->getContent();
        $this->assertStringContainsString('qh_latestkey1234567890123456789012xy', $html);
    }

    public function test_dashboard_shows_unavailable_lead_state_without_valid_widget_key_scope(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $this->get('http://localhost/dashboard')
            ->assertOk()
            ->assertSee('Not available')
            ->assertSeeText('Provide a valid widget site key to scope lead telemetry.');
    }

    public function test_dashboard_index_x_data_does_not_embed_a_raw_script_closing_token_for_copy_embed(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $html = (string) $this->get('http://localhost/dashboard')->getContent();
        $this->assertStringNotContainsString('async></script', $html);
        $this->assertStringContainsString('x-data="window.__createDashboardAlpineState(', $html);
    }

    public function test_onboarding_panel_shows_checklist_and_setup_forms(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $this->get('http://localhost/dashboard?panel=onboarding')
            ->assertOk()
            ->assertSee('Getting Started Checklist')
            ->assertSee('MLS membership verification')
            ->assertSee('MLS ID')
            ->assertSee('MLS Email')
            ->assertSee('Verify MLS Membership')
            ->assertSee('My Approved Domains')
            ->assertSee('Add Domain')
            ->assertSee('Open Domains command center');
    }

    public function test_onboarding_panel_locks_domain_add_after_first_domain(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'one.example.com',
            'is_active' => true,
        ]);
        $this->actingAs($user);

        $this->get('http://localhost/dashboard?panel=onboarding')
            ->assertOk()
            ->assertSee('Onboarding supports one initial widget domain. Add additional domains in the Domains tab.');
    }

    public function test_domains_panel_is_domain_only_command_center(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $this->get('http://localhost/dashboard?panel=domains')
            ->assertOk()
            ->assertSee('Domains Command Center')
            ->assertSee('Register additional domain')
            ->assertSee('Domain verification queue')
            ->assertDontSee('Subscription Details')
            ->assertDontSee('API Access');
    }

    public function test_dashboard_shows_real_scoped_lead_count_when_widget_key_is_supplied(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $token = $this->createToken('co-metrics', 'loc-metrics');
        GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $token->id,
            'ghl_location_id' => 'loc-metrics',
            'ghl_company_id' => 'co-metrics',
            'primary_url' => 'http://localhost',
            'widget_api_key' => 'qh_metricskey1234567890123456789012',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        QuantyraLead::query()->create([
            'ghl_location_id' => 'loc-metrics',
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['email' => 'a@example.com'],
            'created_at' => now()->startOfMonth()->addDay(),
            'updated_at' => now()->startOfMonth()->addDay(),
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'loc-metrics',
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['email' => 'b@example.com'],
            'created_at' => now()->startOfMonth()->addDays(2),
            'updated_at' => now()->startOfMonth()->addDays(2),
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'loc-metrics',
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['email' => 'old@example.com'],
            'created_at' => now()->subMonth()->startOfMonth(),
            'updated_at' => now()->subMonth()->startOfMonth(),
        ]);

        $this->get('http://localhost/dashboard?widget_api_key=qh_metricskey1234567890123456789012')
            ->assertOk()
            ->assertDontSee('Provide a valid widget API key to scope lead telemetry.')
            ->assertSee('Leads this month')
            ->assertSee('2');
    }

    private function createToken(string $companyId, string $locationId): GhlOAuthToken
    {
        $access = 'access-'.$companyId.'-'.$locationId.'-'.uniqid('', true);
        $refresh = 'refresh-'.$companyId.'-'.$locationId.'-'.uniqid('', true);

        $token = new GhlOAuthToken([
            'ghl_company_id' => $companyId,
            'ghl_location_id' => $locationId,
            'ghl_user_id' => 'u1',
            'user_type' => 'Location',
            'scopes' => 'contacts.write',
            'is_bulk_install' => false,
            'expires_at' => now()->addDay(),
            'status' => 'active',
            'access_token_hash' => hash('sha256', $access),
        ]);
        $token->access_token = $access;
        $token->refresh_token = $refresh;
        $token->save();

        return $token->fresh();
    }
}
