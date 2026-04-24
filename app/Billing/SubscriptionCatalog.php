<?php

namespace App\Billing;

use Illuminate\Support\Facades\Cache;

/**
 * Resolves catalog entries for marketing + checkout.
 *
 * Revenue Impact: Cached teaser stats feel “live” without hammering analytics DBs.
 */
final class SubscriptionCatalog
{
    /**
     * @return array<string, array<string, mixed>>
     */
    public function plans(): array
    {
        return config('billing.plans', []);
    }

    public function resolveStripePriceId(string $planKey, string $interval): ?string
    {
        $plan = config("billing.plans.{$planKey}");

        if (! is_array($plan)) {
            return null;
        }

        return match ($interval) {
            'monthly' => $plan['stripe_price_monthly'] ?? null,
            'annual' => $plan['stripe_price_yearly'] ?? null,
            default => null,
        };
    }

    /**
     * Cached pseudo-stat for pricing-card urgency (illustrative range).
     */
    public function teaserLeadsThisMonth(): int
    {
        $ttl = (int) config('billing.teaser_leads_cache_ttl', 900);

        return (int) Cache::remember('idx_sales:teaser_leads_month', $ttl, static function (): int {
            return random_int(1840, 9420);
        });
    }
}
