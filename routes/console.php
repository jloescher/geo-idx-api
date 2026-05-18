<?php

use App\Jobs\BridgeSyncJob;
use App\Jobs\PurgeClosedListingsJob;
use App\Jobs\RefreshCryptoPricingJob;
use App\Jobs\RefreshGisSourceMetadataJob;
use App\Jobs\SparkSyncJob;
use Illuminate\Foundation\Inspiring;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Schedule;

Artisan::command('inspire', function () {
    $this->comment(Inspiring::quote());
})->purpose('Display an inspiring quote');

Schedule::call(function (): void {
    RefreshCryptoPricingJob::dispatch()
        ->onQueue((string) config('coingecko.queue'));
})->everyTenMinutes()->name('coingecko-price-refresh')->withoutOverlapping();

Schedule::command('mls:refresh-cache')
    ->everyFifteenMinutes()
    ->name('mls-listings-cache-refresh')
    ->withoutOverlapping();

Schedule::call(function (): void {
    BridgeSyncJob::dispatch()->onQueue((string) config('bridge.sync_fetch_queue', 'bridge-sync-fetch'));
})->everyFifteenMinutes()->name('bridge-listings-replica-sync')->withoutOverlapping();

Schedule::call(function (): void {
    SparkSyncJob::dispatch()->onQueue((string) config('spark.sync_fetch_queue', 'spark-sync-fetch'));
})->everyFifteenMinutes()->name('spark-listings-replica-sync')->withoutOverlapping();

Schedule::command('bridge:purge-replica-pages')
    ->dailyAt('04:15')
    ->name('bridge-replica-pages-purge')
    ->withoutOverlapping();

Schedule::call(function (): void {
    PurgeClosedListingsJob::dispatch();
})->dailyAt('03:05')->name('bridge-listings-purge-closed')->withoutOverlapping();

Schedule::call(function (): void {
    RefreshGisSourceMetadataJob::dispatch()->onQueue((string) config('gis.queue'));
})->weeklyOn(1, '6:30')->name('gis-source-metadata-probe')->withoutOverlapping();
