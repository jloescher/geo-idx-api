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

        /** @var list<string> $priceIds */
        $priceIds = [];
        $appendPriceId = static function (string $rawId) use (&$priceIds): void {
            $id = trim($rawId);
            if ($id !== '' && ! in_array($id, $priceIds, true)) {
                $priceIds[] = $id;
            }
        };

        $appendPriceId((string) ($subscription->stripe_price ?? ''));
        foreach ($subscription->items as $item) {
            $appendPriceId((string) ($item->stripe_price ?? ''));
        }

        foreach ($priceIds as $priceId) {
            $key = $this->planKeyForPriceId($priceId);
            if (is_string($key) && $key !== '') {
                return $key;
            }
        }

        return null;
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

    public function domainLimitForPlan(?string $planKey): ?int
    {
        return match ($planKey) {
            'pro' => 1,
            'smart' => 5,
            'ultra', 'mega' => null,
            default => 0,
        };
    }

    public function domainLimitForUser(User $user): ?int
    {
        /** @var Subscription|null $subscription */
        $subscription = $user->subscription('default');
        if ($subscription === null || ! $subscription->valid()) {
            return 0;
        }

        $planKey = $this->planKeyForUser($user);
        $baseLimit = $this->domainLimitForPlan($planKey);
        if ($baseLimit === null) {
            return null;
        }

        if ($planKey !== 'pro') {
            return $baseLimit;
        }

        $subscription->loadMissing('items');
        $extraDomainPriceId = (string) config('billing.addons.extra_domain.stripe_price_monthly', '');
        if ($extraDomainPriceId === '') {
            return $baseLimit;
        }

        $extraDomainQty = (int) ($subscription->items
            ->where('stripe_price', $extraDomainPriceId)
            ->sum('quantity'));

        return $baseLimit + max(0, $extraDomainQty);
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
