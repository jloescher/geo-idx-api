<?php

namespace App\Http\Controllers\Billing;

use App\Billing\SubscriptionCatalog;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Http\Controllers\Controller;
use App\Models\Domain;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Laravel\Cashier\Checkout;
use Laravel\Cashier\Exceptions\IncompletePayment;

/**
 * Stripe Checkout entry for IDX subscriptions (Cashier).
 *
 * Revenue Impact: One-click path from pricing → authenticated checkout maximizes paid conversion while preserving webhooks + local subscription rows.
 */
class SubscriptionCheckoutController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
    ) {}

    public function __invoke(Request $request): RedirectResponse|Checkout
    {
        $validated = $request->validate([
            'plan' => ['required', 'string', 'in:pro,smart,ultra,mega'],
            'interval' => ['required', 'string', 'in:monthly,annual'],
        ]);

        $priceId = $this->catalog->resolveStripePriceId($validated['plan'], $validated['interval']);

        if (! is_string($priceId) || $priceId === '') {
            return redirect()
                ->to('/#pricing')
                ->with('flash_billing_error', 'Billing is not configured for this plan yet. Add Stripe Price IDs to the server environment.');
        }

        $user = $request->user();
        if ((string) $user->mls_membership_status !== 'active') {
            return redirect()
                ->to('/dashboard')
                ->withErrors([
                    'billing' => 'Complete MLS verification (MLS ID + email) before starting your paid subscription.',
                ]);
        }

        $hasVerifiedDomain = Domain::query()
            ->where('user_id', $user->id)
            ->where('is_active', true)
            ->whereIn('verification_status', ['verified', 'verified_ghl'])
            ->exists();
        $hasGhlProof = GhlRegisteredUrl::query()
            ->where('quantyra_user_id', $user->id)
            ->where('widget_access_enabled', true)
            ->exists();
        if (! $hasVerifiedDomain && ! $hasGhlProof) {
            return redirect()
                ->to('/dashboard')
                ->withErrors([
                    'billing' => 'Verify a domain by DNS TXT record or connect LeadConnector/GHL before checkout.',
                ]);
        }

        $origin = $request->getSchemeAndHttpHost();
        $successUrl = $origin.route('marketing.sales', [], false).'?checkout=success&session_id={CHECKOUT_SESSION_ID}';
        $cancelUrl = $origin.route('marketing.sales', [], false).'?checkout=cancelled#pricing';

        try {
            return $user
                ->newSubscription('default', $priceId)
                ->allowPromotionCodes()
                ->checkout([
                    'success_url' => $successUrl,
                    'cancel_url' => $cancelUrl,
                ]);
        } catch (IncompletePayment $e) {
            return redirect()->route('cashier.payment', [
                'id' => $e->payment->id,
                'redirect' => route('marketing.sales', [], false),
            ]);
        }
    }
}
