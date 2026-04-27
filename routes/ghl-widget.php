<?php

use App\Ghl\Widgets\Controllers\WidgetLeadIngestController;
use App\Ghl\Widgets\Controllers\WidgetLoaderController;
use App\Ghl\Widgets\Controllers\WidgetSurfaceController;
use App\Ghl\Widgets\Middleware\AppendRegisteredOriginCors;
use App\Ghl\Widgets\Middleware\ValidateWidgetApiKey;
use App\Ghl\Widgets\Middleware\ValidateWidgetOrigin;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Ghl\Widgets\Support\OriginMatcher;
use App\Http\Controllers\Api\WidgetListingsSearchController;
use App\Models\User;
use App\Widgets\DirectSiteWidgetContext;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;

Route::get('/widget/loader.js', WidgetLoaderController::class)
    ->middleware('throttle:600,1')
    ->name('leadconnector.widget.loader');

$widgetGate = [
    ValidateWidgetApiKey::class,
    ValidateWidgetOrigin::class,
    AppendRegisteredOriginCors::class,
    'throttle:'.(int) config('ghl.widgets.rate_limit_per_minute', 120).',1',
];

Route::middleware($widgetGate)->group(function () {
    Route::get('/widget/config/{apiKey}', [WidgetSurfaceController::class, 'config']);
    Route::get('/widget/search/{apiKey}', [WidgetSurfaceController::class, 'search']);
    Route::get('/widget/lead-form/{apiKey}', [WidgetSurfaceController::class, 'leadForm']);
    Route::get('/widget/showcase/{apiKey}', [WidgetSurfaceController::class, 'showcase']);
    Route::post('/widget/api/listings-search', WidgetListingsSearchController::class);
});

Route::options('/widget/api/leads', function (Request $request) {
    $key = (string) $request->query('api_key', '');
    if ($key === '') {
        return response('Missing api_key query for CORS preflight', 400);
    }
    $row = GhlRegisteredUrl::query()->where('widget_api_key', $key)->where('widget_access_enabled', true)->first();
    $allowedOrigins = null;
    if ($row !== null) {
        $allowedOrigins = $row->allowedOrigins();
    } else {
        $user = User::query()->where('widget_embed_site_key', $key)->first();
        if ($user === null) {
            return response('Invalid key', 404);
        }
        $allowedOrigins = (new DirectSiteWidgetContext($user))->allowedOrigins();
    }
    $origin = (string) $request->headers->get('Origin', '');
    $response = response('', 204);
    $validatedOrigin = OriginMatcher::allowedOrigin($origin, $allowedOrigins);
    if ($validatedOrigin !== null) {
        $response->headers->set('Access-Control-Allow-Origin', $validatedOrigin);
        $response->headers->set('Vary', 'Origin');
    }
    $response->headers->set('Access-Control-Allow-Methods', 'POST, OPTIONS');
    $response->headers->set('Access-Control-Allow-Headers', 'Content-Type, X-Quantyra-Widget-Key');

    return $response;
})->middleware('throttle:120,1');

Route::options('/widget/api/listings-search', function (Request $request) {
    $key = (string) $request->query('api_key', '');
    if ($key === '') {
        return response('Missing api_key query for CORS preflight', 400);
    }
    $row = GhlRegisteredUrl::query()->where('widget_api_key', $key)->where('widget_access_enabled', true)->first();
    $allowedOrigins = null;
    if ($row !== null) {
        $allowedOrigins = $row->allowedOrigins();
    } else {
        $user = User::query()->where('widget_embed_site_key', $key)->first();
        if ($user === null) {
            return response('Invalid key', 404);
        }
        $allowedOrigins = (new DirectSiteWidgetContext($user))->allowedOrigins();
    }
    $origin = (string) $request->headers->get('Origin', '');
    $response = response('', 204);
    $validatedOrigin = OriginMatcher::allowedOrigin($origin, $allowedOrigins);
    if ($validatedOrigin !== null) {
        $response->headers->set('Access-Control-Allow-Origin', $validatedOrigin);
        $response->headers->set('Vary', 'Origin');
    }
    $response->headers->set('Access-Control-Allow-Methods', 'POST, OPTIONS');
    $response->headers->set('Access-Control-Allow-Headers', 'Content-Type, X-Quantyra-Widget-Key');

    return $response;
})->middleware('throttle:120,1');

Route::post('/widget/api/leads', WidgetLeadIngestController::class)
    ->middleware($widgetGate);
