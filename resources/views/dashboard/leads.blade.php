<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>GeoIDX Leads</title>
    @livewireStyles
    @vite(['resources/css/app.css', 'resources/js/app.js'])
</head>
<body class="min-h-screen bg-[#0a0a0a] text-slate-100 antialiased">
<main class="mx-auto max-w-[1400px] px-4 py-8 sm:px-6 lg:px-8">
    <div class="grid gap-6 lg:grid-cols-[240px,1fr]">
        <aside class="rounded-3xl border border-white/10 bg-slate-900/70 p-4 lg:sticky lg:top-6 lg:h-fit">
            <p class="px-2 text-xs font-semibold uppercase tracking-wide text-cyan-300">GEOIDX Dashboard</p>
            <nav class="mt-4 space-y-1 text-sm">
                <a href="{{ route('dashboard.index', [], false) }}" class="flex items-center rounded-xl px-3 py-2 text-slate-300 hover:bg-white/5">Dashboard</a>
                <a href="{{ route('dashboard.index', [], false) }}#widget-library" class="flex items-center rounded-xl px-3 py-2 text-slate-300 hover:bg-white/5">Widgets</a>
                <a href="{{ route('dashboard.leads', [], false) }}" class="flex items-center rounded-xl bg-cyan-500/10 px-3 py-2 font-semibold text-cyan-100 ring-1 ring-cyan-400/30">Leads</a>
                <a href="{{ route('dashboard.index', [], false) }}#approved-domains" class="flex items-center rounded-xl px-3 py-2 text-slate-300 hover:bg-white/5">Domains</a>
                <a href="{{ route('dashboard.index', [], false) }}#api-access" class="flex items-center rounded-xl px-3 py-2 text-slate-300 hover:bg-white/5">API</a>
                <a href="{{ route('marketing.sales', [], false) }}#pricing" class="flex items-center rounded-xl px-3 py-2 text-slate-300 hover:bg-white/5">Billing</a>
            </nav>
        </aside>

        <div class="space-y-6">
            <header class="rounded-3xl border border-white/10 bg-gradient-to-br from-slate-900 to-[#111113] p-6 shadow-xl">
                <h1 class="text-3xl font-semibold text-white">Leads</h1>
                <p class="mt-2 text-sm text-slate-300">Track, filter, and convert high-intent leads from your GeoIDX experiences.</p>
            </header>

            @if (session('dashboard_status'))
                <div class="rounded-xl border border-emerald-400/30 bg-emerald-900/20 px-4 py-3 text-sm text-emerald-100">
                    {{ session('dashboard_status') }}
                </div>
            @endif

            <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Total Leads</p>
                    <p class="mt-2 text-2xl font-semibold text-white">{{ number_format($totalLeads) }}</p>
                </article>
                <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Leads This Month</p>
                    <p class="mt-2 text-2xl font-semibold text-white">{{ number_format($leadsThisMonth) }}</p>
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

            <div class="grid gap-4 xl:grid-cols-2">
                <livewire:dashboard.leads.lead-saved-searches-panel />
                <livewire:dashboard.leads.lead-alert-settings-panel />
            </div>
        </div>
    </div>
</main>
@livewireScripts
</body>
</html>

