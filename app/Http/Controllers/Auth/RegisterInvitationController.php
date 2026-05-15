<?php

namespace App\Http\Controllers\Auth;

use App\Actions\Fortify\CreateNewUser;
use App\Models\UserInvitation;
use Illuminate\Auth\Events\Registered;
use Illuminate\Contracts\Auth\StatefulGuard;
use Illuminate\Contracts\View\View;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Controller;
use Illuminate\Support\Str;
use Laravel\Fortify\Contracts\RegisterResponse;
use Laravel\Fortify\Fortify;

class RegisterInvitationController extends Controller
{
    public function __construct(
        private readonly StatefulGuard $guard,
    ) {}

    public function show(string $token): RedirectResponse|View
    {
        $invitation = UserInvitation::findValidByPlainToken($token);

        if ($invitation === null) {
            return redirect()
                ->route('login')
                ->with('status', __('This invitation link is invalid or has expired. Ask your administrator for a new invite.'));
        }

        return view('auth.register-invite', [
            'invitation' => $invitation,
            'invitationToken' => $token,
        ]);
    }

    public function store(Request $request, CreateNewUser $creator): RegisterResponse
    {
        if (config('fortify.lowercase_usernames') && $request->has(Fortify::username())) {
            $request->merge([
                Fortify::username() => Str::lower($request->{Fortify::username()}),
            ]);
        }

        event(new Registered($user = $creator->create($request->all())));

        $this->guard->login($user, $request->boolean('remember'));

        if ($request->hasSession()) {
            $request->session()->regenerate();
        }

        return app(RegisterResponse::class);
    }
}
