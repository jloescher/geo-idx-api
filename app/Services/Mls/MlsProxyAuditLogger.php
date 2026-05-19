<?php

namespace App\Services\Mls;

use App\Models\MlsProxyAuditLog;
use Illuminate\Http\Request;

class MlsProxyAuditLogger
{
    /**
     * Revenue impact: per-request MLS audit rows make compliance responses cheap,
     * avoiding revenue-killing suspension during MLS audits.
     */
    public function log(
        Request $request,
        string $requestType,
        ?int $listingCount,
        ?string $domainSlug,
        ?string $tokenName,
        ?int $userId,
    ): void {
        MlsProxyAuditLog::query()->create([
            'logged_at' => now(),
            'domain_slug' => $domainSlug,
            'token_name' => $tokenName,
            'request_type' => $requestType,
            'listing_count' => $listingCount,
            'ip_address' => $request->ip(),
            'user_id' => $userId,
        ]);
    }
}
