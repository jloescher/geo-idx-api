<?php

namespace App\Ghl\Http\Controllers;

use Illuminate\View\View;

/**
 * Revenue Impact: Marketplace installation landing clarifies value → higher install completion → more CRM-attached leads.
 */
class InstallationController
{
    public function show(): View
    {
        return view('ghl.install');
    }

    public function complete(): View
    {
        return view('ghl.installation-complete', [
            'widgetApiKey' => session('widget_api_key'),
            'ghlLocationId' => session('ghl_location_id'),
        ]);
    }
}
