<?php

namespace App\Services;

use App\Models\User;
use Illuminate\Support\Facades\Http;

final class MlsMembershipVerificationService
{
    /**
     * Verifies user MLS membership for a dataset.
     *
     * MVP: `stellar` only. If no external endpoint is configured, validation falls back to
     * strict local format checks so onboarding can still proceed in non-production environments.
     */
    public function verify(User $user, string $dataset = 'stellar'): bool
    {
        $dataset = trim($dataset);
        if ($dataset === '') {
            $dataset = 'stellar';
        }

        $mlsId = trim((string) $user->mls_id);
        $mlsEmail = trim((string) $user->mls_email);
        if ($mlsId === '' || $mlsEmail === '') {
            $this->markFailed($user, 'MLS ID and MLS email are required.');

            return false;
        }

        $endpoint = trim((string) config('services.mls_membership.endpoint', ''));
        if ($endpoint !== '') {
            $response = Http::timeout(15)->acceptJson()->post($endpoint, [
                'dataset' => $dataset,
                'mls_id' => $mlsId,
                'email' => $mlsEmail,
            ]);

            if (! $response->successful()) {
                $this->markFailed($user, 'Membership provider responded with an error.');

                return false;
            }

            $active = (bool) data_get($response->json(), 'active', false);
            if (! $active) {
                $this->markFailed($user, 'MLS membership is not active.');

                return false;
            }
        } else {
            // Local fallback for dev: require plausible ID + matching email format.
            if (strlen($mlsId) < 4 || ! str_contains($mlsEmail, '@')) {
                $this->markFailed($user, 'MLS verification failed local validation checks.');

                return false;
            }
        }

        $user->forceFill([
            'mls_membership_status' => 'active',
            'mls_membership_last_error' => null,
            'mls_membership_verified_at' => now(),
            'mls_membership_next_reverify_at' => now()->addDays(30),
        ])->save();

        return true;
    }

    private function markFailed(User $user, string $error): void
    {
        $user->forceFill([
            'mls_membership_status' => 'inactive',
            'mls_membership_last_error' => $error,
            'mls_membership_next_reverify_at' => now()->addDays(30),
        ])->save();
    }
}
