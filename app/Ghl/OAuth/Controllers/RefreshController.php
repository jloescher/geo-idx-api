<?php

namespace App\Ghl\OAuth\Controllers;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\OAuth\Services\TokenRefreshService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Revenue Impact: Prevents token expiry churn support tickets → uninterrupted lead ingestion → higher LTV.
 */
class RefreshController
{
    public function __invoke(Request $request, TokenRefreshService $refresh): JsonResponse
    {
        $admin = (string) config('ghl.admin.refresh_token');
        if ($admin === '' || ! hash_equals($admin, (string) $request->header('X-Quantyra-Admin-Token', ''))) {
            abort(403);
        }

        $id = (int) $request->input('token_id');
        $token = GhlOAuthToken::query()->findOrFail($id);
        $refresh->refresh($token);

        return response()->json(['ok' => true, 'expires_at' => $token->fresh()->expires_at]);
    }
}
