<?php

namespace App\Http\Controllers;

use App\Services\UserInvitationIssuer;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

class DashboardUserInvitationController extends Controller
{
    public function __construct(
        private readonly UserInvitationIssuer $issuer,
    ) {}

    public function store(Request $request): RedirectResponse
    {
        $validated = $request->validate([
            'email' => ['required', 'string', 'email', 'max:255'],
        ]);

        $this->issuer->issue($validated['email'], $request->user()?->id);

        return back()->with('dashboard_status', __('Invitation sent to :email.', ['email' => $validated['email']]));
    }
}
