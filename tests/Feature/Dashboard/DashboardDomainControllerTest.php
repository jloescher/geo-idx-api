<?php

namespace Tests\Feature\Dashboard;

use App\Billing\SubscriptionCatalog;
use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Schema;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class DashboardDomainControllerTest extends TestCase
{
    use RefreshDatabase;

    private function attachSubscription(User $user, string $price, int $extraDomainAddonQty = 0): void
    {
        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_domain_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => $price,
            'quantity' => 1,
            'trial_ends_at' => now()->addDays(7),
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_domain_'.$user->id,
            'stripe_product' => 'prod_domain',
            'stripe_price' => $price,
            'quantity' => 1,
        ]);

        if ($extraDomainAddonQty > 0) {
            SubscriptionItem::query()->create([
                'subscription_id' => $subscription->id,
                'stripe_id' => 'si_domain_extra_'.$user->id,
                'stripe_product' => 'prod_domain_extra',
                'stripe_price' => 'price_addon_extra_domain',
                'quantity' => $extraDomainAddonQty,
            ]);
        }
    }

    public function test_authenticated_user_can_add_domain_from_dashboard(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly']);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        $this->post('http://localhost/dashboard/domains', [
            'domain_slug' => 'searchtampabayhouses.com',
        ])->assertRedirect('http://localhost/dashboard');

        $this->assertDatabaseHas('domains', [
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);
    }

    public function test_dashboard_domain_store_normalizes_url_input(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly']);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        $this->post('http://localhost/dashboard/domains', [
            'domain_slug' => 'HTTPS://Example-Idx.com/path?q=1',
        ])->assertRedirect('http://localhost/dashboard');

        $this->assertDatabaseHas('domains', [
            'domain_slug' => 'example-idx.com',
        ]);
    }

    public function test_dashboard_domain_store_gracefully_handles_missing_user_id_column(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly']);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        Schema::shouldReceive('hasColumn')
            ->with('domains', 'user_id')
            ->andReturn(false);

        $this->post('http://localhost/dashboard/domains', [
            'domain_slug' => 'legacy-db.example.com',
        ])->assertRedirect('http://localhost/dashboard');

        $this->assertDatabaseHas('domains', [
            'domain_slug' => 'legacy-db.example.com',
            'is_active' => true,
        ]);
    }

    public function test_dashboard_domain_store_rejects_duplicate_domains(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly']);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'example.com',
            'is_active' => true,
        ]);

        $this->from('http://localhost/dashboard')
            ->post('http://localhost/dashboard/domains', [
                'domain_slug' => 'example.com',
            ])
            ->assertRedirect('http://localhost/dashboard')
            ->assertSessionHasErrors(['domain_slug']);
    }

    public function test_authenticated_user_can_remove_domain_from_dashboard(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly']);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        $domain = Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'remove-me.example.com',
            'is_active' => true,
        ]);

        $this->delete('http://localhost/dashboard/domains/'.$domain->id)
            ->assertRedirect('http://localhost/dashboard');

        $this->assertDatabaseMissing('domains', [
            'id' => $domain->id,
            'domain_slug' => 'remove-me.example.com',
        ]);
    }

    public function test_pro_plan_cannot_add_domain_past_limit(): void
    {
        config([
            'billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly',
            'billing.addons.extra_domain.stripe_price_monthly' => 'price_addon_extra_domain',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly');
        $this->actingAs($user);

        Domain::query()->create(['user_id' => $user->id, 'domain_slug' => 'one.example.com', 'is_active' => true]);

        $this->from('http://localhost/dashboard')
            ->post('http://localhost/dashboard/domains', [
                'domain_slug' => 'two.example.com',
            ])
            ->assertRedirect('http://localhost/dashboard')
            ->assertSessionHasErrors(['domain_slug']);

        $this->assertDatabaseMissing('domains', [
            'domain_slug' => 'two.example.com',
        ]);
    }

    public function test_pro_plan_with_extra_domain_addon_can_add_above_base_limit(): void
    {
        config([
            'billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly',
            'billing.addons.extra_domain.stripe_price_monthly' => 'price_addon_extra_domain',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachSubscription($user, 'price_pro_monthly', 1);
        $this->actingAs($user);
        $this->assertSame(2, app(SubscriptionCatalog::class)->domainLimitForUser($user));

        Domain::query()->create(['user_id' => $user->id, 'domain_slug' => 'one.example.com', 'is_active' => true]);

        $this->post('http://localhost/dashboard/domains', [
            'domain_slug' => 'two.example.com',
        ])->assertRedirect('http://localhost/dashboard');

        $this->assertDatabaseHas('domains', [
            'domain_slug' => 'two.example.com',
            'is_active' => true,
        ]);
    }
}
