<?php

namespace App\Ghl\OAuth\Controllers;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Ghl\Widgets\Models\GhlWidgetConfig;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

/**
 * Revenue Impact: MLS-approved domains only → compliant IDX distribution → enterprise-ready sales motion.
 */
class UrlRegistrationController
{
    public function show(): View
    {
        $id = session('ghl_pending_oauth_token_id');
        $token = GhlOAuthToken::query()->findOrFail($id);

        return view('ghl.register-urls', ['token' => $token]);
    }

    public function store(Request $request): RedirectResponse
    {
        $id = session('ghl_pending_oauth_token_id');
        $token = GhlOAuthToken::query()->findOrFail($id);

        $validated = $request->validate([
            'primary_url' => 'required|url|starts_with:https://',
            'additional_urls' => 'nullable|string',
            'integration_type' => 'required|in:ghl_website,external_website,both',
            'mls_agreement' => 'accepted',
            'manual_location_id' => 'nullable|string|max:64',
        ]);

        $locationId = $token->ghl_location_id ?: ($validated['manual_location_id'] ?? null);
        if (! $locationId) {
            return back()->withErrors([
                'manual_location_id' => 'Provide GHL Location ID for agency-level tokens.',
            ])->withInput();
        }

        $extras = collect(preg_split('/\r\n|\r|\n|,/', (string) ($validated['additional_urls'] ?? '')))
            ->map(fn ($s) => trim((string) $s))
            ->filter()
            ->values();

        foreach ($extras as $u) {
            if (! str_starts_with($u, 'https://')) {
                return back()->withErrors(['additional_urls' => 'Each additional URL must use https://'])->withInput();
            }
        }

        $apiKey = 'qh_'.bin2hex(random_bytes(16));

        $registered = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $token->id,
            'ghl_location_id' => $locationId,
            'ghl_company_id' => $token->ghl_company_id,
            'primary_url' => $validated['primary_url'],
            'additional_urls' => $extras->all(),
            'widget_api_key' => $apiKey,
            'integration_type' => $validated['integration_type'],
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
        ]);

        GhlWidgetConfig::query()->create([
            'ghl_location_id' => $locationId,
            'ghl_registered_url_id' => $registered->id,
        ]);

        session()->forget('ghl_pending_oauth_token_id');
        session()->flash('widget_api_key', $apiKey);
        session()->flash('ghl_location_id', $locationId);

        return redirect()->route('leadconnector.installation-complete');
    }
}
