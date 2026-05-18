<?php

namespace App\Filament\Pages;

use App\Services\Dashboard\DashboardViewData;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;
use Illuminate\Contracts\View\View;

class UserDashboard extends Page
{
    protected string $view = 'filament.pages.user-dashboard';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-squares-2x2';

    protected static ?string $title = 'User Dashboard';

    protected static ?string $slug = '/';

    protected static bool $isDiscovered = false;

    protected static bool $shouldRegisterNavigation = false;

    protected Width|string|null $maxContentWidth = Width::Full;

    public function render(): View
    {
        return parent::render()->with(
            app(DashboardViewData::class)->forRequest(request()),
        );
    }
}
