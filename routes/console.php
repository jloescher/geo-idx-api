<?php

use App\Jobs\RefreshDomainListingsCacheJob;
use App\Models\Domain;
use Illuminate\Foundation\Inspiring;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Schedule;

Artisan::command('inspire', function () {
    $this->comment(Inspiring::quote());
})->purpose('Display an inspiring quote');

Schedule::command('ghl:refresh-tokens')->hourly()->withoutOverlapping();

Schedule::call(function (): void {
    Domain::query()->active()->pluck('domain_slug')->each(function (string $slug): void {
        RefreshDomainListingsCacheJob::dispatch($slug);
    });
})->everyFifteenMinutes()->name('bridge-listings-cache-refresh')->withoutOverlapping();
