<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>GeoIDX Dashboard</title>
    @livewireStyles
    @vite(['resources/css/app.css', 'resources/js/app.js'])
    <style>
        [x-cloak] {
            display: none !important;
        }
    </style>
</head>

<body class="min-h-screen bg-slate-950 text-slate-100 antialiased">
    @if (app()->runningUnitTests())
        {{-- PHPUnit: @vite is often disabled; provide a minimal Alpine factory so x-data still parses. --}}
        <script>
            window.__createDashboardAlpineState = window.__createDashboardAlpineState || function(boot) {
                return {
                    toast: '',
                    previewWidget: '',
                    previewLoading: false,
                    previewError: '',
                    previewApiKey: String(boot.previewApiKey ?? ''),
                    widgetValidateUrl: String(boot.widgetValidateUrl ?? ''),
                    csrfToken: String(boot.csrfToken ?? ''),
                    appUrl: String(boot.appUrl ?? ''),
                    widgetLoaderBaseUrl: String(boot.widgetLoaderBaseUrl ?? boot.appUrl ?? ''),
                    resolveWidgetType(slug) {
                        const map = {
                            'search-bar': 'search',
                            'listing-cards': 'community',
                            'property-detail': 'property',
                            'map-search': 'map'
                        };
                        return map[slug] || 'search';
                    },
                    init() {},
                    async validatePreviewContext() {},
                    ensureLoaderScript() {
                        return Promise.resolve();
                    },
                    async mountPreview() {},
                    async openPreview() {},
                    copyEmbed() {},
                };
            };
        </script>
    @endif
    @php
        $widgetCards = [
            [
                'label' => 'Search Widget',
                'slug' => 'search-bar',
                'description' => 'Capture high-intent leads from neighborhood + criteria search.',
                'preview' => 'Search + location filters',
            ],
            [
                'label' => 'Property Cards',
                'slug' => 'listing-cards',
                'description' => 'Show teaser listing cards that naturally funnel into lead capture.',
                'preview' => 'Grid cards + teaser CTAs',
            ],
            [
                'label' => 'Property Detail',
                'slug' => 'property-detail',
                'description' => 'Deliver rich listing detail pages with conversion checkpoints.',
                'preview' => 'Single listing deep view',
            ],
            [
                'label' => 'Map Search',
                'slug' => 'map-search',
                'description' => 'Let visitors discover homes visually with map-first interactions.',
                'preview' => 'Interactive map + pins',
            ],
        ];
        $isMega = $planKey === 'mega';
        $dashboardAlpineBoot = [
            'previewApiKey' => $widgetPreviewApiKey,
            'widgetValidateUrl' => '/dashboard/widget-validate',
            'csrfToken' => csrf_token(),
            'appUrl' => $appUrl,
            'widgetLoaderBaseUrl' => $widgetLoaderBaseUrl,
        ];
    @endphp
    <main x-cloak x-data="window.__createDashboardAlpineState(@js($dashboardAlpineBoot))"
        @token-created.window="toast = 'API token created successfully!'; setTimeout(() => toast = '', 2200)"
        @token-revoked.window="toast = 'API token revoked.'; setTimeout(() => toast = '', 2200)"
        class="min-h-screen px-3 py-4 sm:px-4 sm:py-6 lg:px-6">
        <div class="grid gap-4 md:grid-cols-[240px_minmax(0,1fr)] md:items-start">
            <aside
                class="rounded-2xl border border-white/10 bg-slate-900/75 p-3 md:fixed md:top-4 md:left-4 md:z-30 md:h-[calc(100vh-2rem)] md:w-[240px] md:overflow-hidden">
                <p class="px-2 text-xs font-semibold uppercase tracking-wide text-cyan-300">GEOIDX Dashboard</p>
                <nav class="mt-4 space-y-1 text-sm">
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'dashboard'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'dashboard' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Dashboard</a>
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'onboarding'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'onboarding' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Onboarding</a>
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'widgets'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'widgets' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Widgets</a>
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'leads'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'leads' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Leads</a>
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'domains'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'domains' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Domains</a>
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'api'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'api' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">API</a>
                    <a wire:navigate href="{{ route('marketing.sales', [], false) }}#pricing"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'billing' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Billing</a>
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'settings'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'settings' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Settings</a>
                </nav>
            </aside>
            <div class="min-w-0 md:col-start-2">
                {{-- Revenue Impact: Premium header establishes trust and steers upgrades early. --}}
                <header
                    class="sticky top-0 z-20 shrink-0 rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900 to-slate-950 p-4 shadow-xl shadow-cyan-900/20 sm:p-5 md:fixed md:top-4 md:left-[272px] md:right-6 md:z-40">
                    <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                        <div class="min-w-0">
                            <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">GeoIDX Subscriber
                                Dashboard</p>
                            <h1 class="mt-1 text-2xl font-bold tracking-tight text-white sm:text-3xl">Welcome back,
                                {{ auth()->user()->name }}</h1>
                            <div class="mt-2 flex flex-wrap items-center gap-2">
                                <span
                                    class="inline-flex items-center rounded-full bg-emerald-500/20 px-3 py-1 text-xs font-semibold text-emerald-200 ring-1 ring-emerald-400/30">
                                    {{ $plan['label'] ?? 'No plan' }} Plan
                                </span>
                                <span
                                    class="inline-flex items-center rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-200 ring-1 ring-white/10">
                                    {{ $subscription?->valid() ? 'Paid subscription active' : 'Paid subscription required for live data' }}
                                </span>
                            </div>
                        </div>
                        <div class="flex flex-wrap items-center gap-3">
                            @unless ($isMega)
                                <a href="{{ route('marketing.sales', [], false) }}#pricing"
                                    class="inline-flex min-h-11 items-center rounded-lg bg-emerald-400 px-4 py-2 text-sm font-semibold text-slate-950 shadow hover:bg-emerald-300">
                                    Manage subscription
                                </a>
                            @endunless
                            <form method="POST" action="{{ route('logout', [], false) }}">
                                @csrf
                                <button type="submit"
                                    class="inline-flex min-h-11 items-center rounded-lg border border-rose-400/40 px-4 py-2 text-sm font-semibold text-rose-200 hover:bg-rose-500/10">
                                    Logout
                                </button>
                            </form>
                        </div>
                    </div>
                </header>

                <div class="mt-4 md:pt-30">
                    @if (session('dashboard_status'))
                        <div
                            class="mt-6 rounded-xl border border-emerald-400/30 bg-emerald-900/20 px-4 py-3 text-sm text-emerald-100">
                            {{ session('dashboard_status') }}
                        </div>
                    @endif
                    @if ($errors->has('billing'))
                        <div
                            class="mt-6 rounded-xl border border-rose-400/30 bg-rose-900/20 px-4 py-3 text-sm text-rose-100">
                            {{ $errors->first('billing') }}
                        </div>
                    @endif

                    @include('dashboard.items.dashboard-home')

                    @include('dashboard.items.widgets')

                    @include('dashboard.items.domains')

                    @include('dashboard.items.api')

                    @include('dashboard.items.onboarding')

                    @include('dashboard.items.leads')

                </div>
            </div>
        </div>

        {{-- Revenue Impact: Persistent support path lowers setup abandonment. --}}
        <a href="mailto:support@quantyralabs.cc"
            class="fixed bottom-5 right-5 inline-flex items-center gap-2 rounded-full bg-cyan-500 px-4 py-3 text-sm font-semibold text-slate-950 shadow-lg shadow-cyan-900/30 hover:bg-cyan-400">
            Need help?
        </a>

        <div x-show="toast" x-transition style="display: none;"
            class="fixed bottom-5 left-1/2 -translate-x-1/2 rounded-full bg-emerald-500 px-4 py-2 text-sm font-semibold text-slate-950 shadow"
            x-text="toast"></div>

        <div x-show="previewWidget !== ''" x-transition x-cloak style="display: none;"
            @click.self="previewWidget = ''"
            class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4">
            <div class="w-full max-w-2xl rounded-2xl border border-white/10 bg-slate-900 p-6">
                <div class="flex items-center justify-between">
                    <h3 class="text-lg font-semibold text-white">Widget Preview</h3>
                    <button type="button" @click="previewWidget = ''"
                        class="rounded-md border border-white/20 px-3 py-1 text-sm text-slate-200 hover:bg-white/10">Close</button>
                </div>
                <p class="mt-2 text-sm text-slate-300">Proxy-backed preview for <span
                        class="font-semibold text-cyan-200" x-text="previewWidget"></span>. This respects token + host
                    validation rules.</p>
                <label class="mt-4 block text-xs font-semibold uppercase tracking-wide text-slate-400">Widget site key
                    for preview</label>
                <input type="text" x-model="previewApiKey"
                    class="mt-2 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-cyan-400 focus:outline-none"
                    placeholder="Paste your qh_… widget site key">
                <p class="mt-2 text-[11px] text-slate-400">
                    Your <span class="font-semibold text-slate-300">site key</span> (above) must match <span
                        class="font-mono text-slate-200">?token=</span> on the page you embed.
                    Origins are checked against <span class="font-semibold text-slate-300">My Approved Domains</span>.
                    Optional: connect GoHighLevel for marketplace + extra install tooling.
                    Pasted overrides are saved in this browser only.
                </p>
                <button type="button" @click="mountPreview()"
                    class="mt-3 inline-flex min-h-10 items-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400">
                    Refresh Preview
                </button>
                <p x-show="previewLoading" class="mt-3 text-xs text-slate-400">Loading preview…</p>
                <p x-show="previewError"
                    class="mt-3 rounded-lg border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-200"
                    x-text="previewError"></p>
                <div
                    class="mt-4 rounded-xl border border-dashed border-cyan-400/40 bg-slate-950 p-4 text-sm text-slate-300">
                    <div data-quantyragidx-footer="true" class="sr-only">Compliance footer anchor</div>
                    <div x-ref="previewCanvas"></div>
                </div>
            </div>
        </div>
    </main>
    @livewireScripts
</body>

</html>
