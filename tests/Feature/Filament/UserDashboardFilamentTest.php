<?php

namespace Tests\Feature\Filament;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

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

        $response = $this->actingAs($user)->get('https://localhost/filament-dashboard?panel=dashboard');

        $response->assertOk();
        $response->assertSee('Verified domains', false);
    }
}
