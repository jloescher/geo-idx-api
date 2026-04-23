<?php

namespace App\Ghl\Sync\Services;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Sync\Models\GhlInstalledLocation;

/**
 * Revenue Impact: Stripe subscription state mirrored into GHL tags → paywall clarity → upgrades.
 */
class SubscriptionSyncService
{
    /**
     * Map Stripe-style status to installed_locations.subscription_status (external billing).
     */
    public function applyStripeStatus(string $ghlLocationId, string $stripeStatus, ?string $subscriptionId = null): void
    {
        GhlInstalledLocation::query()
            ->where('ghl_location_id', $ghlLocationId)
            ->update([
                'subscription_status' => match ($stripeStatus) {
                    'active' => 'active',
                    'trialing' => 'trial',
                    'past_due' => 'past_due',
                    'canceled', 'unpaid' => 'cancelled',
                    default => 'none',
                },
                'subscription_id' => $subscriptionId,
                'subscription_updated_at' => now(),
            ]);

        // Tag sync to GHL CRM is queued via SyncSubscriptionStatusJob when token exists.
    }

    public function resolveTokenForLocation(string $ghlLocationId): ?GhlOAuthToken
    {
        return GhlOAuthToken::query()
            ->where('ghl_location_id', $ghlLocationId)
            ->where('status', 'active')
            ->orderByDesc('id')
            ->first();
    }
}
