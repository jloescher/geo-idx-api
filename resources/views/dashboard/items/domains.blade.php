@if ($activePanel === 'domains')
    @php
        $verifiedDomainCount = $activeDomains->whereIn('verification_status', ['verified', 'verified_ghl'])->count();
        $pendingDomainCount = max(0, $activeDomains->count() - $verifiedDomainCount);
        /** @var list<string> $mlsCatalogFeedCodes */
        $mlsCatalogFeedCodes = $mlsCatalogFeedCodes ?? [];
    @endphp
    <section class="mt-8">
        <div class="rounded-2xl border border-cyan-400/20 bg-slate-900/70 p-6">
            <h2 class="text-xl font-semibold text-white">Domains</h2>
            <p class="mt-1 text-sm text-slate-300">Verify hostnames with DNS TXT, then choose which MLS feeds each domain may call through the API.</p>
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
            <h3 class="text-base font-semibold text-white">Add domain</h3>
            <p class="mt-2 text-sm text-slate-300">Register a hostname you control. After TXT verification you can bind API tokens using <span class="font-mono text-xs">X-Domain-Slug</span>.</p>
            <p class="mt-2 text-xs text-slate-400">
                Domains on file: {{ $activeDomains->count() }}
            </p>
            <form method="POST" action="{{ route('dashboard.domains.store', [], false) }}" class="mt-4 space-y-2">
                @csrf
                <label for="domains-tab-domain-slug" class="text-xs font-semibold uppercase tracking-wide text-slate-400">Domain hostname</label>
                <div class="flex flex-col gap-2 sm:flex-row">
                    <input id="domains-tab-domain-slug" name="domain_slug" type="text" value="{{ old('domain_slug') }}" placeholder="example.com" class="w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none" required>
                    <button type="submit" class="inline-flex min-h-10 items-center justify-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400">Add Domain</button>
                </div>
                @error('domain_slug')
                    <p class="rounded-md border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-200">{{ $message }}</p>
                @enderror
                <p class="text-xs text-slate-400">Hostname only, no protocol/path. Example: <span class="font-mono">searchtampabayhouses.com</span></p>
            </form>
        </article>

        <article id="domain-verification-list" class="idx-card p-5" wire:poll.visible.10s>
            <div class="flex flex-wrap items-center justify-between gap-2">
                <h3 class="text-base font-semibold text-white">Your domains</h3>
            </div>
            <p class="mt-2 text-sm text-slate-300">Run TXT verification, then pick MLS feeds for each verified domain.</p>
            <div class="mt-4 space-y-4 text-xs text-slate-200">
                @forelse ($activeDomains as $domain)
                    <div class="space-y-3 rounded-lg border border-white/10 bg-slate-950/70 px-3 py-3">
                        <div class="flex items-center justify-between gap-3">
                            <span class="truncate text-sm font-medium">{{ $domain->domain_slug }}</span>
                            <span class="rounded-full px-2 py-1 text-[10px] font-semibold {{ in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true) ? 'bg-emerald-500/20 text-emerald-200' : 'bg-amber-500/20 text-amber-200' }}">{{ strtoupper((string) $domain->verification_status) }}</span>
                        </div>
                        @if (! in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true))
                            <div class="rounded-md border border-cyan-400/25 bg-slate-900/60 p-2 text-[11px] text-slate-300">
                                <p>TXT Name: <span class="font-mono">{{ $domain->txt_verification_name ?: '_geoidx.'.$domain->domain_slug }}</span></p>
                                <p class="mt-1">TXT Value: <span class="font-mono">{{ $domain->txt_verification_value ?: 'Pending challenge' }}</span></p>
                            </div>
                        @endif
                        <div class="flex flex-wrap gap-2">
                            <form method="POST" action="{{ route('dashboard.domains.verify-txt', ['domain' => $domain->id], false) }}">@csrf<button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-cyan-400/40 px-2 py-1 text-[11px] font-semibold text-cyan-200 hover:bg-cyan-500/10">Verify TXT</button></form>
                            <form method="POST" action="{{ route('dashboard.domains.destroy', ['domain' => $domain->id], false) }}" onsubmit="return confirm('Remove this domain?');">@csrf @method('DELETE')<button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-rose-400/40 px-2 py-1 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/10">Remove</button></form>
                        </div>

                        @if (in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true))
                            @php
                                $selectedFeeds = $domain->getAllowedMlsDatasets() ?? $mlsCatalogFeedCodes;
                                $defaultFeed = $domain->getMlsDataset() ?? ($selectedFeeds[0] ?? ($mlsCatalogFeedCodes[0] ?? 'stellar'));
                            @endphp
                            <form method="POST" action="{{ route('dashboard.domains.mls.update', ['domain' => $domain->id], false) }}" class="mt-2 space-y-3 border-t border-white/10 pt-3">
                                @csrf
                                @method('PUT')
                                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Allowed MLS feeds</p>
                                <div class="flex flex-wrap gap-2">
                                    @foreach ($mlsCatalogFeedCodes as $code)
                                        <label class="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-slate-900/80 px-2 py-1">
                                            <input
                                                type="checkbox"
                                                name="allowed_mls_datasets[]"
                                                value="{{ $code }}"
                                                @checked(in_array($code, $selectedFeeds, true))
                                            >
                                            <span class="font-mono text-[11px] text-slate-200">{{ $code }}</span>
                                        </label>
                                    @endforeach
                                </div>
                                <label class="block text-[11px] font-semibold uppercase tracking-wide text-slate-400">
                                    Default feed (when <span class="font-mono">?dataset=</span> omitted)
                                    <select name="mls_dataset" class="mt-1 w-full rounded-md border border-white/20 bg-slate-950 px-2 py-1.5 text-sm text-slate-100">
                                        @foreach ($selectedFeeds as $code)
                                            <option value="{{ $code }}" @selected($defaultFeed === $code)>{{ $code }}</option>
                                        @endforeach
                                    </select>
                                </label>
                                <button type="submit" class="inline-flex min-h-9 items-center rounded-lg bg-violet-500 px-3 py-1.5 text-xs font-semibold text-white hover:bg-violet-400">Save MLS access</button>
                            </form>
                        @endif
                    </div>
                @empty
                    <p class="rounded-lg border border-white/10 bg-slate-950/70 px-3 py-3 text-slate-400">No domains yet. Add one to start verification.</p>
                @endforelse
            </div>
        </article>
    </section>
@endif
