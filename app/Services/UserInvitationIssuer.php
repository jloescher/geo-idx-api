<?php

namespace App\Services;

use App\Mail\UserInvitationMail;
use App\Models\User;
use App\Models\UserInvitation;
use Illuminate\Support\Facades\Mail;
use Illuminate\Validation\ValidationException;

class UserInvitationIssuer
{
    /**
     * Create a pending invitation and email the acceptance link.
     *
     * @throws ValidationException
     */
    public function issue(string $email, ?int $invitedByUserId): UserInvitation
    {
        $normalized = mb_strtolower(trim($email));

        if ($normalized === '') {
            throw ValidationException::withMessages([
                'email' => __('A valid email address is required.'),
            ]);
        }

        if (User::query()->where('email', $normalized)->exists()) {
            throw ValidationException::withMessages([
                'email' => __('An account with this email already exists.'),
            ]);
        }

        UserInvitation::query()->forEmail($normalized)->whereNull('accepted_at')->delete();

        $created = UserInvitation::createWithPlainToken([
            'email' => $normalized,
            'invited_by' => $invitedByUserId,
            'expires_at' => now()->addHours(max(1, (int) config('invitations.ttl_hours'))),
        ]);

        $url = $this->acceptanceUrl($created['plain']);

        Mail::to($normalized)->send(new UserInvitationMail($url, $created['model']->expires_at));

        return $created['model'];
    }

    private function acceptanceUrl(string $plainToken): string
    {
        $base = rtrim((string) config('idx.platform_url'), '/');

        return $base.'/register/'.$plainToken;
    }
}
