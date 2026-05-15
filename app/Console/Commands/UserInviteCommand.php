<?php

namespace App\Console\Commands;

use App\Models\User;
use App\Services\UserInvitationIssuer;
use Illuminate\Console\Command;
use Illuminate\Validation\ValidationException;

class UserInviteCommand extends Command
{
    protected $signature = 'user:invite {email : Email address to invite} {--admin-id= : User id to record as inviter (defaults to first admin)}';

    protected $description = 'Send an invite-only registration email to an address';

    public function handle(UserInvitationIssuer $issuer): int
    {
        $email = (string) $this->argument('email');
        $adminId = $this->option('admin-id');

        $invitedBy = null;
        if ($adminId !== null && $adminId !== '') {
            $invitedBy = (int) $adminId;
            if (! User::query()->whereKey($invitedBy)->where('is_admin', true)->exists()) {
                $this->error('The given --admin-id is not an administrator user.');

                return self::FAILURE;
            }
        } else {
            $firstAdmin = User::query()->where('is_admin', true)->orderBy('id')->first();
            $invitedBy = $firstAdmin?->id;
        }

        try {
            $issuer->issue($email, $invitedBy);
        } catch (ValidationException $e) {
            foreach ($e->errors() as $messages) {
                foreach ($messages as $message) {
                    $this->error($message);
                }
            }

            return self::FAILURE;
        }

        $this->info('Invitation sent to '.$email.'.');

        return self::SUCCESS;
    }
}
