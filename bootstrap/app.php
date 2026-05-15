<?php

use App\Http\Controllers\Api\ImageProxyController;
use App\Http\Middleware\CheckMlsAccess;
use App\Http\Middleware\DomainOrTokenAuth;
use App\Http\Middleware\EnsureUserIsAdmin;
use App\Http\Middleware\ProtectMonitoringDashboard;
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
            Route::middleware(['api', 'domain.token'])
                ->group(function (): void {
                    Route::get('/images/{listingKey}/{photoId}', [ImageProxyController::class, 'show'])
                        ->where('listingKey', '[A-Za-z0-9_\-]+')
                        ->where('photoId', '[A-Za-z0-9_\-]+');
                });
        },
    )
    ->withMiddleware(function (Middleware $middleware): void {
        $middleware->trustProxies(at: '*');

        $middleware->statefulApi();

        $middleware->alias([
            'domain.token' => DomainOrTokenAuth::class,
            'mls.access' => CheckMlsAccess::class,
            'monitoring.basic-auth' => ProtectMonitoringDashboard::class,
            'admin' => EnsureUserIsAdmin::class,
        ]);
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        //
    })->create();
