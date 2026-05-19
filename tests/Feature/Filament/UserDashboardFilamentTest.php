<?php

namespace Tests\Feature\Filament;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * Manual accessibility (WCAG 2.1 AA): run axe DevTools or Lighthouse on
 * /dashboard?panel=setup|api in light and dark mode after auth.
 */
class UserDashboardFilamentTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();
    }

    public function test_authenticated_user_can_open_dashboard_setup(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/dashboard');

        $response->assertOk();
        $response->assertSee('Setup', false);
        $response->assertSee('Add your site', false);
    }

    public function test_legacy_dashboard_panel_renders_setup(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/dashboard?panel=dashboard');

        $response->assertOk();
        $response->assertSee('Setup', false);
    }

    public function test_legacy_domains_panel_renders_setup(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/dashboard?panel=domains');

        $response->assertOk();
        $response->assertSee('Setup', false);
    }

    public function test_api_panel_renders_api_keys(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/dashboard?panel=api');

        $response->assertOk();
        $response->assertSee('API Keys', false);
    }

    public function test_filament_dashboard_legacy_path_redirects_to_dashboard(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard?panel=domains');

        $response->assertRedirect('https://localhost/dashboard?panel=domains');
        $response->assertStatus(301);
    }
}
