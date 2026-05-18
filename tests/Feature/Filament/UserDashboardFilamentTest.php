<?php

namespace Tests\Feature\Filament;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * Manual accessibility (WCAG 2.1 AA): run axe DevTools or Lighthouse on
 * /dashboard?panel=dashboard|domains|api in light and dark mode after auth.
 */
class UserDashboardFilamentTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();
    }

    public function test_authenticated_user_can_open_filament_user_dashboard(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/dashboard?panel=dashboard');

        $response->assertOk();
        $response->assertSee('Verified domains', false);
    }

    public function test_filament_dashboard_legacy_path_redirects_to_dashboard_preserving_query(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard?panel=domains');

        $response->assertRedirect('https://localhost/dashboard?panel=domains');
        $response->assertStatus(301);
    }
}
