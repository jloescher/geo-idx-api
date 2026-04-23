<?php

namespace App\Ghl\Http\Controllers;

use Illuminate\Http\RedirectResponse;

/**
 * Revenue Impact: Deep-links GHL users into the IDX App host (`idx.platform_url`) → longer sessions → more leads.
 */
class EmbedRedirectController
{
    public function __invoke(string $locationId): RedirectResponse
    {
        $base = rtrim((string) config('ghl.urls.idx_platform'), '/');

        return redirect()->away($base.'/embed/'.$locationId);
    }
}
