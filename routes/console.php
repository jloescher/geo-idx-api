<?php

use App\Jobs\BridgeSyncJob;
use App\Jobs\PurgeClosedListingsJob;
use App\Jobs\RefreshCryptoPricingJob;
use App\Jobs\RefreshDomainListingsCacheJob;
use App\Jobs\RefreshGisSourceMetadataJob;
use App\Models\Domain;
use Illuminate\Foundation\Inspiring;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Schedule;

Artisan::command('inspire', function () {
    $this->comment(Inspiring::quote());
})->purpose('Display an inspiring quote');

Schedule::command('ghl:refresh-tokens')->hourly()->withoutOverlapping();
Schedule::call(function (): void {
    RefreshCryptoPricingJob::dispatch()
        ->onQueue((string) config('coingecko.queue'));
})->everyTenMinutes()->name('coingecko-price-refresh')->withoutOverlapping();

Schedule::call(function (): void {
    Domain::query()->active()->pluck('domain_slug')->each(function (string $slug): void {
        RefreshDomainListingsCacheJob::dispatch($slug);
    });
})->everyFifteenMinutes()->name('bridge-listings-cache-refresh')->withoutOverlapping();

Schedule::call(function (): void {
    BridgeSyncJob::dispatch();
})->everyFifteenMinutes()->name('bridge-listings-replica-sync')->withoutOverlapping();

Schedule::call(function (): void {
    PurgeClosedListingsJob::dispatch();
})->dailyAt('03:05')->name('bridge-listings-purge-closed')->withoutOverlapping();

Schedule::call(function (): void {
    RefreshGisSourceMetadataJob::dispatch()->onQueue((string) config('gis.queue'));
})->weeklyOn(1, '6:30')->name('gis-source-metadata-probe')->withoutOverlapping();

Schedule::command('mls:reverify-memberships')
    ->monthlyOn(1, '03:00')
    ->name('mls-membership-reverify')
    ->withoutOverlapping();

Schedule::command('leads:send-alerts')
    ->everyFifteenMinutes()
    ->name('leads-alert-digests')
    ->withoutOverlapping();

Schedule::command(sprintf('agent:prune-share-links --days=%d', (int) config('agent_portal.share_links.prune_days', 90)))
    ->dailyAt('03:30')
    ->name('agent-share-links-prune')
    ->withoutOverlapping();

Schedule::command('agent:process-due-alerts')
    ->everyFifteenMinutes()
    ->name('agent-process-due-alerts')
    ->withoutOverlapping();
