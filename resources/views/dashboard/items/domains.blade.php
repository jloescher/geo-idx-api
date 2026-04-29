@if ($activePanel === 'domains')
    @php
        $verifiedDomainCount = $activeDomains->whereIn('verification_status', ['verified', 'verified_ghl'])->count();
        $pendingDomainCount = max(0, $activeDomains->count() - $verifiedDomainCount);
    @endphp
    <section class="mt-8">
        <div class="rounded-2xl border border-cyan-400/20 bg-slate-900/70 p-6">
            <h2 class="text-xl font-semibold text-white">Domains Command Center</h2>
            <p class="mt-1 text-sm text-slate-300">Manage approved hostnames, buy additional domain capacity, and verify ownership through DNS TXT or GHL attachment.</p>
            <div class="mt-5 grid gap-4 sm:grid-cols-3">
                <div class="rounded-xl border border-white/10 bg-slate-950/70 p-4">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Active domains</p>
                    <p class="mt-1 text-2xl font-semibold text-white">{{ number_format($activeDomains->count()) }}</p>
                </div>
                <div class="rounded-xl border border-white/10 bg-slate-950/70 p-4">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Verified</p>
                    <p class="mt-1 text-2xl font-semibold text-emerald-300">{{ number_format($verifiedDomainCount) }}</p>
                </div>
                <div class="rounded-xl border border-white/10 bg-slate-950/70 p-4">
                    <p class="text-xs uppercase tracking-wide text-slate-400">Pending verification</p>
                    <p class="mt-1 text-2xl font-semibold text-amber-300">{{ number_format($pendingDomainCount) }}</p>
                </div>
            </div>
        </div>
    </section>

    <section class="mt-6 grid gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
        <article id="domain-add-flow" class="idx-card p-5">
            <h3 class="text-base font-semibold text-white">Register additional domain</h3>
            <p class="mt-2 text-sm text-slate-300">Add approved hostnames for non-GHL embeds and IDX routing.</p>
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
                <label for="domains-tab-domain-slug" class="text-xs font-semibold uppercase tracking-wide text-slate-400">Domain hostname</label>
                <div class="flex flex-col gap-2 sm:flex-row">
                    <input id="domains-tab-domain-slug" name="domain_slug" type="text" value="{{ old('domain_slug') }}" placeholder="example.com" class="w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none" required @disabled($domainLimitReached)>
                    <button type="submit" @disabled($domainLimitReached) class="inline-flex min-h-10 items-center justify-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400 disabled:cursor-not-allowed disabled:opacity-50">Add Domain</button>
                </div>
                @error('domain_slug')
                    <p class="rounded-md border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-200">{{ $message }}</p>
                @enderror
                <p class="text-xs text-slate-400">Hostname only, no protocol/path. Example: <span class="font-mono">searchtampabayhouses.com</span></p>
            </form>
            @if ($domainLimitReached)
                <p class="mt-3 rounded-md border border-amber-400/30 bg-amber-900/20 px-3 py-2 text-xs text-amber-100">Domain limit reached. Purchase an additional slot or remove an existing domain.</p>
            @endif
            @if ($canPurchaseExtraDomainSlots)
                <div class="mt-4 rounded-lg border border-cyan-400/25 bg-slate-950/60 p-3">
                    <p class="text-xs font-semibold text-cyan-100">Need more capacity?</p>
                    <p class="mt-1 text-[11px] text-slate-400">Add one approved-hostname slot at {{ config('billing.addons.extra_domain.monthly_display', '$39/mo') }}.</p>
                    <form method="POST" action="{{ route('dashboard.billing.extra-domain', [], false) }}" class="mt-2">
                        @csrf
                        <button type="submit" class="inline-flex min-h-10 items-center justify-center rounded-lg border border-cyan-400/50 bg-cyan-500/10 px-3 py-2 text-xs font-semibold text-cyan-100 hover:bg-cyan-500/20">Add domain slot to subscription</button>
                    </form>
                </div>
            @endif
        </article>

        <article id="domain-verification-list" class="idx-card p-5" wire:poll.visible.10s>
            <div class="flex flex-wrap items-center justify-between gap-2">
                <h3 class="text-base font-semibold text-white">Domain verification queue</h3>
                <a href="{{ route('leadconnector.install', [], false) }}" class="inline-flex text-xs font-semibold text-cyan-300 hover:text-cyan-200">Manage GHL site keys</a>
            </div>
            <p class="mt-2 text-sm text-slate-300">Run TXT verification or GHL verification per domain to enable widget access.</p>
            <div class="mt-4 space-y-3 text-xs text-slate-200">
                @forelse ($activeDomains as $domain)
                    <div class="space-y-2 rounded-lg border border-white/10 bg-slate-950/70 px-3 py-3">
                        <div class="flex items-center justify-between gap-3">
                            <span class="truncate text-sm font-medium">{{ $domain->domain_slug }}</span>
                            <span class="rounded-full px-2 py-1 text-[10px] font-semibold {{ in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true) ? 'bg-emerald-500/20 text-emerald-200' : 'bg-amber-500/20 text-amber-200' }}">{{ strtoupper((string) $domain->verification_status) }}</span>
                        </div>
                        @if (!in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true))
                            <div class="rounded-md border border-cyan-400/25 bg-slate-900/60 p-2 text-[11px] text-slate-300">
                                <p>TXT Name: <span class="font-mono">{{ $domain->txt_verification_name ?: '_geoidx.' . $domain->domain_slug }}</span></p>
                                <p class="mt-1">TXT Value: <span class="font-mono">{{ $domain->txt_verification_value ?: 'Pending challenge' }}</span></p>
                            </div>
                        @endif
                        <div class="flex flex-wrap gap-2">
                            <form method="POST" action="{{ route('dashboard.domains.verify-txt', ['domain' => $domain->id], false) }}">@csrf<button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-cyan-400/40 px-2 py-1 text-[11px] font-semibold text-cyan-200 hover:bg-cyan-500/10">Verify TXT</button></form>
                            <form method="POST" action="{{ route('dashboard.domains.verify-ghl', ['domain' => $domain->id], false) }}">@csrf<button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-indigo-400/40 px-2 py-1 text-[11px] font-semibold text-indigo-200 hover:bg-indigo-500/10">Verify via GHL</button></form>
                            <form method="POST" action="{{ route('dashboard.domains.destroy', ['domain' => $domain->id], false) }}" onsubmit="return confirm('Remove this domain from approved hostnames?');">@csrf @method('DELETE')<button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-rose-400/40 px-2 py-1 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/10">Remove</button></form>
                        </div>
                    </div>
                @empty
                    <p class="rounded-lg border border-white/10 bg-slate-950/70 px-3 py-3 text-slate-400">No approved domains found yet. Add one to start verification.</p>
                @endforelse
            </div>
        </article>
    </section>
@endif
