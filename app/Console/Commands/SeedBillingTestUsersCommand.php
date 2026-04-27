<?php

namespace App\Console\Commands;

use App\Billing\SubscriptionCatalog;
use App\Models\User;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Hash;
use Laravel\Cashier\Exceptions\IncompletePayment;

/**
 * Provisions four Billable users against Stripe test mode using monthly catalog prices.
 */
class SeedBillingTestUsersCommand extends Command
{
    protected $signature = 'billing:seed-test-users
                            {--password= : Shared login password (falls back to BILLING_SEED_TEST_USERS_PASSWORD, then "password")}
                            {--payment-method=pm_card_visa : Stripe test PaymentMethod id used when starting paid subscriptions}
                            {--skip-subscription : Create users only; do not call Stripe (use when prices/products are inactive or catalog is being rebuilt)}';

    protected $description = 'Create idx-seed-{plan}@quantyralabs.test users with active Stripe (test) subscriptions for Pro, Smart, Ultra, and Mega';

    public function handle(SubscriptionCatalog $catalog): int
    {
        $skipSubscription = (bool) $this->option('skip-subscription');

        if (! $skipSubscription && ! filled((string) config('cashier.secret'))) {
            $this->error('Stripe is not configured (set STRIPE_SECRET for Cashier).');

            return self::FAILURE;
        }

        $passwordPlain = (string) ($this->option('password') ?: env('BILLING_SEED_TEST_USERS_PASSWORD', 'password'));
        $paymentMethod = (string) $this->option('payment-method');

        /** @var list<string> $planKeys */
        $planKeys = ['pro', 'smart', 'ultra', 'mega'];
        /** @var array<string, string> $priceByPlan */
        $priceByPlan = [];

        if (! $skipSubscription) {
            foreach ($planKeys as $planKey) {
                $priceId = $catalog->resolveStripePriceId($planKey, 'monthly');
                if (! is_string($priceId) || $priceId === '') {
                    $this->error("Missing monthly Stripe price for \"{$planKey}\". Set STRIPE_PRICE_IDX_".mb_strtoupper($planKey).'_MONTHLY in .env.');

                    return self::FAILURE;
                }
                $priceByPlan[$planKey] = $priceId;
            }
        }

        $rows = [];

        foreach ($planKeys as $planKey) {
            $email = "idx-seed-{$planKey}@quantyralabs.test";
            $user = User::query()->where('email', $email)->first();

            if ($user === null) {
                $user = User::query()->create([
                    'name' => 'IDX Seed '.ucfirst($planKey),
                    'email' => $email,
                    'password' => Hash::make($passwordPlain),
                    'email_verified_at' => now(),
                ]);
            } else {
                $user->forceFill([
                    'name' => 'IDX Seed '.ucfirst($planKey),
                    'password' => Hash::make($passwordPlain),
                ])->save();
            }

            if ($skipSubscription) {
                $this->components->warn("Users-only {$planKey}: {$email} (no Stripe subscription).");
                $rows[] = [$email, $planKey, 'users_only'];

                continue;
            }

            if ($user->subscribed('default')) {
                $this->components->info("Skipped {$planKey}: {$email} already has a valid default subscription.");
                $rows[] = [$email, $planKey, 'unchanged (already subscribed)'];

                continue;
            }

            try {
                if (! $user->hasStripeId()) {
                    $user->createAsStripeCustomer([
                        'email' => $user->email,
                        'name' => $user->name,
                    ]);
                }

                $user->newSubscription('default', $priceByPlan[$planKey])
                    ->skipTrial()
                    ->create($paymentMethod);
            } catch (IncompletePayment $e) {
                $this->error("Incomplete payment for {$email}: {$e->getMessage()}");

                return self::FAILURE;
            } catch (\Throwable $e) {
                $message = $e->getMessage();
                $this->error("Failed provisioning {$planKey} ({$email}): {$message}");
                if (str_contains(mb_strtolower($message), 'inactive')) {
                    $this->newLine();
                    $this->line('Hint: In Stripe Dashboard, activate the Product for this price, or run `php artisan billing:provision-stripe-catalog` and update STRIPE_PRICE_IDX_* in .env.');
                    $this->line('Alternatively, create logins only: `php artisan billing:seed-test-users --skip-subscription`');
                }

                return self::FAILURE;
            }

            $this->components->info("Subscribed {$planKey}: {$email}");
            $rows[] = [$email, $planKey, 'created'];
        }

        $this->newLine();
        $this->table(['Email', 'Plan', 'Status'], $rows);
        $this->newLine();
        $this->line('Shared password: '.$passwordPlain);
        if ($skipSubscription) {
            $this->warn('No Stripe subscriptions were created. Dashboard plan/API gates depend on Cashier; fix Stripe catalog then re-run without --skip-subscription.');
        } else {
            $this->line('Log in at your APP_URL dashboard, then use Ultra/Mega accounts to generate Sanctum tokens for /api/v1.');
        }

        return self::SUCCESS;
    }
}
