<?php

use App\Http\Controllers\Billing\SubscriptionCheckoutController;
use App\Http\Controllers\DashboardApiTokenController;
use App\Http\Controllers\DashboardController;
use App\Http\Controllers\DashboardDomainController;
use App\Http\Controllers\DashboardExtraDomainController;
use App\Http\Controllers\DashboardLeadsController;
use App\Http\Controllers\DashboardWidgetAppearanceController;
use App\Http\Controllers\DashboardWidgetValidateController;
use App\Http\Controllers\Marketing\SalesPageController;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;

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

        Route::middleware('auth')->group(function (): void {
            Route::get('/dashboard', DashboardController::class)->name('dashboard.index');
            Route::get('/dashboard/leads', DashboardLeadsController::class)->name('dashboard.leads');
            Route::post('/dashboard/widget-validate', DashboardWidgetValidateController::class)->name('dashboard.widget-validate');
            Route::post('/dashboard/widget-appearance', DashboardWidgetAppearanceController::class)->name('dashboard.widget-appearance');
            Route::post('/dashboard/domains', [DashboardDomainController::class, 'store'])->name('dashboard.domains.store');
            Route::delete('/dashboard/domains/{domain}', [DashboardDomainController::class, 'destroy'])->name('dashboard.domains.destroy');
            Route::post('/dashboard/domains/{domain}/verify-txt', [DashboardDomainController::class, 'verifyTxt'])->name('dashboard.domains.verify-txt');
            Route::post('/dashboard/domains/{domain}/verify-ghl', [DashboardDomainController::class, 'verifyGhl'])->name('dashboard.domains.verify-ghl');
            Route::post('/dashboard/billing/extra-domain', [DashboardExtraDomainController::class, 'store'])->name('dashboard.billing.extra-domain');
            Route::post('/dashboard/api-tokens', [DashboardApiTokenController::class, 'store'])->name('dashboard.api-tokens.store');
            Route::delete('/dashboard/api-tokens/{token}', [DashboardApiTokenController::class, 'destroy'])->name('dashboard.api-tokens.destroy');
            Route::view('/leadconnectorapp', 'leadconnector.app')->name('leadconnector.app');
            Route::get('/billing/checkout', SubscriptionCheckoutController::class)->name('billing.checkout');
        });
    });
}

foreach ($apiHosts as $apiHost) {
    Route::domain($apiHost)->get('/', function (Request $request): RedirectResponse {
        $host = (string) $request->getHost();

        $salesHost = str_replace('-api.', '.', $host);

        return redirect()->to(sprintf('https://%s', $salesHost), 302);
    })->name('api.root.redirect');
}
