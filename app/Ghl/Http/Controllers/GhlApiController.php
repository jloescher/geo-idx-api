<?php

namespace App\Ghl\Http\Controllers;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Sync\Models\GhlInstalledLocation;
use App\Ghl\Sync\Models\GhlLeadMapping;
use App\Ghl\Sync\Models\QuantyraLead;
use App\Ghl\Widgets\Models\GhlWidgetConfig;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Revenue Impact: Teaser stats + upgrade URLs inside GHL → subscription conversions on the IDX App host.
 */
class GhlApiController
{
    public function leads(Request $request): JsonResponse
    {
        $token = $this->token($request);
        $locationId = $token->ghl_location_id ?? (string) $request->query('location_id', '');
        if ($locationId === '') {
            return response()->json(['message' => 'location_id query required for agency tokens'], 422);
        }

        $leads = QuantyraLead::query()
            ->where('ghl_location_id', $locationId)
            ->orderByDesc('id')
            ->paginate(25);

        return response()->json($leads);
    }

    public function lead(Request $request, int $id): JsonResponse
    {
        $token = $this->token($request);
        $locationId = $token->ghl_location_id ?? (string) $request->query('location_id', '');
        $lead = QuantyraLead::query()->findOrFail($id);
        if ($locationId !== '' && $lead->ghl_location_id !== $locationId) {
            abort(404);
        }

        return response()->json($lead);
    }

    public function subscriptions(Request $request): JsonResponse
    {
        $token = $this->token($request);
        $locationId = $token->ghl_location_id ?? (string) $request->query('location_id', '');
        if ($locationId === '') {
            return response()->json(['message' => 'location_id query required'], 422);
        }

        $loc = GhlInstalledLocation::query()
            ->where('ghl_location_id', $locationId)
            ->first();

        $upgrade = rtrim((string) config('ghl.urls.idx_platform'), '/').'/subscribe?ghl_location='.urlencode($locationId);

        return response()->json([
            'location_id' => $locationId,
            'subscription_status' => $loc?->subscription_status ?? 'none',
            'subscription_id' => $loc?->subscription_id,
            'upgrade_url' => $upgrade,
        ]);
    }

    public function stats(Request $request): JsonResponse
    {
        $token = $this->token($request);
        $locationId = $token->ghl_location_id ?? (string) $request->query('location_id', '');
        if ($locationId === '') {
            return response()->json(['message' => 'location_id query required'], 422);
        }

        $loc = GhlInstalledLocation::query()
            ->where('ghl_location_id', $locationId)
            ->first();

        $leadCount = QuantyraLead::query()->where('ghl_location_id', $locationId)->count();

        return response()->json([
            'location_id' => $locationId,
            'stats' => [
                'mls_request_count' => (int) ($loc?->mls_request_count ?? 0),
                'lead_count' => $leadCount,
            ],
            'subscription' => [
                'status' => $loc?->subscription_status ?? 'none',
                'upgrade_url' => rtrim((string) config('ghl.urls.idx_platform'), '/').'/subscribe?ghl_location='.urlencode($locationId),
            ],
            'gated_features' => [
                'full_mls_access' => ($loc?->subscription_status === 'active'),
            ],
        ]);
    }

    public function config(Request $request): JsonResponse
    {
        $token = $this->token($request);
        $locationId = $token->ghl_location_id ?? (string) $request->query('location_id', '');
        if ($locationId === '') {
            return response()->json(['message' => 'location_id query required'], 422);
        }

        $widget = GhlWidgetConfig::query()->where('ghl_location_id', $locationId)->first();
        $mappings = GhlLeadMapping::query()->get();

        return response()->json([
            'location_id' => $locationId,
            'widget' => $widget,
            'lead_mappings' => $mappings,
        ]);
    }

    private function token(Request $request): GhlOAuthToken
    {
        /** @var GhlOAuthToken $t */
        $t = $request->attributes->get('ghl_oauth_token');

        return $t;
    }
}
