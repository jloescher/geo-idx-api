@if ($activePanel === 'api')
    <section class="idx-card mt-8 p-6">
        <h2 class="text-xl font-semibold text-white">API Keys</h2>
        <p class="mt-2 text-sm text-slate-300">
            Manage all your API tokens. Production and Staging keys use the same verified domain slug (<span class="font-mono text-xs">X-Domain-Slug</span>) but different Bearer tokens.
        </p>
        <p class="mt-1 text-sm text-slate-300">
            API base: <span class="font-mono text-cyan-200/90">{{ $apiPublicUrl }}</span>
        </p>
    </section>

    <livewire:dashboard.api-token-manager />
@endif
