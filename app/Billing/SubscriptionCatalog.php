<?php

namespace App\Billing;

use App\Models\User;
use Illuminate\Support\Facades\Cache;
use Laravel\Cashier\Subscription;

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

    public function planKeyForPriceId(?string $stripePriceId): ?string
    {
        if (! is_string($stripePriceId) || $stripePriceId === '') {
            return null;
        }

        foreach ($this->plans() as $plan) {
            if (! is_array($plan)) {
                continue;
            }
            $monthly = (string) ($plan['stripe_price_monthly'] ?? '');
            $yearly = (string) ($plan['stripe_price_yearly'] ?? '');
            if ($stripePriceId === $monthly || $stripePriceId === $yearly) {
                return (string) ($plan['key'] ?? '');
            }
        }

        return null;
    }

    public function planKeyForUser(User $user): ?string
    {
        /** @var Subscription|null $subscription */
        $subscription = $user->subscription('default');
        if ($subscription === null) {
            return null;
        }

        $subscription->loadMissing('items');
        $priceId = (string) ($subscription->items->first()?->stripe_price ?? $subscription->stripe_price ?? '');

        return $this->planKeyForPriceId($priceId !== '' ? $priceId : null);
    }

    /**
     * Ultra/Mega may create Sanctum tokens that authenticate the Bridge / GIS JSON API (`domain.token`).
     */
    public function userMayCreateIdxProxyApiTokens(User $user): bool
    {
        $subscription = $user->subscription('default');
        if ($subscription === null || ! $subscription->valid()) {
            return false;
        }

        $planKey = $this->planKeyForUser($user);

        return in_array($planKey, ['ultra', 'mega'], true);
    }

    /**
     * @return list<string>|null
     */
    public function idxProxyAbilitiesForUser(User $user): ?array
    {
        if (! $this->userMayCreateIdxProxyApiTokens($user)) {
            return null;
        }

        $planKey = $this->planKeyForUser($user);

        return match ($planKey) {
            'mega' => ['idx:full'],
            'ultra' => ['idx:access'],
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
