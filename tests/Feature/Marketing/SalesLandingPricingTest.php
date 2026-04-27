<?php

namespace Tests\Feature\Marketing;

use Livewire\Livewire;
use Tests\TestCase;

class SalesLandingPricingTest extends TestCase
{
    public function test_sales_livewire_contains_pricing_cta_and_plan_labels(): void
    {
        Livewire::test('marketing.sales-landing-page')
            ->assertSee('GeoIDX by Quantyra Labs', false)
            ->assertSee('Geography-first IDX widgets that convert faster.', false)
            ->assertSee('Simple, Transparent Pricing', false)
            ->assertSee('Annual subscriptions include', false)
            ->assertSee('14-day trial includes demo data only', false)
            ->assertSee('Overage billing (Ultra &amp; Mega only): $0.001 per extra API call', false)
            ->assertSee('Start Demo Trial', false)
            ->assertSee('Activate Live Data', false)
            ->assertSee('Pro', false)
            ->assertSee('Smart', false)
            ->assertSee('Ultra', false)
            ->assertSee('Mega', false)
            ->assertDontSee('10,000 API calls / month included', false)
            ->assertDontSee('50,000 API calls / month included', false);
    }

    public function test_billing_interval_toggle_updates_state(): void
    {
        Livewire::test('marketing.sales-landing-page')
            ->assertSet('billingInterval', 'monthly')
            ->call('setBillingInterval', 'annual')
            ->assertSet('billingInterval', 'annual');
    }
}
