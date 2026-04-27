<?php

namespace App\Console\Commands;

use App\Models\User;
use App\Services\MlsMembershipVerificationService;
use Illuminate\Console\Command;

class ReverifyMlsMembershipsCommand extends Command
{
    protected $signature = 'mls:reverify-memberships';

    protected $description = 'Re-verify active MLS memberships for subscribers';

    public function handle(MlsMembershipVerificationService $verifier): int
    {
        $count = 0;

        User::query()
            ->whereNotNull('mls_id')
            ->whereNotNull('mls_email')
            ->where(function ($query): void {
                $query->whereNull('mls_membership_next_reverify_at')
                    ->orWhere('mls_membership_next_reverify_at', '<=', now());
            })
            ->orderBy('id')
            ->chunkById(100, function ($users) use (&$count, $verifier): void {
                foreach ($users as $user) {
                    $verifier->verify($user, 'stellar');
                    $count++;
                }
            });

        $this->info("Processed {$count} MLS membership re-verifications.");

        return self::SUCCESS;
    }
}
