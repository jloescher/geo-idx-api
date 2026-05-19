<?php

namespace App\Services;

use App\Models\Domain;

class DomainOwnershipVerifier
{
    public function issueTxtChallenge(Domain $domain): Domain
    {
        if ($domain->txt_verification_name && $domain->txt_verification_value) {
            return $domain;
        }

        $token = 'geoidx-verify='.bin2hex(random_bytes(16));
        $domain->forceFill([
            'txt_verification_name' => '_geoidx.'.$domain->domain_slug,
            'txt_verification_value' => $token,
            'verification_status' => 'pending',
            'verification_method' => null,
        ])->save();

        return $domain->refresh();
    }

    public function verifyTxtRecord(Domain $domain): bool
    {
        $host = (string) $domain->txt_verification_name;
        $expected = trim((string) $domain->txt_verification_value);
        if ($host === '' || $expected === '') {
            return false;
        }

        $records = dns_get_record($host, DNS_TXT);
        if (! is_array($records)) {
            return false;
        }

        foreach ($records as $record) {
            $txt = trim((string) ($record['txt'] ?? ''));
            if ($txt !== '' && hash_equals($expected, $txt)) {
                $domain->forceFill([
                    'verification_status' => 'verified',
                    'verification_method' => 'txt',
                    'txt_verified_at' => now(),
                    'verification_checked_at' => now(),
                ])->save();

                return true;
            }
        }

        $domain->forceFill([
            'verification_checked_at' => now(),
        ])->save();

        return false;
    }
}
