<?php

namespace App\Filament\Pages;

use App\Http\Controllers\DashboardController;
use Filament\Pages\Page;
use Filament\Support\Enums\Width;
use Illuminate\Contracts\View\View;

class SubscriberDashboard extends Page
{
    protected string $view = 'filament.pages.subscriber-dashboard';

    protected static string|\BackedEnum|null $navigationIcon = 'heroicon-o-squares-2x2';

    protected static ?string $title = 'Subscriber Dashboard';

    protected static ?string $slug = '/';

    protected static bool $isDiscovered = false;

    protected static bool $shouldRegisterNavigation = false;

    protected Width|string|null $maxContentWidth = Width::Full;

    public function render(): View
    {
        /** @var \Illuminate\View\View $legacyView */
        $legacyView = app(DashboardController::class)->__invoke(request());

        return parent::render()->with($legacyView->getData());
    }
}
