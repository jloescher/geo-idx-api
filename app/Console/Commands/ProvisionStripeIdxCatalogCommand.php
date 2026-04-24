<?php

namespace App\Console\Commands;

use Illuminate\Console\Command;
use Laravel\Cashier\Cashier;
use Stripe\Exception\ApiErrorException;

/**
 * Creates Stripe Products + recurring Prices for IDX tiers (run once per environment).
 *
 * Revenue Impact: Repeatable catalog provisioning prevents “wrong price ID in prod” incidents that block checkout.
 */
class ProvisionStripeIdxCatalogCommand extends Command
{
    protected $signature = 'billing:provision-stripe-catalog';

    protected $description = 'Create Quantyra IDX Stripe products/prices and print STRIPE_PRICE_* env lines';

    /**
     * @var array<string, array{label: string, monthly_cents: int, yearly_cents: int}>
     */
    private array $tierDefinitions = [
        'IDX_PRO' => ['label' => 'Quantyra Labs IDX — Pro', 'monthly_cents' => 3900, 'yearly_cents' => 37400],
        'IDX_SMART' => ['label' => 'Quantyra Labs IDX — Smart', 'monthly_cents' => 7900, 'yearly_cents' => 75800],
        'IDX_ULTRA' => ['label' => 'Quantyra Labs IDX — Ultra', 'monthly_cents' => 17900, 'yearly_cents' => 171800],
        'IDX_MEGA' => ['label' => 'Quantyra Labs IDX — Mega', 'monthly_cents' => 44900, 'yearly_cents' => 431000],
    ];

    public function handle(): int
    {
        $secret = config('cashier.secret');

        if (! is_string($secret) || $secret === '') {
            $this->error('Missing STRIPE_SECRET / cashier.secret. Set .env then re-run.');

            return self::FAILURE;
        }

        $stripe = Cashier::stripe();
        $currency = config('cashier.currency', 'usd');

        $lines = [];

        try {
            foreach ($this->tierDefinitions as $envPrefix => $def) {
                $product = $stripe->products->create([
                    'name' => $def['label'],
                    'metadata' => ['quantyra_catalog' => 'idx_api', 'tier' => strtolower(str_replace('IDX_', '', $envPrefix))],
                ]);

                $monthly = $stripe->prices->create([
                    'product' => $product->id,
                    'currency' => $currency,
                    'unit_amount' => $def['monthly_cents'],
                    'recurring' => ['interval' => 'month'],
                    'metadata' => ['billing_interval' => 'monthly'],
                ]);

                $yearly = $stripe->prices->create([
                    'product' => $product->id,
                    'currency' => $currency,
                    'unit_amount' => $def['yearly_cents'],
                    'recurring' => ['interval' => 'year'],
                    'metadata' => ['billing_interval' => 'annual', 'discount_note' => '20_percent_off_monthly_x12'],
                ]);

                $lines[] = match ($envPrefix) {
                    'IDX_PRO' => [
                        "STRIPE_PRICE_IDX_PRO_MONTHLY={$monthly->id}",
                        "STRIPE_PRICE_IDX_PRO_YEARLY={$yearly->id}",
                    ],
                    'IDX_SMART' => [
                        "STRIPE_PRICE_IDX_SMART_MONTHLY={$monthly->id}",
                        "STRIPE_PRICE_IDX_SMART_YEARLY={$yearly->id}",
                    ],
                    'IDX_ULTRA' => [
                        "STRIPE_PRICE_IDX_ULTRA_MONTHLY={$monthly->id}",
                        "STRIPE_PRICE_IDX_ULTRA_YEARLY={$yearly->id}",
                    ],
                    'IDX_MEGA' => [
                        "STRIPE_PRICE_IDX_MEGA_MONTHLY={$monthly->id}",
                        "STRIPE_PRICE_IDX_MEGA_YEARLY={$yearly->id}",
                    ],
                    default => [],
                };

                $this->info("Provisioned {$def['label']}: monthly {$monthly->id}, yearly {$yearly->id}");
            }
        } catch (ApiErrorException $e) {
            $this->error('Stripe API error: '.$e->getMessage());

            return self::FAILURE;
        }

        $this->newLine();
        $this->comment('Append to .env:');
        foreach ($lines as $pair) {
            foreach ($pair as $line) {
                $this->line($line);
            }
        }

        $this->newLine();
        $this->warn('Optional metered overage + add-on prices: create manually in Stripe or extend this command.');

        return self::SUCCESS;
    }
}
