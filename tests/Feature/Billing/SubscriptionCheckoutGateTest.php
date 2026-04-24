<?php

namespace Tests\Feature\Billing;

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
        /** @var User $user */
        $user = User::factory()->createOne();

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
        $user = User::factory()->createOne();

        $this->actingAs($user);

        $response = $this->platformRequest('GET', '/billing/checkout', [
            'plan' => 'invalid',
            'interval' => 'monthly',
        ]);

        $response->assertSessionHasErrors(['plan']);
    }
}
