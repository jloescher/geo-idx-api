<?php

namespace App\Ghl\Http\Controllers;

use App\Support\IdxPublicUrls;
use Illuminate\Http\Request;
use Illuminate\View\View;

/**
 * Revenue Impact: Marketplace installation landing clarifies value → higher install completion → more CRM-attached leads.
 */
class InstallationController
{
    public function show(Request $request): View
    {
        return view('ghl.install', [
            'idxPlatformUrl' => IdxPublicUrls::platformBaseForRequest($request),
            'idxApiPublicUrl' => IdxPublicUrls::apiBaseForRequest($request),
        ]);
    }

    public function complete(Request $request): View
    {
        $platformBase = IdxPublicUrls::platformBaseForRequest($request);
        $apiBase = IdxPublicUrls::apiBaseForRequest($request);
        $locationId = (string) session('ghl_location_id', '');

        return view('ghl.installation-complete', [
            'widgetApiKey' => session('widget_api_key'),
            'ghlLocationId' => session('ghl_location_id'),
            'apiPublicUrl' => $apiBase,
            'subscriberDashboardUrl' => $platformBase.'/dashboard',
            'manageGhlInstallUrl' => $apiBase.'/leadconnector/install',
            'leadsInsightsUrl' => $platformBase.'/leadconnectorapp',
            'subscribeUrl' => $platformBase.'/subscribe?ghl_location='.rawurlencode($locationId),
        ]);
    }
}
