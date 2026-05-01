@vite(['resources/css/app.css', 'resources/js/app.js'])

<x-filament-panels::page>
    @php
        $widgetCards = [
            ['label' => 'Search Widget', 'slug' => 'search-bar', 'description' => 'Capture high-intent leads from neighborhood + criteria search.', 'preview' => 'Search + location filters'],
            ['label' => 'Property Cards', 'slug' => 'listing-cards', 'description' => 'Show teaser listing cards that naturally funnel into lead capture.', 'preview' => 'Grid cards + teaser CTAs'],
            ['label' => 'Property Detail', 'slug' => 'property-detail', 'description' => 'Deliver rich listing detail pages with conversion checkpoints.', 'preview' => 'Single listing deep view'],
            ['label' => 'Map Search', 'slug' => 'map-search', 'description' => 'Let visitors discover homes visually with map-first interactions.', 'preview' => 'Interactive map + pins'],
        ];
        $activePanel = (string) request()->query('panel', 'dashboard');
    @endphp

    <div class="idx-dashboard-shell space-y-4" x-data="window.__createDashboardAlpineState ? window.__createDashboardAlpineState({ previewApiKey: @js($widgetPreviewApiKey), widgetValidateUrl: '/dashboard/widget-validate', csrfToken: @js(csrf_token()), appUrl: @js($appUrl), widgetLoaderBaseUrl: @js($widgetLoaderBaseUrl) }) : {}">
        @if (session('dashboard_status'))
            <div class="rounded-xl border border-emerald-400/30 bg-emerald-900/20 px-4 py-3 text-sm text-emerald-100">{{ session('dashboard_status') }}</div>
        @endif

        @if ($activePanel === 'dashboard')
            <div class="rounded-xl border border-cyan-500/25 bg-cyan-950/30 px-4 py-3 text-sm text-cyan-50">
                <span class="font-semibold text-cyan-100">Agent portal</span>
                <span class="text-cyan-200/90"> — saved searches, alerts, marketing tools, and MLS scope (new).</span>
                <a href="/filament-dashboard/agent-portal-overview" class="ml-2 inline-flex font-medium text-white underline decoration-cyan-300/80 underline-offset-2 hover:text-cyan-100">
                    Open overview
                </a>
            </div>
        @endif

        @include('dashboard.items.dashboard-home')
        @include('dashboard.items.onboarding')
        @include('dashboard.items.widgets')
        @include('dashboard.items.domains')
        @include('dashboard.items.api')
        @include('dashboard.items.leads')
    </div>
</x-filament-panels::page>
