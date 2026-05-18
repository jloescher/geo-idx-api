@if ($activePanel === 'onboarding')
    <section class="mt-8 grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]">
        <div class="space-y-6">
            <section id="onboarding-approved-domains" class="idx-card p-6">
                <h2 class="text-lg font-semibold text-white">Domains</h2>
                <p class="mt-1 text-sm text-slate-300">Add hostnames you control, verify TXT, then choose MLS feeds per domain under the Domains tab.</p>
                <p class="mt-2 text-xs text-slate-400">
                    Domains on file: {{ $activeDomains->count() }}
                </p>
                <form method="POST" action="{{ route('dashboard.domains.store', [], false) }}" class="mt-4 space-y-2">
                    @csrf
                    <label for="dashboard-domain-slug" class="text-xs font-semibold uppercase tracking-wide text-slate-300">Add domain</label>
                    <div class="flex flex-col gap-2 sm:flex-row">
                        <input id="dashboard-domain-slug" name="domain_slug" type="text" value="{{ old('domain_slug') }}" placeholder="example.com" class="w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none" required>
                        <button type="submit" class="inline-flex min-h-10 items-center justify-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400">Add Domain</button>
                    </div>
                    @error('domain_slug')
                        <p class="rounded-md border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-200">{{ $message }}</p>
                    @enderror
                    <p class="text-xs text-slate-400">Use hostname only. Example: <span class="font-mono">searchtampabayhouses.com</span></p>
                </form>
                <div class="mt-4 space-y-2 text-xs text-slate-200">
                    @forelse ($activeDomains as $domain)
                        <div class="space-y-2 rounded-lg border border-white/10 bg-slate-950/70 px-3 py-2">
                            <div class="flex items-center justify-between gap-3">
                                <span class="truncate">{{ $domain->domain_slug }}</span>
                                <span class="rounded-full px-2 py-1 text-[10px] font-semibold {{ in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true) ? 'bg-emerald-500/20 text-emerald-200' : 'bg-amber-500/20 text-amber-200' }}">{{ strtoupper((string) $domain->verification_status) }}</span>
                            </div>
                            @if (! in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true))
                                <div class="rounded-md border border-cyan-400/25 bg-slate-900/60 p-2 text-[11px] text-slate-300">
                                    <p>TXT Name: <span class="font-mono">{{ $domain->txt_verification_name ?: '_geoidx.'.$domain->domain_slug }}</span></p>
                                    <p class="mt-1">TXT Value: <span class="font-mono">{{ $domain->txt_verification_value ?: 'Pending challenge' }}</span></p>
                                </div>
                            @endif
                            <div class="flex flex-wrap gap-2">
                                <form method="POST" action="{{ route('dashboard.domains.verify-txt', ['domain' => $domain->id], false) }}">
                                    @csrf
                                    <button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-cyan-400/40 px-2 py-1 text-[11px] font-semibold text-cyan-200 hover:bg-cyan-500/10">Verify TXT</button>
                                </form>
                            </div>
                        </div>
                    @empty
                        <p class="text-slate-400">No domains yet.</p>
                    @endforelse
                </div>
                <a wire:navigate href="{{ \App\Support\DashboardUrl::panel('domains') }}" class="mt-4 inline-flex text-xs font-semibold text-cyan-200 hover:text-cyan-100">Open Domains tab</a>
            </section>
        </div>

        <section class="idx-card p-6" wire:poll.10s>
            <h2 class="text-xl font-semibold text-white">Getting started</h2>
            <p class="mt-1 text-sm text-slate-300">Complete these steps to use the MLS/GIS API with your own stack.</p>
            <p class="mt-2 text-sm text-slate-300">{{ $onboardingCompletedCount }}/{{ count($onboardingSteps) }} completed</p>
            <div class="mt-4 space-y-3">
                @foreach ($onboardingSteps as $step)
                    <div class="flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left transition {{ $step['done'] ? 'border-emerald-400/40 bg-emerald-900/20' : 'border-white/10 bg-slate-950/70' }}" data-event-name="dashboard_step_{{ $step['key'] }}">
                        <span class="text-sm text-slate-200">{{ $step['label'] }}</span>
                        <span class="text-xs font-semibold {{ $step['done'] ? 'text-emerald-200' : 'text-slate-400' }}">{{ $step['done'] ? 'Done' : 'Pending' }}</span>
                    </div>
                @endforeach
            </div>
        </section>
    </section>
@endif
