@if ($activePanel === 'api')
    <section class="idx-card mt-8 p-6">
        <h2 class="text-xl font-semibold text-white">API access</h2>
        <p class="mt-2 text-sm text-slate-300">
            Use your API base <span class="font-mono text-cyan-200/90">{{ $apiPublicUrl }}</span> with a dashboard token and a
            <span class="font-semibold text-slate-200">verified</span> domain slug on each call.
        </p>
    </section>

    <livewire:dashboard.api-token-manager />
@endif
