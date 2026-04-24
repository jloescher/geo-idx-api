<?php

namespace App\Http\Controllers\Billing;

use App\Billing\SubscriptionCatalog;
use App\Http\Controllers\Controller;
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
                ->route('marketing.sales')
                ->withFragment('pricing')
                ->with('flash_billing_error', 'Billing is not configured for this plan yet. Add Stripe Price IDs to the server environment.');
        }

        $user = $request->user();
        $trialDays = (int) config('billing.trial_days', 14);

        $successUrl = route('marketing.sales', [], true).'?checkout=success&session_id={CHECKOUT_SESSION_ID}';
        $cancelUrl = route('marketing.sales', [], true).'?checkout=cancelled#pricing';

        try {
            return $user
                ->newSubscription('default', $priceId)
                ->trialDays($trialDays)
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
