@vite(['resources/css/app.css', 'resources/js/app.js'])

<x-filament-panels::page>
    @php
        $activePanel = (string) request()->query('panel', 'dashboard');
    @endphp

    <div class="idx-dashboard-shell space-y-4" x-data="window.__createDashboardAlpineState ? window.__createDashboardAlpineState({}) : {}">
        @if (session('dashboard_status'))
            <div class="rounded-xl border border-emerald-400/30 bg-emerald-900/20 px-4 py-3 text-sm text-emerald-100">{{ session('dashboard_status') }}</div>
        @endif

        @include('dashboard.items.dashboard-home')
        @include('dashboard.items.onboarding')
        @include('dashboard.items.domains')
        @include('dashboard.items.api')
    </div>
</x-filament-panels::page>
