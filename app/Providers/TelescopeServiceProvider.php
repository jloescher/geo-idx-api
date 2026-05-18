<?php

namespace App\Providers;

use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Gate;
use Laravel\Telescope\IncomingEntry;
use Laravel\Telescope\Telescope;
use Laravel\Telescope\TelescopeApplicationServiceProvider;

class TelescopeServiceProvider extends TelescopeApplicationServiceProvider
{
    /**
     * Register any application services.
     */
    public function register(): void
    {
        // Telescope::night();

        $this->hideSensitiveRequestDetails();

        $recordAll = $this->shouldRecordAllEntries();

        Telescope::filter(function (IncomingEntry $entry) use ($recordAll) {
            if ($recordAll) {
                return true;
            }

            return $entry->isReportableException() ||
                   $entry->isFailedRequest() ||
                   $entry->isFailedJob() ||
                   $entry->isScheduledTask() ||
                   $entry->isSlowQuery() ||
                   $entry->hasMonitoredTag();
        });

        Telescope::filterBatch(function (Collection $entries) use ($recordAll) {
            if ($recordAll) {
                return true;
            }

            return $entries->contains(function (IncomingEntry $entry) {
                return $entry->isReportableException() ||
                    $entry->isFailedJob() ||
                    $entry->isScheduledTask() ||
                    $entry->isSlowQuery() ||
                    $entry->hasMonitoredTag();
            });
        });
    }

    protected function shouldRecordAllEntries(): bool
    {
        if ($this->app->environment(['local', 'staging'])) {
            return true;
        }

        return filter_var(env('TELESCOPE_RECORD_ALL', false), FILTER_VALIDATE_BOOL);
    }

    /**
     * Prevent sensitive request details from being logged by Telescope.
     */
    protected function hideSensitiveRequestDetails(): void
    {
        if ($this->app->environment(['local', 'staging'])) {
            return;
        }

        Telescope::hideRequestParameters(['_token']);

        Telescope::hideRequestHeaders([
            'cookie',
            'x-csrf-token',
            'x-xsrf-token',
        ]);
    }

    /**
     * Register the Telescope gate.
     *
     * This gate determines who can access Telescope in non-local environments.
     */
    protected function gate(): void
    {
        Gate::define('viewTelescope', function ($user = null): bool {
            if (app()->environment(['local', 'staging'])) {
                return true;
            }

            return app()->environment('production');
        });
    }
}
