@if ($activePanel === 'dashboard')
    <section class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <article class="idx-metric-card transition hover:-translate-y-0.5">
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
        <article class="idx-metric-card transition hover:-translate-y-0.5">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Widgets installed</p>
            <p class="mt-2 text-xl font-semibold text-white">{{ number_format($widgetInstalledCount) }}</p>
            <p class="mt-1 text-sm text-slate-300">GHL marketplace installs plus direct site keys on approved domains.</p>
        </article>
        <article class="idx-metric-card transition hover:-translate-y-0.5">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Leads this month</p>
            @if ($leadsMetricAvailable)
                <p class="mt-2 text-xl font-semibold text-white">{{ number_format((int) $leadsThisMonth) }}</p>
            @else
                <p class="mt-2 text-xl font-semibold text-slate-300">Not available</p>
                <p class="mt-1 text-xs text-slate-400">Provide a valid widget site key to scope lead telemetry.</p>
            @endif
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'leads'], false) }}" class="mt-1 inline-flex text-sm font-medium text-cyan-300 hover:text-cyan-200">Open Leads dashboard</a>
        </article>
        <article class="idx-metric-card transition hover:-translate-y-0.5">
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
