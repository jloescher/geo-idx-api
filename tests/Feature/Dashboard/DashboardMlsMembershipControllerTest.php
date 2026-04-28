<?php

namespace Tests\Feature\Dashboard;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class DashboardMlsMembershipControllerTest extends TestCase
{
    use RefreshDatabase;

    public function test_user_can_submit_mls_credentials_and_be_marked_active_with_local_fallback(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'mls_membership_status' => 'pending',
        ]);
        $this->actingAs($user);

        $this->post('http://localhost/dashboard/mls-membership', [
            'mls_id' => 'MLS5566',
            'mls_email' => 'agent@example.test',
        ])->assertRedirect('http://localhost/dashboard?panel=onboarding');

        $this->assertDatabaseHas('users', [
            'id' => $user->id,
            'mls_id' => 'MLS5566',
            'mls_email' => 'agent@example.test',
            'mls_membership_status' => 'active',
        ]);
    }

    public function test_user_sees_verification_error_when_provider_returns_inactive(): void
    {
        config(['services.mls_membership.endpoint' => 'https://mls-provider.test/verify']);

        Http::fake([
            'https://mls-provider.test/verify' => Http::response(['active' => false], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'mls_membership_status' => 'pending',
        ]);
        $this->actingAs($user);

        $this->from('http://localhost/dashboard?panel=onboarding')
            ->post('http://localhost/dashboard/mls-membership', [
                'mls_id' => 'MLS7788',
                'mls_email' => 'inactive@example.test',
            ])
            ->assertRedirect('http://localhost/dashboard?panel=onboarding')
            ->assertSessionHasErrors(['mls_membership']);

        $this->assertDatabaseHas('users', [
            'id' => $user->id,
            'mls_membership_status' => 'inactive',
            'mls_membership_last_error' => 'MLS membership is not active.',
        ]);
    }
}
