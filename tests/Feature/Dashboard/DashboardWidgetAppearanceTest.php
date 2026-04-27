<?php

namespace Tests\Feature\Dashboard;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Schema;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class DashboardWidgetAppearanceTest extends TestCase
{
    use RefreshDatabase;

    private function attachProSubscription(User $user): void
    {
        $priceId = 'price_test_pro_monthly';
        config(['billing.plans.pro.stripe_price_monthly' => $priceId]);

        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_widget_appearance_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => $priceId,
            'quantity' => 1,
            'trial_ends_at' => now()->addDays(7),
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_widget_appearance_'.$user->id,
            'stripe_product' => 'prod_test',
            'stripe_price' => $priceId,
            'quantity' => 1,
        ]);
    }

    public function test_subscriber_can_save_widget_palette(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachProSubscription($user);

        $this->actingAs($user)->post('http://localhost/dashboard/widget-appearance', [
            'primary' => '#aabbcc',
            'secondary' => '#112233',
            'accent' => '#ff00ff',
            'text' => '#010203',
            'background' => '#fefefe',
            'theme' => 'dark',
        ])->assertRedirect('/dashboard');

        $user->refresh();
        $this->assertSame('#aabbcc', $user->widget_palette['primary'] ?? null);
        $this->assertSame('#112233', $user->widget_palette['secondary'] ?? null);
        $this->assertSame('dark', $user->widget_palette['theme'] ?? null);
    }

    public function test_returns_dashboard_error_when_widget_palette_column_is_missing(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachProSubscription($user);

        Schema::shouldReceive('hasColumn')
            ->once()
            ->with('users', 'widget_palette')
            ->andReturn(false);

        $this->actingAs($user)->post('http://localhost/dashboard/widget-appearance', [
            'primary' => '#aabbcc',
            'secondary' => '#112233',
            'accent' => '#ff00ff',
            'text' => '#010203',
            'background' => '#fefefe',
            'theme' => 'dark',
        ])->assertRedirect('/dashboard')
            ->assertSessionHasErrors(['widget_palette']);

        $user->refresh();
        $this->assertNull($user->widget_palette);
    }

    public function test_invalid_palette_payload_redirects_to_dashboard_not_post_only_route(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->attachProSubscription($user);

        $this->actingAs($user)->post('http://localhost/dashboard/widget-appearance', [
            'primary' => 'not-a-hex',
            'secondary' => '#112233',
            'text' => '#010203',
            'background' => '#fefefe',
            'theme' => 'light',
        ])->assertRedirect('/dashboard')
            ->assertSessionHasErrors(['primary']);
    }
}
