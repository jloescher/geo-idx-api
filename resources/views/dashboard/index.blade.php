<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>GeoIDX Dashboard</title>
    @livewireStyles
    @vite(['resources/css/app.css', 'resources/js/app.js'])
    <style>[x-cloak]{display:none!important;}</style>
</head>
<body class="min-h-screen bg-slate-950 text-slate-100 antialiased">
@if (app()->runningUnitTests())
    {{-- PHPUnit: @vite is often disabled; provide a minimal Alpine factory so x-data still parses. --}}
    <script>
        window.__createDashboardAlpineState = window.__createDashboardAlpineState || function (boot) {
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
                    const map = { 'search-bar': 'search', 'listing-cards': 'community', 'property-detail': 'property', 'map-search': 'map' };
                    return map[slug] || 'search';
                },
                init() {},
                async validatePreviewContext() {},
                ensureLoaderScript() { return Promise.resolve(); },
                async mountPreview() {},
                async openPreview() {},
                copyEmbed() {},
            };
        };
    </script>
@endif
@php
    $widgetCards = [
        ['label' => 'Search Widget', 'slug' => 'search-bar', 'description' => 'Capture high-intent leads from neighborhood + criteria search.', 'preview' => 'Search + location filters'],
        ['label' => 'Property Cards', 'slug' => 'listing-cards', 'description' => 'Show teaser listing cards that naturally funnel into lead capture.', 'preview' => 'Grid cards + teaser CTAs'],
        ['label' => 'Property Detail', 'slug' => 'property-detail', 'description' => 'Deliver rich listing detail pages with conversion checkpoints.', 'preview' => 'Single listing deep view'],
        ['label' => 'Map Search', 'slug' => 'map-search', 'description' => 'Let visitors discover homes visually with map-first interactions.', 'preview' => 'Interactive map + pins'],
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
<main
    x-cloak
    x-data="window.__createDashboardAlpineState(@js($dashboardAlpineBoot))"
    @token-created.window="toast = 'API token created successfully!'; setTimeout(() => toast = '', 2200)"
    @token-revoked.window="toast = 'API token revoked.'; setTimeout(() => toast = '', 2200)"
    class="min-h-screen px-3 py-4 sm:px-4 sm:py-6 lg:px-6"
>
    <div class="grid gap-4 md:grid-cols-[240px_minmax(0,1fr)] md:items-start">
    <aside class="rounded-2xl border border-white/10 bg-slate-900/75 p-3 md:fixed md:top-4 md:left-4 md:z-30 md:h-[calc(100vh-2rem)] md:w-[240px] md:overflow-hidden">
        <p class="px-2 text-xs font-semibold uppercase tracking-wide text-cyan-300">GEOIDX Dashboard</p>
        <nav class="mt-4 space-y-1 text-sm">
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'dashboard'], false) }}" class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'dashboard' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Dashboard</a>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'widgets'], false) }}" class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'widgets' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Widgets</a>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'leads'], false) }}" class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'leads' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Leads</a>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'domains'], false) }}" class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'domains' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Domains</a>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'api'], false) }}" class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'api' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">API</a>
            <a wire:navigate href="{{ route('marketing.sales', [], false) }}#pricing" class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'billing' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Billing</a>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'settings'], false) }}" class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'settings' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Settings</a>
        </nav>
    </aside>
    <div class="min-w-0 md:col-start-2">
    {{-- Revenue Impact: Premium header establishes trust and steers upgrades early. --}}
    <header class="sticky top-0 z-20 shrink-0 rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900 to-slate-950 p-4 shadow-xl shadow-cyan-900/20 sm:p-5 md:fixed md:top-4 md:left-[272px] md:right-6 md:z-40">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="min-w-0">
                <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">GeoIDX Subscriber Dashboard</p>
                <h1 class="mt-1 text-2xl font-bold tracking-tight text-white sm:text-3xl">Welcome back, {{ auth()->user()->name }}</h1>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                    <span class="inline-flex items-center rounded-full bg-emerald-500/20 px-3 py-1 text-xs font-semibold text-emerald-200 ring-1 ring-emerald-400/30">
                        {{ $plan['label'] ?? 'No plan' }} Plan
                    </span>
                    <span class="inline-flex items-center rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-200 ring-1 ring-white/10">
                        {{ $subscription?->valid() ? 'Paid subscription active' : 'Paid subscription required for live data' }}
                    </span>
                </div>
            </div>
            <div class="flex flex-wrap items-center gap-3">
                @unless ($isMega)
                    <a href="{{ route('marketing.sales', [], false) }}#pricing" class="inline-flex min-h-11 items-center rounded-lg bg-emerald-400 px-4 py-2 text-sm font-semibold text-slate-950 shadow hover:bg-emerald-300">
                        Manage subscription
                    </a>
                @endunless
                <form method="POST" action="{{ route('logout', [], false) }}">
                    @csrf
                    <button type="submit" class="inline-flex min-h-11 items-center rounded-lg border border-rose-400/40 px-4 py-2 text-sm font-semibold text-rose-200 hover:bg-rose-500/10">
                        Logout
                    </button>
                </form>
            </div>
        </div>
    </header>

    <div class="mt-4 md:pt-40">
    @if (session('dashboard_status'))
        <div class="mt-6 rounded-xl border border-emerald-400/30 bg-emerald-900/20 px-4 py-3 text-sm text-emerald-100">
            {{ session('dashboard_status') }}
        </div>
    @endif
    @if ($errors->has('billing'))
        <div class="mt-6 rounded-xl border border-rose-400/30 bg-rose-900/20 px-4 py-3 text-sm text-rose-100">
            {{ $errors->first('billing') }}
        </div>
    @endif

    @if ($activePanel === 'dashboard')
    {{-- Revenue Impact: Top KPI cards make value obvious and increase retention. --}}
    <section class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5 transition hover:-translate-y-0.5 hover:border-cyan-400/30">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Subscription status</p>
            <p class="mt-2 text-xl font-semibold text-white">{{ $subscription?->valid() ? 'Active' : 'Needs attention' }}</p>
            <p class="mt-1 text-sm text-slate-300">{{ $subscription?->stripe_status ? ucfirst($subscription->stripe_status) : 'No active subscription' }}</p>
            @if ($trialProgressPercent !== null)
                <div class="mt-3 h-2 rounded-full bg-slate-800">
                    <div class="h-2 rounded-full bg-cyan-400 transition-all" style="width: {{ $trialProgressPercent }}%"></div>
                </div>
                <p class="mt-2 text-xs text-slate-400">Trial progress: {{ $trialProgressPercent }}%</p>
            @endif
        </article>
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5 transition hover:-translate-y-0.5 hover:border-cyan-400/30">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Widgets installed</p>
            <p class="mt-2 text-xl font-semibold text-white">{{ number_format($widgetInstalledCount) }}</p>
            <p class="mt-1 text-sm text-slate-300">GHL marketplace installs plus direct site keys on approved domains.</p>
        </article>
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5 transition hover:-translate-y-0.5 hover:border-cyan-400/30">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Leads this month</p>
            @if ($leadsMetricAvailable)
                <p class="mt-2 text-xl font-semibold text-white">{{ number_format((int) $leadsThisMonth) }}</p>
            @else
                <p class="mt-2 text-xl font-semibold text-slate-300">Not available</p>
                <p class="mt-1 text-xs text-slate-400">Provide a valid widget site key to scope lead telemetry.</p>
            @endif
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'leads'], false) }}" class="mt-1 inline-flex text-sm font-medium text-cyan-300 hover:text-cyan-200">Open Leads dashboard</a>
        </article>
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5 transition hover:-translate-y-0.5 hover:border-cyan-400/30">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">API access</p>
            @if ($hasApiAccess)
                <p class="mt-2 text-xl font-semibold text-white">Enabled</p>
                <p class="mt-1 text-sm text-slate-300">Requests: {{ number_format($apiRequestCount ?? 0) }}</p>
            @else
                <p class="mt-2 text-lg font-semibold text-amber-200">Upgrade required</p>
                <p class="mt-1 text-sm text-slate-300">Upgrade to Ultra/Mega for full API access.</p>
                <a href="/#pricing" class="mt-2 inline-flex text-sm font-medium text-emerald-300 hover:text-emerald-200">Upgrade to API plan</a>
            @endif
        </article>
    </section>
    @endif

    @if ($activePanel === 'widgets')
    {{-- Revenue Impact: Widget library is primary activation surface for Smart/Pro users. --}}
    <section id="widget-library" class="mt-6 rounded-2xl border border-cyan-400/20 bg-slate-900/70 p-5 shadow-xl shadow-cyan-950/20 sm:p-6">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
                <h2 class="text-2xl font-semibold tracking-tight text-white">Widget Library</h2>
                <p class="mt-1 text-sm text-slate-300">Install search and listing widgets in minutes with one-click embed code copy.</p>
            </div>
            <p class="text-xs text-cyan-200">Loader URL: <span class="font-mono">{{ $widgetLoaderBaseUrl }}/widget/loader.js</span></p>
        </div>
        @if ($hasWidgetAccess && $widgetPreviewApiKey !== '')
            <div class="mt-4 rounded-xl border border-cyan-400/25 bg-slate-950/60 p-4">
                <p class="text-sm font-semibold text-cyan-100">Your widget site key</p>
                <p class="mt-1 text-xs text-slate-400">Put this in <span class="font-mono text-slate-200">?token=</span> on your site. Allowed origins are the hostnames under <span class="font-semibold text-slate-300">My Approved Domains</span>. GoHighLevel marketplace is optional.</p>
                <div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
                    <code class="flex-1 break-all rounded-lg border border-white/15 bg-slate-900 px-3 py-2 font-mono text-xs text-slate-100" x-text="previewApiKey"></code>
                    <button
                        type="button"
                        @click="navigator.clipboard.writeText(previewApiKey); toast = 'Site key copied'; setTimeout(() => toast = '', 2200)"
                        class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg border border-cyan-400/40 px-3 py-2 text-xs font-semibold text-cyan-100 hover:bg-cyan-500/10"
                    >
                        Copy key
                    </button>
                </div>
            </div>
        @endif
        <div class="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            @foreach ($widgetCards as $widget)
                <article class="group rounded-2xl border border-white/10 bg-slate-950/70 p-4 transition-all hover:-translate-y-0.5 hover:border-cyan-400/40 hover:shadow-lg hover:shadow-cyan-900/20">
                    <div class="rounded-xl border border-white/10 bg-gradient-to-br from-slate-900 to-slate-800 p-3 text-xs text-slate-300">
                        <p class="font-semibold text-slate-100">{{ $widget['preview'] }}</p>
                        <p class="mt-1">Use Preview Demo for a live proxy-backed render.</p>
                    </div>
                    <h3 class="mt-3 text-base font-semibold text-white">{{ $widget['label'] }}</h3>
                    <p class="mt-1 text-sm text-slate-300">{{ $widget['description'] }}</p>
                    <div class="mt-4 flex flex-wrap gap-2">
                        <button
                            type="button"
                            @click="copyEmbed('{{ $widget['slug'] }}')"
                            class="inline-flex min-h-10 items-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400"
                        >
                            Copy Embed Code
                        </button>
                        <button
                            type="button"
                            @click="openPreview('{{ $widget['slug'] }}')"
                            class="inline-flex min-h-10 items-center rounded-lg border border-white/20 px-3 py-2 text-xs font-semibold text-slate-200 hover:bg-white/10"
                        >
                            Preview Demo
                        </button>
                    </div>
                </article>
            @endforeach
        </div>
    </section>

    @if ($hasWidgetAccess)
        <section class="mt-8 rounded-3xl border border-violet-400/25 bg-slate-900/70 p-6 shadow-xl sm:p-8">
            <h2 class="text-xl font-semibold tracking-tight text-white">Widget appearance</h2>
            <p class="mt-1 text-sm text-slate-300">Set colors once for every embed (search, map, property, footer). Per-page query parameters on the loader still override these when you need a one-off.</p>
            <form method="POST" action="{{ route('dashboard.widget-appearance', [], false) }}" class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                @csrf
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Primary
                    <input type="color" name="primary" value="{{ old('primary', $widgetPaletteForm['primary'] ?? '#2563eb') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Secondary
                    <input type="color" name="secondary" value="{{ old('secondary', $widgetPaletteForm['secondary'] ?? '#1e40af') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Accent (optional)
                    <input type="color" name="accent" value="{{ old('accent', $widgetPaletteForm['accent'] ?? '#10b981') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Text
                    <input type="color" name="text" value="{{ old('text', $widgetPaletteForm['text'] ?? '#0f172a') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Background
                    <input type="color" name="background" value="{{ old('background', $widgetPaletteForm['background'] ?? '#ffffff') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Theme
                    <select name="theme" class="mt-2 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100">
                        <option value="light" @selected(old('theme', $widgetPaletteForm['theme'] ?? 'light') === 'light')>Light</option>
                        <option value="dark" @selected(old('theme', $widgetPaletteForm['theme'] ?? 'light') === 'dark')>Dark</option>
                    </select>
                </label>
                <div class="sm:col-span-2 lg:col-span-3">
                    <button type="submit" class="inline-flex min-h-11 items-center rounded-lg bg-violet-500 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-400">
                        Save widget colors
                    </button>
                </div>
            </form>
            @if ($errors->any())
                <div class="mt-4 rounded-lg border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-100">
                    {{ $errors->first() }}
                </div>
            @endif
        </section>
    @endif
    @endif

    @if ($activePanel === 'dashboard' || $activePanel === 'domains' || $activePanel === 'api')
    {{-- Revenue Impact: Keeps subscription/billing context visible without leaving dashboard. --}}
    <section class="mt-8 grid gap-6 lg:grid-cols-3">
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
            <h2 class="text-base font-semibold text-white">Subscription Details</h2>
            <dl class="mt-4 space-y-2 text-sm text-slate-300">
                <div class="flex justify-between gap-2"><dt>Plan</dt><dd class="font-semibold text-slate-100">{{ $plan['label'] ?? 'Unknown' }}</dd></div>
                <div class="flex justify-between gap-2"><dt>Status</dt><dd class="font-semibold text-slate-100">{{ $subscription->stripe_status ?? 'N/A' }}</dd></div>
                <div class="flex justify-between gap-2"><dt>Live data access</dt><dd class="font-semibold text-slate-100">{{ $subscription?->valid() ? 'Paid + Active' : 'Blocked until paid' }}</dd></div>
            </dl>
            @if (in_array($planKey, ['pro', 'smart'], true))
                <span class="mt-4 inline-flex rounded-full bg-emerald-500/20 px-3 py-1 text-xs font-semibold text-emerald-200 ring-1 ring-emerald-400/30">
                    Unlimited JS Widget usage
                </span>
            @endif
        </article>
        <article id="approved-domains" class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
            <h2 class="text-base font-semibold text-white">My Approved Domains</h2>
            <p class="mt-2 text-sm text-slate-300">Add hostnames for non-GHL website embeds and IDX routing. Widget access requires TXT verification or LeadConnector domain attachment.</p>
            <p class="mt-2 text-xs text-slate-400">
                @if (is_int($domainLimit))
                    Plan usage: {{ $activeDomains->count() }} / {{ $domainLimit }} domains
                @else
                    Plan usage: {{ $activeDomains->count() }} domains (unlimited on current plan)
                @endif
            </p>
            @if ($planKey === 'pro')
                <p class="mt-1 text-xs text-slate-400">Pro includes 1 domain. Additional domains are $39/mo each.</p>
            @endif
            <form method="POST" action="{{ route('dashboard.domains.store', [], false) }}" class="mt-4 space-y-2">
                @csrf
                <label for="dashboard-domain-slug" class="text-xs font-semibold uppercase tracking-wide text-slate-400">Add domain</label>
                <div class="flex flex-col gap-2 sm:flex-row">
                    <input
                        id="dashboard-domain-slug"
                        name="domain_slug"
                        type="text"
                        value="{{ old('domain_slug') }}"
                        placeholder="example.com"
                        class="w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none"
                        required
                        @disabled($domainLimitReached)
                    >
                    <button type="submit" @disabled($domainLimitReached) class="inline-flex min-h-10 items-center justify-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400 disabled:cursor-not-allowed disabled:opacity-50">
                        Add Domain
                    </button>
                </div>
                @if ($domainLimitReached)
                    <p class="rounded-md border border-amber-400/30 bg-amber-900/20 px-3 py-2 text-xs text-amber-100">
                        Domain limit reached for your plan. Remove a domain, add a paid domain slot below, or upgrade.
                    </p>
                @endif
                @if ($canPurchaseExtraDomainSlots)
                    <div class="mt-3 rounded-lg border border-cyan-400/25 bg-slate-950/60 p-3">
                        <p class="text-xs font-semibold text-cyan-100">Add domain to subscription</p>
                        <p class="mt-1 text-[11px] text-slate-400">Adds one extra approved-hostname slot at {{ config('billing.addons.extra_domain.monthly_display', '$39/mo') }} (Stripe bills with your subscription).</p>
                        <form method="POST" action="{{ route('dashboard.billing.extra-domain', [], false) }}" class="mt-2">
                            @csrf
                            <button type="submit" class="inline-flex min-h-10 items-center justify-center rounded-lg border border-cyan-400/50 bg-cyan-500/10 px-3 py-2 text-xs font-semibold text-cyan-100 hover:bg-cyan-500/20">
                                Add domain slot to subscription
                            </button>
                        </form>
                    </div>
                @endif
                @error('domain_slug')
                    <p class="rounded-md border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-200">{{ $message }}</p>
                @enderror
                <p class="text-xs text-slate-400">Use hostname only. Example: <span class="font-mono">searchtampabayhouses.com</span></p>
            </form>
            <div class="mt-4 space-y-2 text-xs text-slate-200">
                @forelse ($activeDomains as $domain)
                    <div class="space-y-2 rounded-lg border border-white/10 bg-slate-950/70 px-3 py-2">
                        <div class="flex items-center justify-between gap-3">
                            <span class="truncate">{{ $domain->domain_slug }}</span>
                            <span class="rounded-full px-2 py-1 text-[10px] font-semibold {{ in_array((string) $domain->verification_status, ['verified','verified_ghl'], true) ? 'bg-emerald-500/20 text-emerald-200' : 'bg-amber-500/20 text-amber-200' }}">
                                {{ strtoupper((string) $domain->verification_status) }}
                            </span>
                        </div>
                        @if (! in_array((string) $domain->verification_status, ['verified','verified_ghl'], true))
                            <div class="rounded-md border border-cyan-400/25 bg-slate-900/60 p-2 text-[11px] text-slate-300">
                                <p>TXT Name: <span class="font-mono">{{ $domain->txt_verification_name ?: '_geoidx.'.$domain->domain_slug }}</span></p>
                                <p class="mt-1">TXT Value: <span class="font-mono">{{ $domain->txt_verification_value ?: 'Pending challenge' }}</span></p>
                            </div>
                        @endif
                        <div class="flex flex-wrap gap-2">
                            <form method="POST" action="{{ route('dashboard.domains.verify-txt', ['domain' => $domain->id], false) }}">
                                @csrf
                                <button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-cyan-400/40 px-2 py-1 text-[11px] font-semibold text-cyan-200 hover:bg-cyan-500/10">Verify TXT</button>
                            </form>
                            <form method="POST" action="{{ route('dashboard.domains.verify-ghl', ['domain' => $domain->id], false) }}">
                                @csrf
                                <button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-indigo-400/40 px-2 py-1 text-[11px] font-semibold text-indigo-200 hover:bg-indigo-500/10">Verify via GHL</button>
                            </form>
                            <form method="POST" action="{{ route('dashboard.domains.destroy', ['domain' => $domain->id], false) }}" onsubmit="return confirm('Remove this domain from approved hostnames?');">
                                @csrf
                                @method('DELETE')
                                <button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-rose-400/40 px-2 py-1 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/10">Remove</button>
                            </form>
                        </div>
                    </div>
                @empty
                    <p class="text-slate-400">No approved domains found yet.</p>
                @endforelse
            </div>
            <a href="{{ route('leadconnector.install', [], false) }}" class="mt-4 inline-flex text-xs font-semibold text-cyan-300 hover:text-cyan-200">Manage GHL site keys and origins</a>
        </article>
        <article id="api-access" class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
            <h2 class="text-base font-semibold text-white">API Access</h2>
            <p class="mt-2 text-sm text-slate-300">
                @if ($hasApiAccess)
                    Full API access is enabled for this account.
                @else
                    Smart/Pro keep things simple. Upgrade when you need full API access and overage billing controls.
                @endif
            </p>
            <p class="mt-3 rounded-lg bg-slate-950/70 px-3 py-2 text-xs font-mono text-slate-200">{{ $apiPublicUrl }}/api/v1</p>
            @unless ($hasApiAccess)
                <a href="{{ route('marketing.sales', [], false) }}#pricing" class="mt-3 inline-flex text-xs font-semibold text-emerald-300 hover:text-emerald-200">Upgrade to Ultra or Mega</a>
            @endunless
        </article>
    </section>
    @endif

    @if ($activePanel === 'api' && $hasApiAccess)
        <section class="mt-8 rounded-2xl border border-white/10 bg-slate-900/70 p-6">
            <h2 class="text-xl font-semibold text-white">API Usage &amp; Billing</h2>
            <div class="mt-4 grid gap-4 sm:grid-cols-3">
                <div class="rounded-xl border border-white/10 bg-slate-950/70 p-4">
                    <p class="text-xs uppercase tracking-wide text-slate-400">API requests (period)</p>
                    <p class="mt-1 text-2xl font-semibold text-white">{{ number_format($apiRequestCount ?? 0) }}</p>
                </div>
                <div class="rounded-xl border border-white/10 bg-slate-950/70 p-4">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Included</p>
                    <p class="mt-1 text-2xl font-semibold text-white">{{ $apiRequestLimit === null ? 'Unlimited' : number_format($apiRequestLimit) }}</p>
                </div>
                <div class="rounded-xl border border-white/10 bg-slate-950/70 p-4">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Overage billing</p>
                    <p class="mt-1 text-sm font-semibold text-white">{{ $apiOverageRate }}</p>
                    <p class="mt-1 text-xs text-slate-400">Overage requests: {{ number_format($apiOverageCount) }}</p>
                </div>
            </div>
        </section>

        <livewire:dashboard.api-token-manager />
    @endif

    @if ($activePanel === 'dashboard' || $activePanel === 'settings')
    {{-- Revenue Impact: Progress checklist drives setup completion and activation. --}}
    <section id="checklist" class="mt-8 rounded-2xl border border-white/10 bg-slate-900/70 p-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 class="text-xl font-semibold text-white">Getting Started Checklist</h2>
            <p class="text-sm text-slate-300">{{ $onboardingCompletedCount }}/{{ count($onboardingSteps) }} completed</p>
        </div>
        <div class="mt-4 space-y-3">
            @foreach ($onboardingSteps as $step)
                <div
                    class="flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left transition {{ $step['done'] ? 'border-emerald-400/40 bg-emerald-900/20' : 'border-white/10 bg-slate-950/70' }}"
                    data-event-name="dashboard_step_{{ $step['key'] }}"
                >
                    <span class="text-sm text-slate-200">{{ $step['label'] }}</span>
                    <span class="text-xs font-semibold {{ $step['done'] ? 'text-emerald-200' : 'text-slate-400' }}">{{ $step['done'] ? 'Done' : 'Pending' }}</span>
                </div>
            @endforeach
        </div>
    </section>
    @endif

    @if ($activePanel === 'leads')
        @if ($leadsEligible)
            <section class="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Total Leads</p>
                    <p class="mt-2 text-2xl font-semibold text-white">{{ number_format($totalLeads) }}</p>
                </article>
                <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Leads This Month</p>
                    <p class="mt-2 text-2xl font-semibold text-white">{{ number_format((int) ($leadsThisMonth ?? 0)) }}</p>
                </article>
                <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Avg Conversion Rate</p>
                    <p class="mt-2 text-2xl font-semibold text-white">{{ number_format($conversionRate, 1) }}%</p>
                </article>
                <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Hot Leads (24h)</p>
                    <p class="mt-2 text-2xl font-semibold text-white">{{ number_format($hotLeads24h) }}</p>
                </article>
            </section>

            <livewire:dashboard.leads.leads-table />

            <div class="mt-4 grid gap-4 xl:grid-cols-2">
                <livewire:dashboard.leads.lead-saved-searches-panel />
                <livewire:dashboard.leads.lead-alert-settings-panel />
            </div>
        @else
            <section class="mt-8 rounded-2xl border border-amber-400/30 bg-amber-900/20 p-5 text-amber-100">
                <h2 class="text-lg font-semibold">Leads access is locked</h2>
                <p class="mt-2 text-sm">Requirements: active paid subscription, eligible plan, and verified MLS membership.</p>
                <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'settings'], false) }}" class="mt-3 inline-flex text-sm font-semibold text-amber-200 underline">Review setup checklist</a>
            </section>
        @endif
    @endif

    </div>
    </div>
    </div>

    {{-- Revenue Impact: Persistent support path lowers setup abandonment. --}}
    <a href="mailto:support@quantyralabs.cc" class="fixed bottom-5 right-5 inline-flex items-center gap-2 rounded-full bg-cyan-500 px-4 py-3 text-sm font-semibold text-slate-950 shadow-lg shadow-cyan-900/30 hover:bg-cyan-400">
        Need help?
    </a>

    <div
        x-show="toast"
        x-transition
        style="display: none;"
        class="fixed bottom-5 left-1/2 -translate-x-1/2 rounded-full bg-emerald-500 px-4 py-2 text-sm font-semibold text-slate-950 shadow"
        x-text="toast"
    ></div>

    <div
        x-show="previewWidget !== ''"
        x-transition
        x-cloak
        style="display: none;"
        @click.self="previewWidget = ''"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4"
    >
        <div class="w-full max-w-2xl rounded-2xl border border-white/10 bg-slate-900 p-6">
            <div class="flex items-center justify-between">
                <h3 class="text-lg font-semibold text-white">Widget Preview</h3>
                <button type="button" @click="previewWidget = ''" class="rounded-md border border-white/20 px-3 py-1 text-sm text-slate-200 hover:bg-white/10">Close</button>
            </div>
            <p class="mt-2 text-sm text-slate-300">Proxy-backed preview for <span class="font-semibold text-cyan-200" x-text="previewWidget"></span>. This respects token + host validation rules.</p>
            <label class="mt-4 block text-xs font-semibold uppercase tracking-wide text-slate-400">Widget site key for preview</label>
            <input
                type="text"
                x-model="previewApiKey"
                class="mt-2 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-cyan-400 focus:outline-none"
                placeholder="Paste your qh_… widget site key"
            >
            <p class="mt-2 text-[11px] text-slate-400">
                Your <span class="font-semibold text-slate-300">site key</span> (above) must match <span class="font-mono text-slate-200">?token=</span> on the page you embed.
                Origins are checked against <span class="font-semibold text-slate-300">My Approved Domains</span>. Optional: connect GoHighLevel for marketplace + extra install tooling.
                Pasted overrides are saved in this browser only.
            </p>
            <button
                type="button"
                @click="mountPreview()"
                class="mt-3 inline-flex min-h-10 items-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400"
            >
                Refresh Preview
            </button>
            <p x-show="previewLoading" class="mt-3 text-xs text-slate-400">Loading preview…</p>
            <p x-show="previewError" class="mt-3 rounded-lg border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-200" x-text="previewError"></p>
            <div class="mt-4 rounded-xl border border-dashed border-cyan-400/40 bg-slate-950 p-4 text-sm text-slate-300">
                <div data-quantyragidx-footer="true" class="sr-only">Compliance footer anchor</div>
                <div x-ref="previewCanvas"></div>
            </div>
        </div>
    </div>
</main>
@livewireScripts
</body>
</html>
