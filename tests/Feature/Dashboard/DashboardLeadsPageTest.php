<?php

namespace Tests\Feature\Dashboard;

use App\Ghl\Sync\Models\QuantyraLead;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class DashboardLeadsPageTest extends TestCase
{
    use RefreshDatabase;

    public function test_leads_page_requires_active_subscription_and_mls_verification(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly']);
        $user = User::factory()->createOne([
            'mls_id' => 'MLS-55',
            'mls_email' => 'user@example.test',
            'assigned_mls_datasets' => ['stellar'],
            'mls_membership_status' => 'active',
            'mls_membership_verified_at' => now(),
        ]);
        $this->attachSubscription($user, 'price_pro_monthly');

        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'registration',
            'source' => 'widget',
            'payload' => ['name' => 'Jane Doe', 'status' => 'new'],
            'listing_id' => 'stellar:123',
            'quantyra_domain' => 'idx.example.com',
        ]);

        $this->actingAs($user)
            ->get('http://localhost/dashboard/leads')
            ->assertRedirect('http://localhost/dashboard?panel=leads');

        $this->actingAs($user)
            ->get('http://localhost/dashboard?panel=leads')
            ->assertOk()
            ->assertSee('Leads')
            ->assertSee('Jane Doe');
    }

    private function attachSubscription(User $user, string $price): void
    {
        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_leads_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => $price,
            'quantity' => 1,
            'trial_ends_at' => null,
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_leads_'.$user->id,
            'stripe_product' => 'prod_leads',
            'stripe_price' => $price,
            'quantity' => 1,
        ]);
    }
}
