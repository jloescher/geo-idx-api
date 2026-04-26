<?php

namespace Tests\Feature\Dashboard;

use App\Livewire\Dashboard\ApiTokenManager;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Laravel\Sanctum\PersonalAccessToken;
use Livewire\Livewire;
use Tests\TestCase;

class ApiTokenManagerTest extends TestCase
{
    use RefreshDatabase;

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

    public function test_smart_plan_user_cannot_access_api_token_manager(): void
    {
        config(['billing.plans.smart.stripe_price_monthly' => 'price_smart_monthly']);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_smart_monthly');
        $this->actingAs($user);

        Livewire::test(ApiTokenManager::class)
            ->assertStatus(403);
    }

    public function test_ultra_plan_user_can_render_api_token_manager(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_ultra_monthly');
        $this->actingAs($user);

        Livewire::test(ApiTokenManager::class)
            ->assertSee('API Keys')
            ->assertSet('tokens', []);
    }

    public function test_user_can_create_api_token(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_ultra_monthly');
        $this->actingAs($user);

        $component = Livewire::test(ApiTokenManager::class)
            ->set('tokenName', 'My Ultra Token')
            ->call('createToken')
            ->assertSet('tokenName', '')
            ->assertDispatched('token-created');

        $plainTextToken = $component->get('newToken');
        $this->assertIsString($plainTextToken);
        $this->assertGreaterThan(10, strlen($plainTextToken));

        $this->assertDatabaseHas('personal_access_tokens', [
            'tokenable_type' => User::class,
            'tokenable_id' => $user->id,
            'name' => 'My Ultra Token',
        ]);

        $stored = PersonalAccessToken::query()
            ->where('name', 'My Ultra Token')
            ->firstOrFail();
        $this->assertSame(['idx:access'], $stored->abilities);
    }

    public function test_mega_plan_user_receives_idx_full_abilities(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_mega_monthly');
        $this->actingAs($user);

        Livewire::test(ApiTokenManager::class)
            ->set('tokenName', 'Mega Token')
            ->call('createToken');

        $stored = PersonalAccessToken::query()
            ->where('name', 'Mega Token')
            ->firstOrFail();
        $this->assertSame(['idx:full'], $stored->abilities);
    }

    public function test_user_can_revoke_own_token(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_ultra_monthly');
        $this->actingAs($user);

        $token = $user->createToken('To Revoke', ['idx:access']);

        Livewire::test(ApiTokenManager::class)
            ->call('revokeToken', $token->accessToken->id)
            ->assertDispatched('token-revoked');

        $this->assertDatabaseMissing('personal_access_tokens', [
            'id' => $token->accessToken->id,
        ]);
    }

    public function test_user_cannot_revoke_another_users_token(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
        ]);

        /** @var User $userA */
        $userA = User::factory()->createOne();
        /** @var User $userB */
        $userB = User::factory()->createOne();

        $this->attachSubscription($userA, 'price_ultra_monthly');
        $this->attachSubscription($userB, 'price_ultra_monthly');

        $token = $userA->createToken('User A Token', ['idx:access']);

        $this->actingAs($userB);

        Livewire::test(ApiTokenManager::class)
            ->call('revokeToken', $token->accessToken->id)
            ->assertStatus(403);
    }

    public function test_token_name_is_required_and_max_60(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_ultra_monthly');
        $this->actingAs($user);

        Livewire::test(ApiTokenManager::class)
            ->set('tokenName', '')
            ->call('createToken')
            ->assertHasErrors(['tokenName' => 'required']);

        Livewire::test(ApiTokenManager::class)
            ->set('tokenName', str_repeat('a', 61))
            ->call('createToken')
            ->assertHasErrors(['tokenName' => 'max']);
    }

    public function test_token_list_shows_recent_tokens(): void
    {
        config([
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_monthly',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_ultra_monthly');
        $this->actingAs($user);

        $user->createToken('First Token', ['idx:access']);
        $user->createToken('Second Token', ['idx:access']);

        Livewire::test(ApiTokenManager::class)
            ->assertSee('First Token')
            ->assertSee('Second Token');
    }
}
