<?php

use App\Http\Controllers\Api\ImageProxyController;
use App\Http\Middleware\DomainOrTokenAuth;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Support\Facades\Route;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        web: __DIR__.'/../routes/web.php',
        api: __DIR__.'/../routes/api.php',
        commands: __DIR__.'/../routes/console.php',
        health: '/up',
        then: function (): void {
            Route::middleware('web')
                ->group(base_path('routes/ghl-web.php'));
            Route::middleware('api')
                ->group(base_path('routes/ghl-widget.php'));

            Route::middleware(['api', 'domain.token'])
                ->group(function (): void {
                    Route::get('/images/{listingKey}/{photoId}', [ImageProxyController::class, 'show'])
                        ->where('listingKey', '[A-Za-z0-9_\-]+')
                        ->where('photoId', '[A-Za-z0-9_\-]+');
                });
        },
    )
    ->withMiddleware(function (Middleware $middleware): void {
        $middleware->validateCsrfTokens(except: [
            'webhooks/leadconnector',
        ]);

        $middleware->alias([
            'domain.token' => DomainOrTokenAuth::class,
        ]);
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        //
    })->create();
