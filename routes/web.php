<?php

use App\Http\Controllers\Billing\SubscriptionCheckoutController;
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

$platformHosts = $parseHostList((string) env(
    'IDX_PLATFORM_HOSTS',
    'idx.quantyralabs.cc,dev-idx.quantyralabs.cc,staging-idx.quantyralabs.cc'
));

$apiHosts = $parseHostList((string) env(
    'IDX_API_HOSTS',
    'idx-api.quantyralabs.cc,dev-idx-api.quantyralabs.cc,staging-idx-api.quantyralabs.cc'
));

foreach ($platformHosts as $platformHost) {
    Route::domain($platformHost)->group(function (): void {
        Route::get('/', SalesPageController::class)->name('marketing.sales');

        Route::middleware('auth')->group(function (): void {
            Route::view('/dashboard', 'dashboard.index')->name('dashboard.index');
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
