<?php

use App\Http\Controllers\Auth\RegisterInvitationController;
use App\Http\Controllers\DashboardApiTokenController;
use App\Http\Controllers\DashboardDomainController;
use App\Http\Controllers\DashboardDomainMlsController;
use App\Http\Controllers\DashboardUserInvitationController;
use App\Http\Controllers\Marketing\SalesPageController;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;
use Symfony\Component\HttpFoundation\BinaryFileResponse;

$parseHostList = static function (string $value): array {
    return collect(explode(',', $value))
        ->map(static fn (string $host): string => trim($host))
        ->filter(static fn (string $host): bool => $host !== '')
        ->values()
        ->all();
};

$appHost = (string) parse_url((string) env('APP_URL', 'http://localhost'), PHP_URL_HOST);
$apiHost = (string) parse_url((string) env('API_URL', (string) env('APP_URL', 'http://localhost')), PHP_URL_HOST);
$defaultPlatformHost = str_contains($appHost, '-api.')
    ? str_replace('-api.', '.', $appHost)
    : $appHost;

$platformHosts = array_values(array_unique([
    ...$parseHostList((string) env(
        'IDX_PLATFORM_HOSTS',
        implode(',', array_values(array_filter([$defaultPlatformHost, 'localhost', '127.0.0.1'])))
    )),
    'localhost',
    '127.0.0.1',
]));

$apiHosts = $parseHostList((string) env(
    'IDX_API_HOSTS',
    implode(',', array_values(array_filter([$apiHost])))
));

foreach ($platformHosts as $platformHost) {
    Route::domain($platformHost)->group(function (): void {
        Route::get('/', SalesPageController::class)->name('marketing.sales');

        Route::middleware(['web', 'guest', 'turnstile'])->group(function (): void {
            Route::get('/register/{token}', [RegisterInvitationController::class, 'show'])
                ->middleware('throttle:60,1')
                ->where('token', '[A-Za-z0-9]+')
                ->name('register.invite');
            Route::post('/register', [RegisterInvitationController::class, 'store'])
                ->middleware('throttle:registration')
                ->name('register.store');
        });

        Route::get('/filament-dashboard', function (Request $request): RedirectResponse {
            $target = '/dashboard';
            if ($request->getQueryString() !== null && $request->getQueryString() !== '') {
                $target .= '?'.$request->getQueryString();
            }

            return redirect()->to($target, 301);
        })->middleware('web');

        Route::middleware('auth')->group(function (): void {
            Route::post('/dashboard/domains', [DashboardDomainController::class, 'store'])->name('dashboard.domains.store');
            Route::delete('/dashboard/domains/{domain}', [DashboardDomainController::class, 'destroy'])->name('dashboard.domains.destroy');
            Route::post('/dashboard/domains/{domain}/verify-txt', [DashboardDomainController::class, 'verifyTxt'])->name('dashboard.domains.verify-txt');
            Route::put('/dashboard/domains/{domain}/mls', [DashboardDomainMlsController::class, 'update'])->name('dashboard.domains.mls.update');
            Route::post('/dashboard/api-tokens', [DashboardApiTokenController::class, 'store'])->name('dashboard.api-tokens.store');
            Route::delete('/dashboard/api-tokens/{token}', [DashboardApiTokenController::class, 'destroy'])->name('dashboard.api-tokens.destroy');
            Route::post('/dashboard/invitations', [DashboardUserInvitationController::class, 'store'])
                ->middleware(['admin', 'throttle:dashboard-invitations'])
                ->name('dashboard.invitations.store');
        });
    });
}

foreach ($apiHosts as $apiHost) {
    Route::domain($apiHost)->group(function (): void {
        Route::get('/openapi.json', function (): BinaryFileResponse {
            return response()->file(
                base_path('docs/yaak-api-collection.json'),
                ['Content-Type' => 'application/json; charset=UTF-8']
            );
        })->name('api.openapi');

        Route::view('/swagger', 'docs.swagger', [
            'openApiSpecUrl' => '/openapi.json',
            'openApiVersion' => '3.1.0',
        ])->name('api.swagger');

        Route::get('/', function (Request $request): RedirectResponse {
            $host = (string) $request->getHost();

            $salesHost = str_replace('-api.', '.', $host);

            return redirect()->to(sprintf('https://%s', $salesHost), 302);
        })->name('api.root.redirect');
    });
}
