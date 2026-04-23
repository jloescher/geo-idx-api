<?php

namespace App\Ghl\Widgets\Controllers;

use App\Ghl\Sync\Jobs\SyncLeadToGhlJob;
use App\Ghl\Sync\Models\QuantyraLead;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Revenue Impact: Widget POST creates CRM-bound leads immediately → measurable pipeline ROI for agencies.
 */
class WidgetLeadIngestController
{
    public function __invoke(Request $request): JsonResponse
    {
        $row = $request->attributes->get('ghl_registered_url');

        $data = $request->validate([
            'api_key' => 'required|string',
            'lead_type' => 'required|string|max:64',
            'first_name' => 'nullable|string|max:120',
            'last_name' => 'nullable|string|max:120',
            'email' => 'nullable|email',
            'phone' => 'nullable|string|max:40',
            'listing_id' => 'nullable|string|max:64',
            'quantyra_domain' => 'nullable|string|max:255',
        ]);

        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => $row->ghl_location_id,
            'lead_type' => $data['lead_type'],
            'source' => 'widget',
            'payload' => collect($data)->except('api_key')->all(),
            'listing_id' => $data['listing_id'] ?? null,
            'quantyra_domain' => $data['quantyra_domain'] ?? null,
        ]);

        SyncLeadToGhlJob::dispatch($lead->id);

        return response()->json(['id' => $lead->id], 201);
    }
}
