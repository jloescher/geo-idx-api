<?php

namespace App\Actions\Fortify;

use App\Models\Domain;
use App\Models\User;
use App\Models\UserInvitation;
use App\Services\DomainOwnershipVerifier;
use App\Services\MlsMembershipVerificationService;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Validator;
use Illuminate\Validation\Rule;
use Illuminate\Validation\ValidationException;
use Laravel\Fortify\Contracts\CreatesNewUsers;

class CreateNewUser implements CreatesNewUsers
{
    use PasswordValidationRules;

    /**
     * Validate and create a newly registered user.
     *
     * @param  array<string, mixed>  $input
     *
     * @throws ValidationException
     */
    public function create(array $input): User
    {
        $invitation = UserInvitation::findValidByPlainToken((string) ($input['invitation_token'] ?? ''));

        if ($invitation === null) {
            throw ValidationException::withMessages([
                'invitation_token' => __('This invitation link is invalid or has expired.'),
            ]);
        }

        if (User::query()->where('email', $invitation->email)->exists()) {
            throw ValidationException::withMessages([
                'email' => __('An account already exists for this email address.'),
            ]);
        }

        $input['email'] = $invitation->email;

        Validator::make($input, [
            'name' => ['required', 'string', 'max:255'],
            'email' => [
                'required',
                'string',
                'email',
                'max:255',
                Rule::unique(User::class),
            ],
            'invitation_token' => ['required', 'string'],
            'domain_slug' => [
                'required',
                'string',
                'max:255',
                'regex:/^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$/',
                Rule::unique('domains', 'domain_slug'),
            ],
            'dataset' => ['required', 'string', 'in:stellar'],
            'mls_id' => ['required', 'string', 'min:4', 'max:80'],
            'mls_email' => ['required', 'email', 'max:255'],
            'password' => $this->passwordRules(),
        ])->validate();

        /** @var User $user */
        $user = DB::transaction(function () use ($input, $invitation): User {
            $lockedInvitation = UserInvitation::query()
                ->whereKey($invitation->id)
                ->lockForUpdate()
                ->first();

            if ($lockedInvitation === null
                || $lockedInvitation->accepted_at !== null
                || $lockedInvitation->expires_at->isPast()) {
                throw ValidationException::withMessages([
                    'invitation_token' => __('This invitation link is invalid or has expired.'),
                ]);
            }

            $user = User::create([
                'name' => $input['name'],
                'email' => $lockedInvitation->email,
                'password' => Hash::make($input['password']),
                'mls_id' => trim((string) $input['mls_id']),
                'mls_email' => trim((string) $input['mls_email']),
                'assigned_mls_datasets' => ['stellar'],
                'mls_membership_status' => 'pending',
            ]);

            $domain = Domain::query()->create([
                'user_id' => $user->id,
                'domain_slug' => mb_strtolower(trim((string) $input['domain_slug'])),
                'is_active' => true,
                'mls_dataset' => 'stellar',
                'allowed_mls_datasets' => ['stellar'],
                'verification_status' => 'pending',
            ]);

            app(DomainOwnershipVerifier::class)->issueTxtChallenge($domain);
            app(MlsMembershipVerificationService::class)->verify($user, 'stellar');

            $lockedInvitation->markAccepted();

            return $user;
        });

        return $user;
    }
}
