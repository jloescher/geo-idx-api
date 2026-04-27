<?php

namespace App\Ghl\OAuth\Controllers;

use App\Ghl\OAuth\Services\OAuthService;
use App\Ghl\OAuth\Support\OAuthStateToken;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

/**
 * Revenue Impact: Starts OAuth consent → marketplace installs → embedded IDX distribution.
 */
class AuthorizeController
{
    public function __invoke(Request $request, OAuthService $oauth): RedirectResponse
    {
        $userType = (string) $request->query('user_type', config('ghl.oauth.default_user_type'));
        $state = OAuthStateToken::encode($userType);
        session([
            'ghl_oauth_state' => $state,
            'ghl_oauth_user_type' => $userType,
        ]);

        return redirect()->away($oauth->buildAuthorizationUrl($state));
    }
}
