@if ($activePanel === 'dashboard')
    @if (! empty($canInviteUsers))
        <section class="mt-6 rounded-2xl border border-cyan-400/25 bg-slate-900/80 p-5 shadow-lg shadow-cyan-900/10">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-cyan-300">Invite a teammate</h2>
            <p class="mt-1 text-sm text-slate-300">Send an email invitation. The recipient completes registration using the same MLS and domain flow as a normal subscriber.</p>
            <form method="POST" action="{{ route('dashboard.invitations.store', [], false) }}" class="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end">
                @csrf
                <div class="min-w-0 flex-1">
                    <label for="invite-email" class="block text-xs font-medium text-slate-300">Email</label>
                    <input
                        id="invite-email"
                        name="email"
                        type="email"
                        value="{{ old('email') }}"
                        required
                        class="mt-1 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-2.5 text-sm text-slate-100 focus:border-cyan-400/60 focus:outline-none focus:ring-2 focus:ring-cyan-400/30"
                        placeholder="colleague@example.com"
                    >
                    @error('email')
                        <p class="mt-1 text-xs font-medium text-red-300">{{ $message }}</p>
                    @enderror
                </div>
                <button
                    type="submit"
                    class="inline-flex shrink-0 items-center justify-center rounded-xl bg-cyan-500 px-4 py-2.5 text-sm font-semibold text-slate-950 hover:bg-cyan-400"
                >
                    Send invitation
                </button>
            </form>
        </section>
    @endif

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
