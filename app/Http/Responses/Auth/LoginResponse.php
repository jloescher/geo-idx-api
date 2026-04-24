<?php

namespace App\Http\Responses\Auth;

use Illuminate\Http\RedirectResponse;
use Laravel\Fortify\Contracts\LoginResponse as LoginResponseContract;

class LoginResponse implements LoginResponseContract
{
    public function toResponse($request): RedirectResponse
    {
        $user = $request->user();

        if ($user?->subscribed('default')) {
            return redirect()->intended('/dashboard');
        }

        return redirect()->intended('/#pricing');
    }
}
