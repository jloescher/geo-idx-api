<?php

namespace App\Ghl\Widgets\Controllers;

use App\Ghl\Widgets\Models\GhlWidgetConfig;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Response;

/**
 * Revenue Impact: Embeddable surfaces keep buyers on partner domains → brand lift + lead capture for Quantyra.
 */
class WidgetSurfaceController
{
    public function search(Request $request, string $apiKey): Response
    {
        return $this->shell('search', $request);
    }

    public function leadForm(Request $request, string $apiKey): Response
    {
        return $this->shell('lead-form', $request);
    }

    public function showcase(Request $request, string $apiKey): Response
    {
        return $this->shell('showcase', $request);
    }

    public function config(Request $request, string $apiKey): JsonResponse
    {
        $row = $request->attributes->get('ghl_registered_url');
        $cfg = GhlWidgetConfig::query()->where('ghl_location_id', $row->ghl_location_id)->first();

        return response()->json([
            'location_id' => $row->ghl_location_id,
            'theme' => $cfg?->widget_theme ?? 'light',
            'primary_color' => $cfg?->primary_color,
            'idx_platform' => config('ghl.urls.idx_platform'),
            'images_cdn' => config('ghl.urls.images'),
        ]);
    }

    private function shell(string $kind, Request $request): Response
    {
        $row = $request->attributes->get('ghl_registered_url');
        $loc = e($row->ghl_location_id);

        $html = '<div class="quantyra-idx-widget" data-kind="'.e($kind).'" data-location="'.$loc.'">'
            .'<strong>Quantyra GeoIDX</strong> — MLS content is served only through the approved idx-api Bridge proxy. '
            .'Connect listings UI on the IDX App (<code>'.e((string) config('ghl.urls.idx_platform')).'</code>) while keeping this embed on the IDX API.</div>';

        return response($html, 200, ['Content-Type' => 'text/html; charset=UTF-8']);
    }
}
