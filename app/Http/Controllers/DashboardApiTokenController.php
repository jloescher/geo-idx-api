<?php

namespace App\Http\Controllers;

use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Collection;
use Laravel\Sanctum\PersonalAccessToken;

class DashboardApiTokenController extends Controller
{
    /**
     * Revenue Impact: Self-serve API token lifecycle reduces support friction
     * for Ultra/Mega teams and speeds time-to-first integration.
     */
    public function store(Request $request): RedirectResponse
    {
        $validated = $request->validate([
            'token_name' => ['required', 'string', 'max:60'],
        ]);

        $user = $request->user();
        abort_unless($this->hasApiAccess($user), 403);

        $token = $user->createToken($validated['token_name'], ['idx:read', 'idx:search']);

        return redirect()
            ->route('dashboard.index', [], false)
            ->with('dashboard_new_api_token', $token->plainTextToken);
    }

    public function destroy(Request $request, PersonalAccessToken $token): RedirectResponse
    {
        $user = $request->user();
        abort_unless($this->hasApiAccess($user), 403);

        abort_unless($token->tokenable_id === $user->id && $token->tokenable_type === $user::class, 403);

        $token->delete();

        return redirect()
            ->route('dashboard.index', [], false)
            ->with('dashboard_status', 'API token revoked.');
    }

    private function hasApiAccess(object $user): bool
    {
        $subscription = $user->subscription('default');

        if (! $subscription) {
            return false;
        }

        $subscription->loadMissing('items');
        $activePrice = (string) ($subscription->items->first()->stripe_price ?? $subscription->stripe_price ?? '');

        if ($activePrice === '') {
            return false;
        }

        /** @var Collection<int, array<string, mixed>> $plans */
        $plans = collect(config('billing.plans', []));
        $apiPrices = $plans
            ->only(['ultra', 'mega'])
            ->flatMap(static fn (array $plan): array => [
                (string) ($plan['stripe_price_monthly'] ?? ''),
                (string) ($plan['stripe_price_yearly'] ?? ''),
            ])
            ->filter(static fn (string $price): bool => $price !== '');

        return $apiPrices->contains($activePrice);
    }
}
