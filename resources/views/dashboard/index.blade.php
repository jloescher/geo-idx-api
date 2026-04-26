<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>GeoIDX Dashboard</title>
    @vite(['resources/css/app.css', 'resources/js/app.js'])
</head>
<body class="min-h-screen bg-slate-950 text-slate-100 antialiased">
@php
    $widgetCards = [
        ['label' => 'Search Widget', 'slug' => 'search-bar', 'description' => 'Capture high-intent leads from neighborhood + criteria search.', 'preview' => 'Search + location filters'],
        ['label' => 'Property Cards', 'slug' => 'listing-cards', 'description' => 'Show teaser listing cards that naturally funnel into lead capture.', 'preview' => 'Grid cards + teaser CTAs'],
        ['label' => 'Property Detail', 'slug' => 'property-detail', 'description' => 'Deliver rich listing detail pages with conversion checkpoints.', 'preview' => 'Single listing deep view'],
        ['label' => 'Map Search', 'slug' => 'map-search', 'description' => 'Let visitors discover homes visually with map-first interactions.', 'preview' => 'Interactive map + pins'],
    ];
    $isMega = $planKey === 'mega';
@endphp
<main
    x-data="{
        toast: '',
        previewWidget: '',
        done: JSON.parse(localStorage.getItem('dashboardChecklistV1') || '{}'),
        copyEmbed(slug) {
            const code = `<script src='{{ $appUrl }}/widgets/idx-loader.js' data-quantyra-site-key='YOUR_SITE_KEY' data-quantyra-widget='${slug}' async></script>`;
            navigator.clipboard.writeText(code);
            this.toast = 'Copied embed code!';
            setTimeout(() => this.toast = '', 2200);
        },
        toggleDone(key) {
            this.done[key] = !this.done[key];
            localStorage.setItem('dashboardChecklistV1', JSON.stringify(this.done));
        },
        completedCount() {
            return Object.values(this.done).filter(Boolean).length;
        }
    }"
    @token-created.window="toast = 'API token created successfully!'; setTimeout(() => toast = '', 2200)"
    @token-revoked.window="toast = 'API token revoked.'; setTimeout(() => toast = '', 2200)"
    class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8"
