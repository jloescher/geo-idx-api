<?php

namespace Tests\Feature\Dashboard;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class DashboardExtraDomainControllerTest extends TestCase
{
    use RefreshDatabase;

    private function attachSubscription(User $user, string $price): void
    {
        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_extra_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => $price,
            'quantity' => 1,
            'trial_ends_at' => now()->addDays(7),
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_extra_'.$user->id,
            'stripe_product' => 'prod_extra',
            'stripe_price' => $price,
            'quantity' => 1,
        ]);
    }

    public function test_smart_plan_cannot_purchase_extra_domain_addon(): void
    {
        config(['billing.plans.smart.stripe_price_monthly' => 'price_smart_monthly']);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_smart_monthly');
        $this->actingAs($user);

        $this->from('http://localhost/dashboard')
            ->post('http://localhost/dashboard/billing/extra-domain', [])
            ->assertRedirect('http://localhost/dashboard')
            ->assertSessionHasErrors(['billing']);
    }

    public function test_pro_without_addon_price_configured_gets_validation_error(): void
    {
        config([
            'billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly',
            'billing.addons.extra_domain.stripe_price_monthly' => '',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        $this->from('http://localhost/dashboard')
            ->post('http://localhost/dashboard/billing/extra-domain', [])
            ->assertRedirect('http://localhost/dashboard')
            ->assertSessionHasErrors(['billing']);
    }

    public function test_pro_with_addon_price_surfaces_stripe_error_in_session(): void
    {
        config([
            'billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly',
            'billing.addons.extra_domain.stripe_price_monthly' => 'price_addon_extra_domain',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        $this->from('http://localhost/dashboard')
            ->post('http://localhost/dashboard/billing/extra-domain', [])
            ->assertRedirect('http://localhost/dashboard')
            ->assertSessionHasErrors(['billing']);
    }
}
