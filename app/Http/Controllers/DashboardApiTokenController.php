<?php

namespace App\Http\Controllers;

use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
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
        $token = $user->createToken($validated['token_name'], ['idx:read', 'idx:search']);

        return redirect()
            ->route('dashboard.index', [], false)
            ->with('dashboard_new_api_token', $token->plainTextToken);
    }

    public function destroy(Request $request, PersonalAccessToken $token): RedirectResponse
    {
        $user = $request->user();

        abort_unless($token->tokenable_id === $user->id && $token->tokenable_type === $user::class, 403);

        $token->delete();

        return redirect()
            ->route('dashboard.index', [], false)
            ->with('dashboard_status', 'API token revoked.');
    }
}
