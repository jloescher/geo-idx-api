<?php

namespace Tests\Unit\Billing;

use App\Billing\SubscriptionCatalog;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class SubscriptionCatalogTest extends TestCase
{
    use RefreshDatabase;

    public function test_plan_key_for_price_id_resolves_monthly_and_yearly(): void
    {
        config([
            'billing.plans.pro.stripe_price_monthly' => 'price_pro_m',
            'billing.plans.pro.stripe_price_yearly' => 'price_pro_y',
        ]);

        $catalog = new SubscriptionCatalog;

        $this->assertSame('pro', $catalog->planKeyForPriceId('price_pro_m'));
        $this->assertSame('pro', $catalog->planKeyForPriceId('price_pro_y'));
        $this->assertNull($catalog->planKeyForPriceId('price_unknown'));
        $this->assertNull($catalog->planKeyForPriceId(null));
        $this->assertNull($catalog->planKeyForPriceId(''));
    }

    public function test_plan_key_for_user_checks_all_subscription_prices_not_only_first_item(): void
    {
        config([
            'billing.plans.pro.stripe_price_monthly' => 'price_pro_m',
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_m',
        ]);

        $catalog = new SubscriptionCatalog;
        $user = User::factory()->createOne();

        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_test_metered_first',
            'stripe_status' => 'active',
            'stripe_price' => 'price_metered_only',
            'quantity' => 1,
            'trial_ends_at' => null,
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_metered',
            'stripe_product' => 'prod_metered',
            'stripe_price' => 'price_metered_only',
            'quantity' => 1,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_pro',
            'stripe_product' => 'prod_pro',
            'stripe_price' => 'price_pro_m',
            'quantity' => 1,
        ]);

        $this->assertSame('pro', $catalog->planKeyForUser($user->fresh()));
    }

    public function test_user_may_create_idx_proxy_api_tokens_only_for_valid_ultra_or_mega(): void
    {
        config([
            'billing.plans.smart.stripe_price_monthly' => 'price_smart_m',
            'billing.plans.ultra.stripe_price_monthly' => 'price_ultra_m',
            'billing.plans.mega.stripe_price_monthly' => 'price_mega_m',
        ]);

        $catalog = new SubscriptionCatalog;

        $smartUser = User::factory()->createOne();
        $this->attachCashierSubscription($smartUser, 'price_smart_m', 'active');

        $this->assertFalse($catalog->userMayCreateIdxProxyApiTokens($smartUser));
        $this->assertNull($catalog->idxProxyAbilitiesForUser($smartUser));

        $ultraUser = User::factory()->createOne();
        $this->attachCashierSubscription($ultraUser, 'price_ultra_m', 'active');

        $this->assertTrue($catalog->userMayCreateIdxProxyApiTokens($ultraUser));
        $this->assertSame(['idx:access'], $catalog->idxProxyAbilitiesForUser($ultraUser));

        $megaUser = User::factory()->createOne();
        $this->attachCashierSubscription($megaUser, 'price_mega_m', 'active');

        $this->assertTrue($catalog->userMayCreateIdxProxyApiTokens($megaUser));
        $this->assertSame(['idx:full'], $catalog->idxProxyAbilitiesForUser($megaUser));

        $invalidUltra = User::factory()->createOne();
        $this->attachCashierSubscription($invalidUltra, 'price_ultra_m', 'incomplete_expired');

        $this->assertFalse($catalog->userMayCreateIdxProxyApiTokens($invalidUltra));
    }

    private function attachCashierSubscription(User $user, string $price, string $stripeStatus): void
    {
        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_test_'.$user->id.'_'.$price,
            'stripe_status' => $stripeStatus,
            'stripe_price' => $price,
            'quantity' => 1,
            'trial_ends_at' => null,
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
}