>
    {{-- Revenue Impact: Premium header establishes trust and steers upgrades early. --}}
    <header class="rounded-3xl border border-white/10 bg-gradient-to-br from-slate-900 to-slate-950 p-6 shadow-2xl shadow-cyan-900/20 sm:p-8">
        <div class="flex flex-col gap-6 md:flex-row md:items-start md:justify-between">
            <div class="min-w-0">
                <p class="text-sm font-semibold uppercase tracking-wide text-cyan-300">GeoIDX Subscriber Dashboard</p>
                <h1 class="mt-2 text-3xl font-bold tracking-tight text-white sm:text-4xl">Welcome back, {{ auth()->user()->name }}</h1>
                <p class="mt-2 max-w-2xl text-sm text-slate-300">Launch widgets faster, monitor subscription health, and manage API access without leaving your dashboard.</p>
                <div class="mt-4 flex flex-wrap items-center gap-3">
                    <span class="inline-flex items-center rounded-full bg-emerald-500/20 px-3 py-1 text-xs font-semibold text-emerald-200 ring-1 ring-emerald-400/30">
                        {{ $plan['label'] ?? 'No plan' }} Plan
                    </span>
                    <span class="inline-flex items-center rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-200 ring-1 ring-white/10">
                        {{ $trialEndsAt ? 'Trial ends '.$trialEndsAt->toFormattedDateString() : 'Auto-renewing subscription' }}
                    </span>
                </div>
            </div>
            <div class="flex flex-wrap items-center gap-3">
                @unless ($isMega)
                    <a href="/#pricing" class="inline-flex min-h-11 items-center rounded-lg bg-emerald-400 px-4 py-2 text-sm font-semibold text-slate-950 shadow hover:bg-emerald-300">
                        Upgrade plan
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

    @if (session('dashboard_status'))
        <div class="mt-6 rounded-xl border border-emerald-400/30 bg-emerald-900/20 px-4 py-3 text-sm text-emerald-100">
            {{ session('dashboard_status') }}
        </div>
    @endif

    {{-- Revenue Impact: Top KPI cards make value obvious and increase retention. --}}
    <section class="mt-8 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
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
            <p class="mt-2 text-xl font-semibold text-white">{{ number_format($widgetInstalledCount) }} / 4</p>
            <p class="mt-1 text-sm text-slate-300">Search, cards, detail, and map widgets ready to deploy.</p>
        </article>
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5 transition hover:-translate-y-0.5 hover:border-cyan-400/30">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Leads this month</p>
            <p class="mt-2 text-xl font-semibold text-white">{{ number_format($leadsThisMonth) }}</p>
            <a href="/#pricing" class="mt-1 inline-flex text-sm font-medium text-cyan-300 hover:text-cyan-200">View leads insights</a>
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

    {{-- Revenue Impact: Widget library is primary activation surface for Smart/Pro users. --}}
    <section class="mt-8 rounded-3xl border border-cyan-400/20 bg-slate-900/70 p-6 shadow-xl shadow-cyan-950/20 sm:p-8">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
                <h2 class="text-2xl font-semibold tracking-tight text-white">Widget Library</h2>
                <p class="mt-1 text-sm text-slate-300">Install search and listing widgets in minutes with one-click embed code copy.</p>
            </div>
            <p class="text-xs text-cyan-200">Loader URL: <span class="font-mono">{{ $appUrl }}/widgets/idx-loader.js</span></p>
        </div>
        <div class="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            @foreach ($widgetCards as $widget)
                <article class="group rounded-2xl border border-white/10 bg-slate-950/70 p-4 transition-all hover:-translate-y-0.5 hover:border-cyan-400/40 hover:shadow-lg hover:shadow-cyan-900/20">
                    <div class="rounded-xl border border-white/10 bg-gradient-to-br from-slate-900 to-slate-800 p-3 text-xs text-slate-300">
                        <p class="font-semibold text-slate-100">{{ $widget['preview'] }}</p>
                        <p class="mt-1">Preview thumbnail</p>
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
                            @click="previewWidget = '{{ $widget['slug'] }}'"
                            class="inline-flex min-h-10 items-center rounded-lg border border-white/20 px-3 py-2 text-xs font-semibold text-slate-200 hover:bg-white/10"
                        >
                            Preview Demo
                        </button>
                    </div>
                </article>
            @endforeach
        </div>
    </section>

    {{-- Revenue Impact: Keeps subscription/billing context visible without leaving dashboard. --}}
    <section class="mt-8 grid gap-6 lg:grid-cols-3">
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
            <h2 class="text-base font-semibold text-white">Subscription Details</h2>
            <dl class="mt-4 space-y-2 text-sm text-slate-300">
                <div class="flex justify-between gap-2"><dt>Plan</dt><dd class="font-semibold text-slate-100">{{ $plan['label'] ?? 'Unknown' }}</dd></div>
                <div class="flex justify-between gap-2"><dt>Status</dt><dd class="font-semibold text-slate-100">{{ $subscription->stripe_status ?? 'N/A' }}</dd></div>
                <div class="flex justify-between gap-2"><dt>Trial ends</dt><dd class="font-semibold text-slate-100">{{ $trialEndsAt?->toDayDateTimeString() ?? 'No trial' }}</dd></div>
            </dl>
            @if (in_array($planKey, ['pro', 'smart'], true))
                <span class="mt-4 inline-flex rounded-full bg-emerald-500/20 px-3 py-1 text-xs font-semibold text-emerald-200 ring-1 ring-emerald-400/30">
                    Unlimited JS Widget usage
                </span>
            @endif
        </article>
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
            <h2 class="text-base font-semibold text-white">My Approved Domains</h2>
            <p class="mt-2 text-sm text-slate-300">Domains currently approved for widget deployment and IDX routing.</p>
            <div class="mt-4 space-y-2 text-xs text-slate-200">
                @forelse ($activeDomains as $domain)
                    <div class="rounded-lg border border-white/10 bg-slate-950/70 px-3 py-2">{{ $domain->domain_slug }}</div>
                @empty
                    <p class="text-slate-400">No approved domains found yet.</p>
                @endforelse
            </div>
            <a href="/#pricing" class="mt-4 inline-flex text-xs font-semibold text-cyan-300 hover:text-cyan-200">Manage site keys and origins</a>
        </article>
        <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-5">
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
                <a href="/#pricing" class="mt-3 inline-flex text-xs font-semibold text-emerald-300 hover:text-emerald-200">Upgrade to Ultra or Mega</a>
            @endunless
        </article>
    </section>

    @if ($hasApiAccess)
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

    {{-- Revenue Impact: Progress checklist drives setup completion and activation. --}}
    <section class="mt-8 rounded-2xl border border-white/10 bg-slate-900/70 p-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 class="text-xl font-semibold text-white">Getting Started Checklist</h2>
            <p class="text-sm text-slate-300"><span x-text="completedCount()"></span>/4 completed</p>
        </div>
        <div class="mt-4 space-y-3">
            @php
                $steps = [
                    ['key' => 'subscription', 'label' => 'Confirm your active subscription status'],
                    ['key' => 'widgets', 'label' => 'Install at least one widget'],
                    ['key' => 'lead-routing', 'label' => 'Connect LeadConnector / GHL routing'],
                    ['key' => 'api', 'label' => 'Generate API token (Ultra/Mega)'],
                ];
            @endphp
            @foreach ($steps as $step)
                <button
                    type="button"
                    @click="toggleDone('{{ $step['key'] }}')"
                    class="flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left transition"
                    :class="done['{{ $step['key'] }}'] ? 'border-emerald-400/40 bg-emerald-900/20' : 'border-white/10 bg-slate-950/70 hover:border-cyan-400/30'"
                >
                    <span class="text-sm text-slate-200">{{ $step['label'] }}</span>
                    <span class="text-xs font-semibold" :class="done['{{ $step['key'] }}'] ? 'text-emerald-200' : 'text-slate-400'" x-text="done['{{ $step['key'] }}'] ? 'Done' : 'Pending'"></span>
                </button>
            @endforeach
        </div>
    </section>

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
            <p class="mt-2 text-sm text-slate-300">Live demo placeholder for <span class="font-semibold text-cyan-200" x-text="previewWidget"></span>. Connect this to sandbox data for interactive preview.</p>
            <div class="mt-4 rounded-xl border border-dashed border-cyan-400/40 bg-slate-950 p-8 text-center text-sm text-slate-400">
                Widget demo canvas area
            </div>
        </div>
    </div>
</main>
</body>
</html>
