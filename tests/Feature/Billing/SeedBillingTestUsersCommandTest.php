<?php

namespace Tests\Feature\Billing;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Hash;
use Tests\TestCase;

class SeedBillingTestUsersCommandTest extends TestCase
{
    use RefreshDatabase;

    public function test_seed_test_users_skip_subscription_creates_four_users_without_stripe(): void
    {
        $this->artisan('billing:seed-test-users', [
            '--skip-subscription' => true,
            '--password' => 'seed-test-secret',
        ])->assertSuccessful();

        foreach (['pro', 'smart', 'ultra', 'mega'] as $planKey) {
            $email = "idx-seed-{$planKey}@quantyralabs.test";
            $user = User::query()->where('email', $email)->first();
            $this->assertNotNull($user);
            $this->assertTrue(Hash::check('seed-test-secret', (string) $user->password));
            $this->assertFalse($user->subscribed('default'));
        }
    }

    public function test_seed_test_users_requires_stripe_secret_when_not_skipping(): void
    {
        config(['cashier.secret' => '']);

        $this->artisan('billing:seed-test-users')
            ->assertFailed()
            ->expectsOutputToContain('Stripe is not configured');
    }
}
