<?php

use App\Providers\AppServiceProvider;
use App\Providers\Filament\DashboardPanelProvider;
use App\Providers\FortifyServiceProvider;
use App\Providers\TelescopeServiceProvider;

return [
    AppServiceProvider::class,
    DashboardPanelProvider::class,
    FortifyServiceProvider::class,
    TelescopeServiceProvider::class,
];
