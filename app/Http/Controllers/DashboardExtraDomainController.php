<?php

namespace App\Http\Controllers;

use App\Billing\SubscriptionCatalog;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Laravel\Cashier\Exceptions\IncompletePayment;
use Stripe\Exception\ApiErrorException;

/**
 * Adds the Stripe "extra domain" recurring price to the subscriber's active subscription (Pro).
 */
class DashboardExtraDomainController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
    ) {}

    public function store(Request $request): RedirectResponse
    {
        $user = $request->user();
        if ($user === null) {
            return redirect('/dashboard');
        }

        $planKey = $this->catalog->planKeyForUser($user);
        if ($planKey !== 'pro') {
            return redirect('/dashboard')
                ->withErrors(['billing' => 'Extra domain add-ons are available on the Pro plan.']);
        }

        $subscription = $user->subscription('default');
        if ($subscription === null || ! $subscription->valid()) {
            return redirect('/dashboard')
                ->withErrors(['billing' => 'You need an active subscription before purchasing add-ons.']);
        }

        $priceId = (string) config('billing.addons.extra_domain.stripe_price_monthly', '');
        if ($priceId === '') {
            return redirect('/dashboard')
                ->withErrors(['billing' => 'Extra domain billing is not configured yet. Contact support.']);
        }

        try {
            $subscription->loadMissing('items');
            $item = $subscription->items->firstWhere('stripe_price', $priceId);
            if ($item !== null) {
                $subscription->incrementQuantity(1, $priceId);
            } else {
                $subscription->addPriceAndInvoice($priceId);
            }
        } catch (IncompletePayment $e) {
            return redirect()->route('cashier.payment', [
                'id' => $e->payment->id,
                'redirect' => '/dashboard',
            ]);
        } catch (ApiErrorException $e) {
            return redirect('/dashboard')
                ->withErrors(['billing' => 'Stripe could not add the add-on: '.$e->getMessage()]);
        }

        return redirect('/dashboard')
            ->with('dashboard_status', 'Extra domain slot added to your subscription. You can register another approved hostname below.');
    }
}
