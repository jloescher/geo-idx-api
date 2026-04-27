<?php

namespace Tests\Feature\Billing;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Testing\TestResponse;
use Tests\TestCase;

class SubscriptionCheckoutGateTest extends TestCase
{
    use RefreshDatabase;

    private function platformRequest(string $method, string $path, array $query = []): TestResponse
    {
        $uri = 'https://localhost'.$path.($query !== [] ? '?'.http_build_query($query) : '');

        return $this->call($method, $uri, [], [], [], [
            'HTTP_HOST' => 'localhost',
            'HTTPS' => 'on',
        ]);
    }

    public function test_guest_is_redirected_to_login_when_visiting_checkout(): void
    {
        $response = $this->platformRequest('GET', '/billing/checkout', [
            'plan' => 'pro',
            'interval' => 'monthly',
        ]);

        $response->assertRedirect(route('login'));
    }

    public function test_authenticated_user_without_stripe_price_ids_is_redirected_to_pricing_with_flash(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => null]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'mls_id' => 'MLS1234',
            'mls_email' => 'idx-seed-mega@quantyralabs.test',
            'assigned_mls_datasets' => ['stellar'],
            'mls_membership_status' => 'active',
            'mls_membership_verified_at' => now(),
        ]);
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'checkout-gate.example.com',
            'is_active' => true,
            'verification_status' => 'verified',
            'verification_method' => 'txt',
            'txt_verification_name' => '_geoidx.checkout-gate.example.com',
            'txt_verification_value' => 'geoidx-verify=test',
            'txt_verified_at' => now(),
        ]);

        $this->actingAs($user);

        $response = $this->platformRequest('GET', '/billing/checkout', [
            'plan' => 'pro',
            'interval' => 'monthly',
        ]);

        $response->assertRedirect('/#pricing');
        $response->assertSessionHas('flash_billing_error');
    }

    public function test_checkout_rejects_invalid_plan(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'mls_id' => 'MLS1234',
            'mls_email' => 'idx-seed-mega@quantyralabs.test',
            'assigned_mls_datasets' => ['stellar'],
            'mls_membership_status' => 'active',
            'mls_membership_verified_at' => now(),
        ]);
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'checkout-invalid-plan.example.com',
            'is_active' => true,
            'verification_status' => 'verified',
            'verification_method' => 'txt',
            'txt_verification_name' => '_geoidx.checkout-invalid-plan.example.com',
            'txt_verification_value' => 'geoidx-verify=test',
            'txt_verified_at' => now(),
        ]);

        $this->actingAs($user);

        $response = $this->platformRequest('GET', '/billing/checkout', [
            'plan' => 'invalid',
            'interval' => 'monthly',
        ]);

        $response->assertSessionHasErrors(['plan']);
    }
}
