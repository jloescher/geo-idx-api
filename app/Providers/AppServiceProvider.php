<?php

namespace App\Providers;

use App\Services\Bridge\BridgeRateLimitGuard;
use App\Services\Bridge\BridgeSyncService;
use App\Services\Bridge\HybridReplicaSearchDecision;
use App\Services\Bridge\HybridSearchService;
use App\Services\Bridge\PostgisSearchService;
use App\Services\Geocoding\GoogleGeocodingService;
use App\Services\Mls\MlsMirrorRollingWindow;
use App\Support\DestructiveDatabaseCommandGuard;
use Illuminate\Console\Events\CommandStarting;
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

        /*
         * Octane: stateless collaborators; singletons amortize translator wiring and reuse the
         * same hydrated service graphs without request-scoped mutation.
         */
        $this->app->singleton(MlsMirrorRollingWindow::class);
        $this->app->singleton(BridgeRateLimitGuard::class);
        $this->app->singleton(BridgeSyncService::class);
        $this->app->singleton(PostgisSearchService::class);
        $this->app->singleton(HybridReplicaSearchDecision::class);
        $this->app->singleton(HybridSearchService::class);
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        Volt::mount([
            config('livewire.view_path', resource_path('views/livewire')),
        ]);

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
