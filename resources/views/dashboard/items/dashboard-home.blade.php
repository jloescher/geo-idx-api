@if ($activePanel === 'dashboard')
    <section class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <article class="idx-metric-card transition hover:-translate-y-0.5">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Verified domains</p>
            <p class="mt-2 text-xl font-semibold text-white">{{ number_format($verifiedDomainCount) }}</p>
            <p class="mt-1 text-sm text-slate-300">TXT-verified hostnames you can pair with API tokens.</p>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'domains'], false) }}" class="mt-1 inline-flex text-sm font-medium text-cyan-300 hover:text-cyan-200">Manage domains</a>
        </article>
        <article class="idx-metric-card transition hover:-translate-y-0.5">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">Bearer + domain</p>
            <p class="mt-2 text-lg font-semibold text-white">Required together</p>
            <p class="mt-1 text-sm text-slate-300">Send <span class="font-mono text-xs">Authorization</span> and <span class="font-mono text-xs">X-Domain-Slug</span> (verified) on every MLS/GIS API call.</p>
        </article>
        <article class="idx-metric-card transition hover:-translate-y-0.5">
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-400">API tokens</p>
            <p class="mt-2 text-xl font-semibold text-white">{{ number_format($apiTokens->count()) }}</p>
            <p class="mt-1 text-sm text-slate-300">Create and revoke keys from the API tab.</p>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'api'], false) }}" class="mt-1 inline-flex text-sm font-medium text-cyan-300 hover:text-cyan-200">Open API tab</a>
        </article>
    </section>
@endif
