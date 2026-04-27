<?php

namespace App\Http\Controllers;

use App\Billing\SubscriptionCatalog;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

class DashboardLeadsController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
    ) {}

    public function __invoke(Request $request): RedirectResponse
    {
        $user = $request->user();
        if ($user === null) {
            abort(403);
        }

        $planKey = $this->catalog->planKeyForUser($user) ?? '';
        $eligible = in_array($planKey, ['pro', 'smart', 'ultra', 'mega'], true)
            && (string) $user->mls_membership_status === 'active'
            && $user->subscription('default')?->valid() === true;

        if (! $eligible) {
            return redirect('/dashboard')
                ->withErrors([
                    'dashboard' => 'Leads are available after an active paid subscription and verified MLS membership.',
                ]);
        }

        return redirect('/dashboard?panel=leads');
    }
}
