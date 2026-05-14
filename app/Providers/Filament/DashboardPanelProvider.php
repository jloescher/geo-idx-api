<?php

namespace App\Providers\Filament;

use App\Filament\Pages\UserDashboard;
use Filament\Http\Middleware\Authenticate;
use Filament\Http\Middleware\AuthenticateSession;
use Filament\Http\Middleware\DisableBladeIconComponents;
use Filament\Http\Middleware\DispatchServingFilamentEvent;
use Filament\Navigation\NavigationGroup;
use Filament\Navigation\NavigationItem;
use Filament\Panel;
use Filament\PanelProvider;
use Filament\Support\Colors\Color;
use Illuminate\Cookie\Middleware\AddQueuedCookiesToResponse;
use Illuminate\Cookie\Middleware\EncryptCookies;
use Illuminate\Foundation\Http\Middleware\PreventRequestForgery;
use Illuminate\Routing\Middleware\SubstituteBindings;
use Illuminate\Session\Middleware\StartSession;
use Illuminate\View\Middleware\ShareErrorsFromSession;

class DashboardPanelProvider extends PanelProvider
{
    public function panel(Panel $panel): Panel
    {
        return $panel
            ->default()
            ->id('dashboard')
            ->path('filament-dashboard')
            ->viteTheme('resources/css/filament-dashboard.css')
            ->login()
            ->colors([
                'primary' => Color::Cyan,
                'gray' => Color::Slate,
                'success' => Color::Emerald,
                'warning' => Color::Amber,
                'danger' => Color::Rose,
            ])
            ->discoverResources(in: app_path('Filament/Resources'), for: 'App\Filament\Resources')
            ->pages([
                UserDashboard::class,
            ])
            ->navigationGroups([
                NavigationGroup::make('Dashboard'),
            ])
            ->navigationItems([
                NavigationItem::make('Dashboard')
                    ->group('Dashboard')
                    ->icon('heroicon-o-home')
                    ->url('/filament-dashboard?panel=dashboard')
                    ->isActiveWhen(fn (): bool => request()->query('panel', 'dashboard') === 'dashboard'),
                NavigationItem::make('Onboarding')
                    ->group('Dashboard')
                    ->icon('heroicon-o-rocket-launch')
                    ->url('/filament-dashboard?panel=onboarding')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'onboarding'),
                NavigationItem::make('Domains')
                    ->group('Dashboard')
                    ->icon('heroicon-o-globe-alt')
                    ->url('/filament-dashboard?panel=domains')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'domains'),
                NavigationItem::make('API')
                    ->group('Dashboard')
                    ->icon('heroicon-o-command-line')
                    ->url('/filament-dashboard?panel=api')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'api'),
            ])
            ->discoverWidgets(in: app_path('Filament/Widgets'), for: 'App\Filament\Widgets')
            ->widgets([])
            ->middleware([
                EncryptCookies::class,
                AddQueuedCookiesToResponse::class,
                StartSession::class,
                AuthenticateSession::class,
                ShareErrorsFromSession::class,
                PreventRequestForgery::class,
                SubstituteBindings::class,
                DisableBladeIconComponents::class,
                DispatchServingFilamentEvent::class,
            ])
            ->authMiddleware([
                Authenticate::class,
            ]);
    }
}
