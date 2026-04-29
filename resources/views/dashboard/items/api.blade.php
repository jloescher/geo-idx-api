@if ($activePanel === 'api' && $hasApiAccess)
    <section class="idx-card mt-8 p-6">
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
