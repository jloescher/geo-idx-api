<?php

namespace App\Http\Controllers;

use App\Services\MlsMembershipVerificationService;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

class DashboardMlsMembershipController extends Controller
{
    public function __construct(
        private readonly MlsMembershipVerificationService $verificationService,
    ) {}

    public function __invoke(Request $request): RedirectResponse
    {
        $user = $request->user();
        if ($user === null) {
            return redirect('/dashboard?panel=onboarding');
        }

        $validated = $request->validate([
            'mls_id' => ['required', 'string', 'min:4', 'max:80'],
            'mls_email' => ['required', 'email', 'max:255'],
        ]);

        $user->forceFill([
            'mls_id' => trim((string) $validated['mls_id']),
            'mls_email' => trim((string) $validated['mls_email']),
            'mls_membership_status' => 'pending',
            'mls_membership_last_error' => null,
        ])->save();

        $verified = $this->verificationService->verify($user, 'stellar');
        if (! $verified) {
            return redirect('/dashboard?panel=onboarding')
                ->withErrors([
                    'mls_membership' => (string) ($user->fresh()?->mls_membership_last_error ?? 'MLS membership verification failed.'),
                ])
                ->withInput();
        }

        return redirect('/dashboard?panel=onboarding')
            ->with('dashboard_status', 'MLS membership verified. You can now continue with domain + billing setup.');
    }
}
