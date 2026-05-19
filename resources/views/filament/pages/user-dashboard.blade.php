@vite(['resources/css/app.css', 'resources/js/app.js'])

<x-filament-panels::page>
    <div class="idx-dashboard-shell space-y-4" x-data="window.__createDashboardAlpineState ? window.__createDashboardAlpineState({}) : {}">
        @if (session('dashboard_status'))
            <div class="rounded-xl border border-emerald-400/30 bg-emerald-900/20 px-4 py-3 text-sm text-emerald-100">{{ session('dashboard_status') }}</div>
        @endif

        @if ($activePanel === 'setup')
            @include('dashboard.items.setup')
        @elseif ($activePanel === 'api')
            @include('dashboard.items.api')
        @endif
    </div>
</x-filament-panels::page>
