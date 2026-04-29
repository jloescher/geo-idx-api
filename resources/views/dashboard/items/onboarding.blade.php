@if ($activePanel === 'onboarding')
    <section class="mt-8 grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]">
        <div class="space-y-6">
            <section id="mls-membership" class="idx-card p-6">
                <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                    <div>
                        <h2 class="text-lg font-semibold text-white">MLS membership verification</h2>
                        <p class="mt-1 text-sm text-slate-300">Submit your Stellar MLS ID and MLS email to verify account eligibility for checkout and lead access.</p>
                    </div>
                    <span class="inline-flex w-fit rounded-full px-3 py-1 text-xs font-semibold {{ (string) auth()->user()->mls_membership_status === 'active' ? 'bg-emerald-500/20 text-emerald-200 ring-1 ring-emerald-400/30' : 'bg-amber-500/20 text-amber-200 ring-1 ring-amber-400/30' }}">
                        Status: {{ strtoupper((string) auth()->user()->mls_membership_status ?: 'pending') }}
                    </span>
                </div>
                @if (is_string(auth()->user()->mls_membership_last_error) && auth()->user()->mls_membership_last_error !== '')
                    <div class="mt-4 rounded-lg border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-100">{{ auth()->user()->mls_membership_last_error }}</div>
                @endif
                @error('mls_membership')
                    <div class="mt-4 rounded-lg border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-100">{{ $message }}</div>
                @enderror
                <form method="POST" action="{{ route('dashboard.mls-membership.store', [], false) }}" class="mt-5 grid gap-4 sm:grid-cols-2">
                    @csrf
                    <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">MLS ID
                        <input name="mls_id" type="text" value="{{ old('mls_id', (string) auth()->user()->mls_id) }}" placeholder="MLS123456" class="mt-2 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none" required>
                        @error('mls_id')<span class="mt-1 block text-xs text-rose-300">{{ $message }}</span>@enderror
                    </label>
                    <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">MLS Email
                        <input name="mls_email" type="email" value="{{ old('mls_email', (string) auth()->user()->mls_email) }}" placeholder="you@brokerage.com" class="mt-2 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none" required>
                        @error('mls_email')<span class="mt-1 block text-xs text-rose-300">{{ $message }}</span>@enderror
                    </label>
                    <div class="sm:col-span-2">
                        <button type="submit" class="inline-flex min-h-11 items-center rounded-lg bg-cyan-500 px-4 py-2 text-sm font-semibold text-slate-950 hover:bg-cyan-400">Verify MLS Membership</button>
                    </div>
                </form>
            </section>

            <section id="onboarding-approved-domains" class="idx-card p-6">
                <h2 class="text-lg font-semibold text-white">My Approved Domains</h2>
                <p class="mt-1 text-sm text-slate-300">Add your first widget domain here. Additional domains are managed in the Domains command center.</p>
                <p class="mt-2 text-xs text-slate-400">
                    @if (is_int($domainLimit))
                        Plan usage: {{ $activeDomains->count() }} / {{ $domainLimit }} domains
                    @else
                        Plan usage: {{ $activeDomains->count() }} domains (unlimited on current plan)
                    @endif
                </p>
                @php($onboardingDomainLimitReached = $activeDomains->count() >= 1)
                <form method="POST" action="{{ route('dashboard.domains.store', [], false) }}" class="mt-4 space-y-2">
                    @csrf
                    <label for="dashboard-domain-slug" class="text-xs font-semibold uppercase tracking-wide text-slate-400">Add domain</label>
                    <div class="flex flex-col gap-2 sm:flex-row">
                        <input id="dashboard-domain-slug" name="domain_slug" type="text" value="{{ old('domain_slug') }}" placeholder="example.com" class="w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none" required @disabled($onboardingDomainLimitReached)>
                        <button type="submit" @disabled($onboardingDomainLimitReached) class="inline-flex min-h-10 items-center justify-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400 disabled:cursor-not-allowed disabled:opacity-50">Add Domain</button>
                    </div>
                    @if ($onboardingDomainLimitReached)
                        <p class="rounded-md border border-amber-400/30 bg-amber-900/20 px-3 py-2 text-xs text-amber-100">Onboarding supports one initial widget domain. Add additional domains in the Domains tab.</p>
                    @endif
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
                            @if (!in_array((string) $domain->verification_status, ['verified', 'verified_ghl'], true))
                                <div class="rounded-md border border-cyan-400/25 bg-slate-900/60 p-2 text-[11px] text-slate-300">
                                    <p>TXT Name: <span class="font-mono">{{ $domain->txt_verification_name ?: '_geoidx.' . $domain->domain_slug }}</span></p>
                                    <p class="mt-1">TXT Value: <span class="font-mono">{{ $domain->txt_verification_value ?: 'Pending challenge' }}</span></p>
                                </div>
                            @endif
                            <div class="flex flex-wrap gap-2">
                                <form method="POST" action="{{ route('dashboard.domains.verify-txt', ['domain' => $domain->id], false) }}">
                                    @csrf
                                    <button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-cyan-400/40 px-2 py-1 text-[11px] font-semibold text-cyan-200 hover:bg-cyan-500/10">Verify TXT</button>
                                </form>
                                <form method="POST" action="{{ route('dashboard.domains.verify-ghl', ['domain' => $domain->id], false) }}">
                                    @csrf
                                    <button type="submit" class="inline-flex min-h-8 items-center rounded-md border border-indigo-400/40 px-2 py-1 text-[11px] font-semibold text-indigo-200 hover:bg-indigo-500/10">Verify via GHL</button>
                                </form>
                            </div>
                        </div>
                    @empty
                        <p class="text-slate-400">No approved domains found yet.</p>
                    @endforelse
                </div>
                <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'domains'], false) }}" class="mt-4 inline-flex text-xs font-semibold text-cyan-300 hover:text-cyan-200">Open Domains command center</a>
            </section>
        </div>

        <section class="idx-card p-6" wire:poll.10s>
            <h2 class="text-xl font-semibold text-white">Getting Started Checklist</h2>
            <p class="mt-1 text-sm text-slate-300">Complete these steps to unlock leads, checkout, and live widget usage.</p>
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
