<?php

namespace App\Providers\Filament;

use App\Filament\Pages\SubscriberDashboard;
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
            ->login()
            ->colors([
                'primary' => Color::Cyan,
                'gray' => Color::Slate,
                'success' => Color::Emerald,
                'warning' => Color::Amber,
                'danger' => Color::Rose,
            ])
            ->discoverResources(in: app_path('Filament/Resources'), for: 'App\Filament\Resources')
            ->discoverPages(in: app_path('Filament/Pages'), for: 'App\Filament\Pages')
            ->pages([
                SubscriberDashboard::class,
            ])
            ->navigationGroups([
                NavigationGroup::make('Subscriber Dashboard'),
            ])
            ->navigationItems([
                NavigationItem::make('Dashboard')
                    ->group('Subscriber Dashboard')
                    ->icon('heroicon-o-home')
                    ->url('/filament-dashboard?panel=dashboard')
                    ->isActiveWhen(fn (): bool => request()->query('panel', 'dashboard') === 'dashboard'),
                NavigationItem::make('Onboarding')
                    ->group('Subscriber Dashboard')
                    ->icon('heroicon-o-rocket-launch')
                    ->url('/filament-dashboard?panel=onboarding')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'onboarding'),
                NavigationItem::make('Widgets')
                    ->group('Subscriber Dashboard')
                    ->icon('heroicon-o-squares-plus')
                    ->url('/filament-dashboard?panel=widgets')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'widgets'),
                NavigationItem::make('Domains')
                    ->group('Subscriber Dashboard')
                    ->icon('heroicon-o-globe-alt')
                    ->url('/filament-dashboard?panel=domains')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'domains'),
                NavigationItem::make('API')
                    ->group('Subscriber Dashboard')
                    ->icon('heroicon-o-command-line')
                    ->url('/filament-dashboard?panel=api')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'api'),
                NavigationItem::make('Leads')
                    ->group('Subscriber Dashboard')
                    ->icon('heroicon-o-user-group')
                    ->url('/filament-dashboard?panel=leads')
                    ->isActiveWhen(fn (): bool => request()->query('panel') === 'leads'),
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
