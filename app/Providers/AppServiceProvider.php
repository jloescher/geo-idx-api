<?php

namespace App\Providers;

use App\Services\Geocoding\GoogleGeocodingService;
use App\Support\DestructiveDatabaseCommandGuard;
use Illuminate\Console\Events\CommandStarting;
use Illuminate\Http\Middleware\HandleCors;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Event;
use Illuminate\Support\Facades\Gate;
use Illuminate\Support\ServiceProvider;
use Livewire\Volt\Volt;

class AppServiceProvider extends ServiceProvider
{
    /**
     * Register any application services.
     */
    public function register(): void
    {
        $this->app->singleton(GoogleGeocodingService::class, function (): GoogleGeocodingService {
            return new GoogleGeocodingService(
                apiKey: (string) config('geocoding.google_api_key'),
                cacheTtl: (int) config('geocoding.cache_ttl_seconds'),
                timeout: (int) config('geocoding.timeout_seconds'),
            );
        });
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        $this->loadMigrationsFrom(database_path('migrations/ghl'));
        Volt::mount([
            config('livewire.view_path', resource_path('views/livewire')),
        ]);

        HandleCors::skipWhen(static fn (Request $request): bool => $request->is('api/widgets/validate'));

        Event::listen(CommandStarting::class, function (CommandStarting $event): void {
            if (app()->runningUnitTests() || app()->environment('testing')) {
                return;
            }

            DestructiveDatabaseCommandGuard::assertNotRefused($event->command);
        });

        Gate::define('viewPulse', function ($user = null): bool {
            if (app()->environment(['local', 'staging'])) {
                return true;
            }

            return app()->environment('production');
        });
    }
}
