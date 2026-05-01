<?php

use App\Http\Controllers\AgentPortal\AgentAlertController;
use App\Http\Controllers\AgentPortal\AgentAlertTemplateController;
use App\Http\Controllers\AgentPortal\AgentAutomationSettingsController;
use App\Http\Controllers\AgentPortal\AgentCompsController;
use App\Http\Controllers\AgentPortal\AgentContactController;
use App\Http\Controllers\AgentPortal\AgentDashboardSummaryController;
use App\Http\Controllers\AgentPortal\AgentFeedAccessController;
use App\Http\Controllers\AgentPortal\AgentPortalSettingsController;
use App\Http\Controllers\AgentPortal\AgentSearchController;
use App\Http\Controllers\AgentPortal\AgentShareLinkController;
use App\Http\Controllers\Billing\SubscriptionCheckoutController;
use App\Http\Controllers\DashboardApiTokenController;
use App\Http\Controllers\DashboardController;
use App\Http\Controllers\DashboardDomainController;
use App\Http\Controllers\DashboardExtraDomainController;
use App\Http\Controllers\DashboardLeadsController;
use App\Http\Controllers\DashboardMlsMembershipController;
use App\Http\Controllers\DashboardWidgetAppearanceController;
use App\Http\Controllers\DashboardWidgetValidateController;
use App\Http\Controllers\Marketing\SalesPageController;
use App\Http\Controllers\Marketing\SharedLinkController;
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
        Route::get('/shared/{token}/{slug?}', [SharedLinkController::class, 'show'])->name('marketing.shared.show');
        Route::post('/shared/{token}/execute', [SharedLinkController::class, 'execute'])->name('marketing.shared.execute');

        Route::middleware('auth')->group(function (): void {
            Route::get('/dashboard', DashboardController::class)->name('dashboard.index');
            Route::get('/dashboard/leads', DashboardLeadsController::class)->name('dashboard.leads');
            Route::post('/dashboard/widget-validate', DashboardWidgetValidateController::class)->name('dashboard.widget-validate');
            Route::post('/dashboard/widget-appearance', DashboardWidgetAppearanceController::class)->name('dashboard.widget-appearance');
            Route::post('/dashboard/mls-membership', DashboardMlsMembershipController::class)->name('dashboard.mls-membership.store');
            Route::post('/dashboard/domains', [DashboardDomainController::class, 'store'])->name('dashboard.domains.store');
            Route::delete('/dashboard/domains/{domain}', [DashboardDomainController::class, 'destroy'])->name('dashboard.domains.destroy');
            Route::post('/dashboard/domains/{domain}/verify-txt', [DashboardDomainController::class, 'verifyTxt'])->name('dashboard.domains.verify-txt');
            Route::post('/dashboard/domains/{domain}/verify-ghl', [DashboardDomainController::class, 'verifyGhl'])->name('dashboard.domains.verify-ghl');
            Route::post('/dashboard/billing/extra-domain', [DashboardExtraDomainController::class, 'store'])->name('dashboard.billing.extra-domain');
            Route::post('/dashboard/api-tokens', [DashboardApiTokenController::class, 'store'])->name('dashboard.api-tokens.store');
            Route::delete('/dashboard/api-tokens/{token}', [DashboardApiTokenController::class, 'destroy'])->name('dashboard.api-tokens.destroy');
            Route::view('/leadconnectorapp', 'leadconnector.app')->name('leadconnector.app');
            Route::get('/billing/checkout', SubscriptionCheckoutController::class)->name('billing.checkout');
            Route::get('/agent/feed-access', AgentFeedAccessController::class)->name('agent.feed-access');
            Route::middleware('agent.module:dashboard')->group(function (): void {
                Route::get('/agent/dashboard/summary', AgentDashboardSummaryController::class)->name('agent.dashboard.summary');
                Route::post('/agent/dashboard/events', [AgentDashboardSummaryController::class, 'recordEvent'])->name('agent.dashboard.events.record');
            });
            Route::get('/agent/settings', [AgentPortalSettingsController::class, 'show'])->name('agent.settings.show');
            Route::put('/agent/settings', [AgentPortalSettingsController::class, 'update'])->name('agent.settings.update');
            Route::get('/agent/settings/feature-flags', [AgentPortalSettingsController::class, 'featureFlags'])->name('agent.settings.feature-flags');
            Route::put('/agent/settings/feature-flags', [AgentPortalSettingsController::class, 'updateFeatureFlags'])->name('agent.settings.feature-flags.update');
            Route::post('/agent/settings/onboarding-checklist/dismiss', [AgentPortalSettingsController::class, 'dismissOnboardingChecklist'])->name('agent.settings.onboarding-checklist.dismiss');
            Route::post('/agent/settings/onboarding-checklist/restore', [AgentPortalSettingsController::class, 'restoreOnboardingChecklist'])->name('agent.settings.onboarding-checklist.restore');
            Route::middleware('agent.module:contacts')->group(function (): void {
                Route::get('/agent/contacts', [AgentContactController::class, 'index'])->name('agent.contacts.index');
                Route::post('/agent/contacts/bulk/status', [AgentContactController::class, 'bulkStatus'])->name('agent.contacts.bulk.status');
                Route::post('/agent/contacts/bulk/delete', [AgentContactController::class, 'bulkDelete'])->name('agent.contacts.bulk.delete');
                Route::get('/agent/contacts/export.csv', [AgentContactController::class, 'exportCsv'])->name('agent.contacts.export');
                Route::post('/agent/contacts/handoff/alert', [AgentContactController::class, 'createAndHandoffToAlert'])->name('agent.contacts.handoff.alert.create-contact');
                Route::get('/agent/contacts/{contactId}', [AgentContactController::class, 'show'])->name('agent.contacts.show');
                Route::get('/agent/contacts/{contactId}/activity', [AgentContactController::class, 'activity'])->name('agent.contacts.activity');
                Route::put('/agent/contacts/{contactId}', [AgentContactController::class, 'update'])->name('agent.contacts.update');
                Route::get('/agent/contacts/{contactId}/tags', [AgentContactController::class, 'getTags'])->name('agent.contacts.tags.show');
                Route::put('/agent/contacts/{contactId}/tags', [AgentContactController::class, 'syncTags'])->name('agent.contacts.tags.sync');
                Route::post('/agent/contacts/{contactId}/handoff/alert', [AgentContactController::class, 'handoffToAlert'])->name('agent.contacts.handoff.alert');
            });
            Route::middleware('agent.module:search')->group(function (): void {
                Route::get('/agent/searches', [AgentSearchController::class, 'index'])->name('agent.searches.index');
                Route::get('/agent/searches/geocode', [AgentSearchController::class, 'geocode'])->name('agent.searches.geocode');
                Route::post('/agent/searches', [AgentSearchController::class, 'store'])->name('agent.searches.store');
                Route::post('/agent/searches/execute', [AgentSearchController::class, 'execute'])->name('agent.searches.execute');
                Route::post('/agent/searches/serialize', [AgentSearchController::class, 'serialize'])->name('agent.searches.serialize');
                Route::get('/agent/searches/shared', [AgentSearchController::class, 'sharedSearch'])->name('agent.searches.shared');
                Route::get('/agent/searches/lookups/options', [AgentSearchController::class, 'lookupOptions'])->name('agent.searches.lookups.options');
                Route::get('/agent/searches/fields', [AgentSearchController::class, 'fieldCatalog'])->name('agent.searches.fields');
                Route::get('/agent/searches/{searchId}', [AgentSearchController::class, 'show'])->name('agent.searches.show');
                Route::put('/agent/searches/{searchId}', [AgentSearchController::class, 'update'])->name('agent.searches.update');
                Route::delete('/agent/searches/{searchId}', [AgentSearchController::class, 'destroy'])->name('agent.searches.destroy');
            });

            Route::middleware('agent.module:alerts')->group(function (): void {
                Route::get('/agent/alerts', [AgentAlertController::class, 'index'])->name('agent.alerts.index');
                Route::get('/agent/alerts/summary', [AgentAlertController::class, 'summary'])->name('agent.alerts.summary');
                Route::get('/agent/alerts/history', [AgentAlertController::class, 'history'])->name('agent.alerts.history');
                Route::post('/agent/alerts', [AgentAlertController::class, 'store'])->name('agent.alerts.store');
                Route::post('/agent/alerts/from-template', [AgentAlertController::class, 'fromTemplate'])->name('agent.alerts.from-template');
                Route::get('/agent/alerts/{alertId}/runs', [AgentAlertController::class, 'runs'])->name('agent.alerts.runs');
                Route::get('/agent/alerts/{alertId}', [AgentAlertController::class, 'show'])->name('agent.alerts.show');
                Route::put('/agent/alerts/{alertId}', [AgentAlertController::class, 'update'])->name('agent.alerts.update');
                Route::delete('/agent/alerts/{alertId}', [AgentAlertController::class, 'destroy'])->name('agent.alerts.destroy');
                Route::get('/agent/alert-templates', [AgentAlertTemplateController::class, 'index'])->name('agent.alert-templates.index');
                Route::post('/agent/alert-templates', [AgentAlertTemplateController::class, 'store'])->name('agent.alert-templates.store');
                Route::put('/agent/alert-templates/{templateId}', [AgentAlertTemplateController::class, 'update'])->name('agent.alert-templates.update');
                Route::delete('/agent/alert-templates/{templateId}', [AgentAlertTemplateController::class, 'destroy'])->name('agent.alert-templates.destroy');
            });
            Route::middleware('agent.module:automations')->group(function (): void {
                Route::get('/agent/automations/settings', [AgentAutomationSettingsController::class, 'show'])->name('agent.automations.settings.show');
                Route::put('/agent/automations/settings', [AgentAutomationSettingsController::class, 'update'])->name('agent.automations.settings.update');
                Route::post('/agent/automations/settings/integrations/connect', [AgentAutomationSettingsController::class, 'connect'])->name('agent.automations.settings.integrations.connect');
                Route::post('/agent/automations/settings/integrations/reconnect', [AgentAutomationSettingsController::class, 'reconnect'])->name('agent.automations.settings.integrations.reconnect');
                Route::post('/agent/automations/settings/integrations/disconnect', [AgentAutomationSettingsController::class, 'disconnect'])->name('agent.automations.settings.integrations.disconnect');
            });
            Route::middleware('agent.module:marketing')->group(function (): void {
                Route::post('/agent/comps/run', [AgentCompsController::class, 'run'])->name('agent.comps.run');
                Route::get('/agent/share-links', [AgentShareLinkController::class, 'index'])->name('agent.share-links.index');
                Route::get('/agent/share-links/export.csv', [AgentShareLinkController::class, 'exportCsv'])->name('agent.share-links.export');
                Route::get('/agent/share-links/operations', [AgentShareLinkController::class, 'operations'])->name('agent.share-links.operations');
                Route::get('/agent/share-links/operations/estimate', [AgentShareLinkController::class, 'operationsEstimate'])->name('agent.share-links.operations.estimate');
                Route::get('/agent/share-links/metrics', [AgentShareLinkController::class, 'metrics'])->name('agent.share-links.metrics');
                Route::get('/agent/share-links/metrics/history', [AgentShareLinkController::class, 'metricsHistory'])->name('agent.share-links.metrics.history');
                Route::get('/agent/share-links/metrics/history.csv', [AgentShareLinkController::class, 'metricsHistoryCsv'])->name('agent.share-links.metrics.history.csv');
                Route::middleware('agent.module:seo_landing_pages')->group(function (): void {
                    Route::get('/agent/share-links/seo-landings', [AgentShareLinkController::class, 'seoLandings'])->name('agent.share-links.seo-landings');
                });
                Route::middleware('agent.module:widgets')->group(function (): void {
                    Route::post('/agent/share-links/embed-code', [AgentShareLinkController::class, 'embedCode'])->name('agent.share-links.embed-code');
                });
                Route::post('/agent/share-links', [AgentShareLinkController::class, 'store'])->name('agent.share-links.store');
                Route::put('/agent/share-links/{shareLinkId}', [AgentShareLinkController::class, 'update'])->name('agent.share-links.update');
                Route::delete('/agent/share-links/{shareLinkId}', [AgentShareLinkController::class, 'destroy'])->name('agent.share-links.destroy');
            });
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
