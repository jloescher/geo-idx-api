<?php

namespace Tests\Feature\Marketing;

use App\Livewire\Marketing\SalesLandingPage;
use Livewire\Livewire;
use Tests\TestCase;

class SalesLandingPricingTest extends TestCase
{
    public function test_sales_livewire_contains_pricing_cta_and_plan_labels(): void
    {
        Livewire::test(SalesLandingPage::class)
            ->assertSee('Quantyra Labs IDX API', false)
            ->assertSee('Add to cart', false)
            ->assertSee('Pro', false)
            ->assertSee('Smart', false)
            ->assertSee('Ultra', false)
            ->assertSee('Mega', false)
            ->assertSee('Annual Subscriptions with 20% Discount', false);
    }

    public function test_billing_interval_toggle_updates_state(): void
    {
        Livewire::test(SalesLandingPage::class)
            ->assertSet('billingInterval', 'monthly')
            ->call('setBillingInterval', 'annual')
            ->assertSet('billingInterval', 'annual');
    }
}
