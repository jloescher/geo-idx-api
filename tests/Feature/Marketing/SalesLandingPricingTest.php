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
            ->assertSee('GeoIDX by Quantyra Labs', false)
            ->assertSee('Multi-MLS IDX platform', false)
            ->assertSee('Simple, Transparent Pricing', false)
            ->assertSee('20% Discount', false)
            ->assertSee('Pro &amp; Smart plans include completely unlimited JS Widget usage. No monthly API call limits.', false)
            ->assertSee('Overage billing (Ultra &amp; Mega only): $0.001 per extra API call', false)
            ->assertSee('Add to cart', false)
            ->assertSee('Pro', false)
            ->assertSee('Smart', false)
            ->assertSee('Ultra', false)
            ->assertSee('Mega', false)
            ->assertDontSee('10,000 API calls / month included', false)
            ->assertDontSee('50,000 API calls / month included', false);
    }

    public function test_billing_interval_toggle_updates_state(): void
    {
        Livewire::test(SalesLandingPage::class)
            ->assertSet('billingInterval', 'monthly')
            ->call('setBillingInterval', 'annual')
            ->assertSet('billingInterval', 'annual');
    }
}
