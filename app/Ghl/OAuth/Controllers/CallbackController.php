<?php

namespace App\Ghl\OAuth\Controllers;

use App\Ghl\OAuth\Services\OAuthService;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

/**
 * Revenue Impact: Completes OAuth → stores encrypted tokens → unlocks MLS widget + CRM sync revenue paths.
 */
class CallbackController
{
    public function __invoke(Request $request, OAuthService $oauth): RedirectResponse
    {
        if ($request->query('error')) {
            return redirect()->route('leadconnector.install')
                ->with('error', (string) $request->query('error_description', $request->query('error')));
        }

        $code = (string) $request->query('code', '');
        $state = (string) $request->query('state', '');
        $sessionState = (string) session('ghl_oauth_state', '');

        if ($code === '' || $state === '' || ! hash_equals($sessionState, $state)) {
            abort(403, 'Invalid OAuth state');
        }

        $userType = (string) session('ghl_oauth_user_type', config('ghl.oauth.default_user_type'));
        session()->forget(['ghl_oauth_state', 'ghl_oauth_user_type']);

        $data = $oauth->exchangeAuthorizationCode($code, $userType);
        $token = $oauth->persistTokenFromResponse($data, $userType);

        session(['ghl_pending_oauth_token_id' => $token->id]);

        return redirect()->route('leadconnector.register-urls');
    }
}
