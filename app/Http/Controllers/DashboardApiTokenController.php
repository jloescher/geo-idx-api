<?php

namespace App\Http\Controllers;

use App\Billing\SubscriptionCatalog;
use App\Models\User;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Laravel\Sanctum\PersonalAccessToken;

class DashboardApiTokenController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
    ) {}

    /**
     * Revenue Impact: Self-serve API token lifecycle reduces support friction
     * for Ultra/Mega teams and speeds time-to-first integration.
     */
    public function store(Request $request): RedirectResponse
    {
        $validated = $request->validate([
            'token_name' => ['required', 'string', 'max:60'],
        ]);

        /** @var User $user */
        $user = $request->user();
        abort_unless($this->catalog->userMayCreateIdxProxyApiTokens($user), 403);

        $abilities = $this->catalog->idxProxyAbilitiesForUser($user);
        abort_if($abilities === null, 403);

        $token = $user->createToken($validated['token_name'], $abilities);

        return redirect('/dashboard')
            ->with('dashboard_new_api_token', $token->plainTextToken);
    }

    public function destroy(Request $request, PersonalAccessToken $token): RedirectResponse
    {
        /** @var User $user */
        $user = $request->user();
        abort_unless($this->catalog->userMayCreateIdxProxyApiTokens($user), 403);

        abort_unless($token->tokenable_id === $user->id && $token->tokenable_type === $user::class, 403);

        $token->delete();

        return redirect('/dashboard')
            ->with('dashboard_status', 'API token revoked.');
    }
}
