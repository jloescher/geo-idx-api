<?php

namespace App\Services\Bridge;

use App\Models\BridgeProxyAuditLog;
use Illuminate\Http\Request;

class BridgeProxyAuditLogger
{
    /**
     * Revenue impact: per-request MLS audit rows make compliance responses cheap,
     * avoiding revenue-killing suspension during Stellar MLS audits.
     */
    public function log(
        Request $request,
        string $requestType,
        ?int $listingCount,
        ?string $domainSlug,
        ?string $tokenName,
        ?int $userId,
    ): void {
        BridgeProxyAuditLog::query()->create([
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
