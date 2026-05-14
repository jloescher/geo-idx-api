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
            window.__createDashboardAlpineState = window.__createDashboardAlpineState || function() {
                return {
                    toast: '',
                };
            };
        </script>
    @endif
    <main x-cloak x-data="window.__createDashboardAlpineState({})"
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
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'domains'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'domains' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">Domains</a>
                    <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'api'], false) }}"
                        class="flex items-center rounded-xl px-3 py-2 {{ $activePanel === 'api' ? 'bg-cyan-500/10 font-semibold text-cyan-100 ring-1 ring-cyan-400/30' : 'text-slate-300 hover:bg-white/5' }}">API</a>
                </nav>
            </aside>
            <div class="min-w-0 md:col-start-2">
                {{-- Revenue Impact: Premium header establishes trust and steers upgrades early. --}}
                <header
                    class="sticky top-0 z-20 shrink-0 rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900 to-slate-950 p-4 shadow-xl shadow-cyan-900/20 sm:p-5 md:fixed md:top-4 md:left-[272px] md:right-6 md:z-40">
                    <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                        <div class="min-w-0">
                            <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">GeoIDX dashboard</p>
                            <h1 class="mt-1 text-2xl font-bold tracking-tight text-white sm:text-3xl">Welcome back,
                                {{ auth()->user()->name }}</h1>
                            <div class="mt-2 flex flex-wrap items-center gap-2">
                                <span
                                    class="inline-flex items-center rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-200 ring-1 ring-white/10">
                                    Domains &amp; API keys
                                </span>
                            </div>
                        </div>
                        <div class="flex flex-wrap items-center gap-3">
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

                    @include('dashboard.items.dashboard-home')

                    @include('dashboard.items.domains')

                    @include('dashboard.items.api')

                    @include('dashboard.items.onboarding')
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
    </main>
    @livewireScripts
</body>

</html>
