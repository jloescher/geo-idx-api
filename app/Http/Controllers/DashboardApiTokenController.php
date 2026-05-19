<?php

namespace App\Http\Controllers;

use App\Models\User;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Laravel\Sanctum\PersonalAccessToken;

class DashboardApiTokenController extends Controller
{
    public function store(Request $request): RedirectResponse
    {
        $validated = $request->validate([
            'token_name' => ['required', 'string', 'max:60'],
        ]);

        /** @var User $user */
        $user = $request->user();
        $token = $user->createToken($validated['token_name'], ['idx:full']);

        return redirect('/dashboard')
            ->with('dashboard_new_api_token', $token->plainTextToken);
    }

    public function storeStaging(Request $request): RedirectResponse
    {
        /** @var User $user */
        $user = $request->user();

        $alreadyExists = $user->tokens()
            ->whereRaw('LOWER(name) = ?', ['staging'])
            ->exists();

        if ($alreadyExists) {
            return redirect('/dashboard')
                ->with('dashboard_status', 'A Staging key already exists. Revoke it first if you need a new one.');
        }

        $token = $user->createToken('Staging', ['idx:full']);

        return redirect('/dashboard')
            ->with('dashboard_new_api_token', $token->plainTextToken)
            ->with('dashboard_status', 'Staging API key generated. Copy it now — it will not be shown again.');
    }

    public function destroy(Request $request, PersonalAccessToken $token): RedirectResponse
    {
        /** @var User $user */
        $user = $request->user();
        abort_unless($token->tokenable_id === $user->id && $token->tokenable_type === $user::class, 403);

        $token->delete();

        return redirect('/dashboard')
            ->with('dashboard_status', 'API token revoked.');
    }
}
