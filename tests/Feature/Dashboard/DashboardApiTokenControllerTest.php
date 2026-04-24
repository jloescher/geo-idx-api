<?php

namespace Tests\Feature\Dashboard;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Testing\TestResponse;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Laravel\Sanctum\PersonalAccessToken;
use Tests\TestCase;

class DashboardApiTokenControllerTest extends TestCase
{
    use RefreshDatabase;

    private function platformRequest(string $method, string $path, array $data = []): TestResponse
    {
        $uri = 'https://localhost'.$path;

        return $this->call($method, $uri, $data, [], [], [
            'HTTP_HOST' => 'localhost',
            'HTTPS' => 'on',
        ]);
    }

    private function attachSubscription(User $user, string $price): void
    {
        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_test_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => $price,
            'quantity' => 1,
            'trial_ends_at' => now()->addDays(7),
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_test_'.$user->id,
            'stripe_product' => 'prod_test',
            'stripe_price' => $price,
            'quantity' => 100,
        ]);
    }

    public function test_smart_plan_user_cannot_create_dashboard_api_token(): void
    {
        config([
            'billing.plans.smart.stripe_price_monthly' => 'price_smart_monthly',
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_smart_monthly');
        $this->actingAs($user);

        $response = $this->platformRequest('POST', '/dashboard/api-tokens', [
            'token_name' => 'Smart test token',
        ]);

        $response->assertForbidden();
    }

    public function test_ultra_plan_user_can_create_dashboard_api_token(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.ultra.stripe_price_yearly' => 'price_ultra_yearly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
            'billing.plans.mega.stripe_price_yearly' => 'price_mega_yearly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_ultra_monthly');
        $this->actingAs($user);

        $response = $this->platformRequest('POST', '/dashboard/api-tokens', [
            'token_name' => 'Ultra test token',
        ]);

        $response->assertRedirect('/dashboard');
        $response->assertSessionHas('dashboard_new_api_token');
        $this->assertDatabaseHas('personal_access_tokens', [
            'tokenable_type' => User::class,
            'tokenable_id' => $user->id,
            'name' => 'Ultra test token',
        ]);
        $stored = PersonalAccessToken::query()->where('name', 'Ultra test token')->firstOrFail();
        $this->assertSame(['idx:access'], $stored->abilities);
    }

    public function test_mega_plan_user_receives_idx_full_token(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
            'billing.plans.mega.stripe_price_yearly' => 'price_mega_yearly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_mega_monthly');
        $this->actingAs($user);

        $response = $this->platformRequest('POST', '/dashboard/api-tokens', [
            'token_name' => 'Mega test token',
        ]);

        $response->assertRedirect('/dashboard');
        $stored = PersonalAccessToken::query()->where('name', 'Mega test token')->firstOrFail();
        $this->assertSame(['idx:full'], $stored->abilities);
    }
}
