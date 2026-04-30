<?php

namespace Tests\Feature\Widgets;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class DirectSiteWidgetEmbedTest extends TestCase
{
    use RefreshDatabase;

    private function subscribePro(User $user): void
    {
        $priceId = 'price_pro_monthly';
        config(['billing.plans.pro.stripe_price_monthly' => $priceId]);

        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_test_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => $priceId,
            'quantity' => 1,
            'trial_ends_at' => now()->addDays(7),
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_test_'.$user->id,
            'stripe_product' => 'prod_test',
            'stripe_price' => $priceId,
            'quantity' => 100,
        ]);
    }

    public function test_dashboard_widget_validate_accepts_subscriber_embed_site_key(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $key = $user->ensureWidgetEmbedSiteKey();

        $this->actingAs($user);

        $this->postJson('http://localhost/dashboard/widget-validate', [
            'token' => $key,
            'hostname' => 'localhost',
            'requireFooter' => true,
        ])->assertOk()
            ->assertJsonPath('ok', true)
            ->assertJsonPath('locationId', 'direct-'.$user->id);
    }

    public function test_widget_config_accepts_direct_site_key_with_approved_domain_origin(): void
    {
        $user = User::factory()->createOne(['mls_membership_status' => 'active']);
        $this->subscribePro($user);
        $key = $user->ensureWidgetEmbedSiteKey();
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'embed-client.test',
            'is_active' => true,
            'verification_status' => 'verified',
        ]);

        $this->withHeaders([
            'Origin' => 'https://embed-client.test',
        ])->get('http://localhost/widget/config/'.$key)
            ->assertOk()
            ->assertJsonPath('location_id', 'direct-'.$user->id);
    }
}
