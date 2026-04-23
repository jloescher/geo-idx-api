<?php

use App\Ghl\Http\Controllers\EmbedRedirectController;
use App\Ghl\Http\Controllers\InstallationController;
use App\Ghl\Http\Middleware\RequiresPendingGhlInstallSession;
use App\Ghl\OAuth\Controllers\AuthorizeController;
use App\Ghl\OAuth\Controllers\CallbackController;
use App\Ghl\OAuth\Controllers\RefreshController;
use App\Ghl\OAuth\Controllers\UrlRegistrationController;
use App\Ghl\Webhooks\Controllers\WebhookController;
use App\Http\Middleware\VerifyGhlWebhookSignature;
use Illuminate\Support\Facades\Route;

Route::get('/leadconnector/install', [InstallationController::class, 'show'])->name('leadconnector.install');

Route::get('/oauth/leadconnector/authorize', AuthorizeController::class)->name('leadconnector.oauth.authorize');
Route::get('/oauth/leadconnector/callback', CallbackController::class)->name('leadconnector.oauth.callback');
Route::post('/oauth/leadconnector/refresh', RefreshController::class)->name('leadconnector.oauth.refresh');

Route::middleware([RequiresPendingGhlInstallSession::class])->group(function () {
    Route::get('/leadconnector/register-urls', [UrlRegistrationController::class, 'show'])->name('leadconnector.register-urls');
    Route::post('/leadconnector/register-urls', [UrlRegistrationController::class, 'store'])->name('leadconnector.register-urls.store');
});

Route::get('/leadconnector/installation-complete', [InstallationController::class, 'complete'])->name('leadconnector.installation-complete');

Route::post('/webhooks/leadconnector', WebhookController::class)
    ->middleware([VerifyGhlWebhookSignature::class, 'throttle:120,1'])
    ->name('leadconnector.webhooks');

Route::get('/leadconnector/embed/{locationId}', EmbedRedirectController::class)->name('leadconnector.embed');
