@php
    /** @var string $setupPhase */
    /** @var \Illuminate\Support\Collection<int, \App\Models\Domain> $activeDomains */
    /** @var \Illuminate\Support\Collection<int, \App\Models\Domain> $pendingDomains */
    /** @var \Illuminate\Support\Collection<int, \Laravel\Sanctum\PersonalAccessToken> $apiTokens */
    /** @var list<string> $mlsCatalogFeedCodes */
    /** @var array<string, string> $mlsFeedLabels */
    /** @var bool $hasProductionToken */
    /** @var bool $hasStagingToken */
    /** @var bool $canCreateStagingToken */
    /** @var ?string $primaryVerifiedDomainSlug */

    $verifiedDomains = $activeDomains->whereIn('verification_status', ['verified', 'verified_ghl']);
    $steps = [
        ['key' => 'register', 'label' => 'Add your site'],
        ['key' => 'verify', 'label' => 'Verify ownership'],
        ['key' => 'connect', 'label' => 'Connect your app'],
    ];
    $phaseIndex = array_search($setupPhase, ['register', 'verify', 'ready'], true);
    $tokenNames = $apiTokens->pluck('name')->map(fn (string $n): string => mb_strtolower($n))->all();
@endphp

<section class="mt-4">
    {{-- Stepper --}}
    <div class="rounded-2xl border border-cyan-400/20 bg-slate-900/70 p-6">
        <h1 class="text-xl font-semibold text-white">Setup</h1>
        <p class="mt-1 text-sm text-slate-300">Get your MLS API running in three steps.</p>
        <ol class="mt-5 flex gap-4" aria-label="Setup progress">
            @foreach ($steps as $i => $step)
                @php
                    $stepIndex = $i;
                    $isComplete = $stepIndex < $phaseIndex || ($stepIndex === 2 && $setupPhase === 'ready');
                    $isCurrent = ($stepIndex === 0 && $setupPhase === 'register')
                        || ($stepIndex === 1 && $setupPhase === 'verify')
                        || ($stepIndex === 2 && $setupPhase === 'ready');
                @endphp
                <li class="flex flex-1 items-center gap-2">
                    <span
                        class="flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-bold
                            {{ $isComplete ? 'bg-emerald-500 text-slate-950' : ($isCurrent ? 'bg-cyan-500 text-slate-950 ring-2 ring-cyan-400/50' : 'bg-slate-700 text-slate-400') }}"
                        {{ $isCurrent ? 'aria-current="step"' : '' }}
                    >
                        @if ($isComplete)
                            <svg class="size-4" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path></svg>
                        @else
                            {{ $stepIndex + 1 }}
                        @endif
                    </span>
                    <span class="text-sm font-medium {{ $isCurrent ? 'text-white' : ($isComplete ? 'text-emerald-200' : 'text-slate-400') }}">{{ $step['label'] }}</span>
                    @if ($stepIndex < 2)
                        <span class="ml-auto hidden h-px flex-1 bg-white/10 sm:block"></span>
                    @endif
                </li>
            @endforeach
        </ol>
    </div>
</section>

{{-- Step 1: Register domain + MLS --}}
@if ($setupPhase === 'register')
<section class="mt-6">
    <div class="idx-card p-6">
        <h2 class="text-lg font-semibold text-white">Add your site</h2>
        <p class="mt-1 text-sm text-slate-300">Register the hostname where your app runs. You can add more domains later.</p>

        <form method="POST" action="{{ route('dashboard.domains.store', [], false) }}" class="mt-5 space-y-4">
            @csrf
            <div>
                <label for="setup-domain-slug" class="block text-xs font-semibold uppercase tracking-wide text-slate-300">Domain hostname</label>
                <div class="mt-1 flex flex-col gap-2 sm:flex-row">
                    <input
                        id="setup-domain-slug"
                        name="domain_slug"
                        type="text"
                        value="{{ old('domain_slug') }}"
                        placeholder="searchtampabayhouses.com"
                        required
                        class="min-h-11 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none"
                    >
                </div>
                @error('domain_slug')
                    <p class="mt-1 text-xs text-rose-300">{{ $message }}</p>
                @enderror
                <p class="mt-1 text-xs text-slate-400">Hostname only — no protocol or path.</p>
            </div>

            <div>
                <p class="text-xs font-semibold uppercase tracking-wide text-slate-300">MLS feeds</p>
                <p class="mt-0.5 text-xs text-slate-400">Select the MLS data sources this domain may access.</p>
                <div class="mt-2 flex flex-wrap gap-2">
                    @foreach ($mlsCatalogFeedCodes as $code)
                        <label class="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-slate-900/80 px-2.5 py-1.5">
                            <input
                                type="checkbox"
                                name="allowed_mls_datasets[]"
                                value="{{ $code }}"
                                @checked($loop->first)
                            >
                            <span class="text-xs text-slate-200">{{ $mlsFeedLabels[$code] ?? $code }}</span>
                        </label>
                    @endforeach
                </div>
                @error('allowed_mls_datasets')
                    <p class="mt-1 text-xs text-rose-300">{{ $message }}</p>
                @enderror
            </div>

            <button type="submit" class="inline-flex min-h-11 items-center justify-center rounded-lg bg-cyan-500 px-5 py-2.5 text-sm font-semibold text-slate-950 hover:bg-cyan-400">
                Add domain
            </button>
        </form>
    </div>
</section>
@endif

{{-- Step 2: Verify TXT --}}
@if ($setupPhase === 'verify')
<section class="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
    {{-- Pending domains --}}
    <div class="idx-card p-6" wire:poll.visible.10s>
        <h2 class="text-lg font-semibold text-white">Verify ownership</h2>
        <p class="mt-1 text-sm text-slate-300">Publish the DNS TXT record for each domain, then click Verify.</p>

        <div class="mt-4 space-y-4">
            @forelse ($pendingDomains as $domain)
                <div class="space-y-3 rounded-lg border border-amber-400/20 bg-slate-950/70 p-4">
                    <div class="flex items-center justify-between gap-3">
                        <span class="truncate text-sm font-medium text-slate-100">{{ $domain->domain_slug }}</span>
                        <span class="rounded-full bg-amber-500/20 px-2 py-1 text-[10px] font-semibold text-amber-200">PENDING</span>
                    </div>
                    <div class="rounded-md border border-cyan-400/25 bg-slate-900/60 p-3 text-xs text-slate-300 space-y-1" x-data="{ copiedName: false, copiedValue: false }">
                        <div class="flex items-center justify-between gap-2">
                            <p>TXT Name: <span class="font-mono">{{ $domain->txt_verification_name ?: '_geoidx.'.$domain->domain_slug }}</span></p>
                            <button type="button" @click="navigator.clipboard.writeText('{{ $domain->txt_verification_name ?: '_geoidx.'.$domain->domain_slug }}'); copiedName = true; setTimeout(() => copiedName = false, 2000);" class="shrink-0 text-xs font-semibold text-cyan-200 hover:text-cyan-100">
                                <span x-text="copiedName ? 'Copied' : 'Copy'"></span>
                            </button>
                        </div>
                        <div class="flex items-center justify-between gap-2">
                            <p>TXT Value: <span class="font-mono">{{ $domain->txt_verification_value ?: 'Pending challenge' }}</span></p>
                            @if ($domain->txt_verification_value)
                                <button type="button" @click="navigator.clipboard.writeText('{{ $domain->txt_verification_value }}'); copiedValue = true; setTimeout(() => copiedValue = false, 2000);" class="shrink-0 text-xs font-semibold text-cyan-200 hover:text-cyan-100">
                                    <span x-text="copiedValue ? 'Copied' : 'Copy'"></span>
                                </button>
                            @endif
                        </div>
                    </div>
                    <div class="flex flex-wrap gap-2">
                        <form method="POST" action="{{ route('dashboard.domains.verify-txt', ['domain' => $domain->id], false) }}">
                            @csrf
                            <button type="submit" class="inline-flex min-h-9 items-center rounded-lg border border-cyan-400/40 px-3 py-1.5 text-xs font-semibold text-cyan-200 hover:bg-cyan-500/10">Verify TXT</button>
                        </form>
                        <form method="POST" action="{{ route('dashboard.domains.destroy', ['domain' => $domain->id], false) }}" onsubmit="return confirm('Remove this domain?');">
                            @csrf @method('DELETE')
                            <button type="submit" class="inline-flex min-h-9 items-center rounded-lg border border-rose-400/40 px-3 py-1.5 text-xs font-semibold text-rose-200 hover:bg-rose-500/10">Remove</button>
                        </form>
                    </div>
                </div>
            @empty
                <p class="text-sm text-slate-400">All domains verified. Generating your API key...</p>
            @endforelse
        </div>
    </div>

    {{-- Add another domain --}}
    <div class="idx-card p-6">
        <h3 class="text-base font-semibold text-white">Add another domain</h3>
        <form method="POST" action="{{ route('dashboard.domains.store', [], false) }}" class="mt-4 space-y-4">
            @csrf
            <div>
                <label for="setup-domain-slug-extra" class="block text-xs font-semibold uppercase tracking-wide text-slate-300">Domain hostname</label>
                <input
                    id="setup-domain-slug-extra"
                    name="domain_slug"
                    type="text"
                    value="{{ old('domain_slug') }}"
                    placeholder="example.com"
                    required
                    class="mt-1 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-400 focus:outline-none"
                >
                @error('domain_slug')
                    <p class="mt-1 text-xs text-rose-300">{{ $message }}</p>
                @enderror
            </div>
            <div>
                <p class="text-xs font-semibold uppercase tracking-wide text-slate-300">MLS feeds</p>
                <div class="mt-2 flex flex-wrap gap-2">
                    @foreach ($mlsCatalogFeedCodes as $code)
                        <label class="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-slate-900/80 px-2.5 py-1.5">
                            <input type="checkbox" name="allowed_mls_datasets[]" value="{{ $code }}" @checked($loop->first)>
                            <span class="text-xs text-slate-200">{{ $mlsFeedLabels[$code] ?? $code }}</span>
                        </label>
                    @endforeach
                </div>
            </div>
            <button type="submit" class="inline-flex min-h-10 items-center justify-center rounded-lg bg-cyan-500 px-4 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400">Add domain</button>
        </form>
    </div>
</section>
@endif

{{-- Step 3: Connect --}}
@if ($setupPhase === 'ready')
<section class="mt-6 space-y-6">
    {{-- New API token flash --}}
    @php $newToken = session('dashboard_new_api_token') @endphp
    @if ($newToken)
        <div class="rounded-xl border border-amber-400/40 bg-amber-900/20 p-5" x-data="{ copied: false }">
            <p class="text-sm font-semibold text-amber-100">Your Production API key was generated — copy it now. It will not be shown again.</p>
            <p class="mt-3 break-all font-mono text-xs text-amber-50/90">{{ $newToken }}</p>
            <button
                type="button"
                @click="
                    navigator.clipboard.writeText('{{ $newToken }}');
                    copied = true;
                    setTimeout(() => copied = false, 2000);
                "
                class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-amber-400/30 px-3 py-1.5 text-xs font-semibold text-amber-200 transition hover:bg-amber-500/10"
            >
                <svg x-show="!copied" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"></path>
                </svg>
                <svg x-show="copied" x-cloak class="size-3.5 text-emerald-400" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path>
                </svg>
                <span x-text="copied ? 'Copied!' : 'Copy API key'"></span>
            </button>
        </div>
    @endif

    {{-- API connection info --}}
    <div class="idx-card p-6">
        <h2 class="text-lg font-semibold text-white">Connect your app</h2>
        <p class="mt-1 text-sm text-slate-300">Use these values in every API request.</p>

        <div class="mt-4 space-y-3 rounded-lg border border-white/10 bg-slate-950/70 p-4 text-xs" x-data="{ copiedHeader: false, copiedSlug: false }">
            <div>
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-300">API base URL</p>
                <p class="mt-1 font-mono text-slate-100">{{ $apiPublicUrl }}</p>
            </div>
            <div class="flex items-center justify-between gap-2">
                <div>
                    <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-300">Domain slug</p>
                    <p class="mt-1 font-mono text-slate-100">X-Domain-Slug: {{ $primaryVerifiedDomainSlug ?? 'your-verified-domain.com' }}</p>
                </div>
                @if ($primaryVerifiedDomainSlug)
                    <button type="button" @click="navigator.clipboard.writeText('{{ $primaryVerifiedDomainSlug }}'); copiedSlug = true; setTimeout(() => copiedSlug = false, 2000);" class="shrink-0 text-xs font-semibold text-cyan-200 hover:text-cyan-100">
                        <span x-text="copiedSlug ? 'Copied' : 'Copy'"></span>
                    </button>
                @endif
            </div>
            <div>
                <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-300">Authorization header</p>
                <p class="mt-1 font-mono text-slate-100">Authorization: Bearer &lt;your-api-key&gt;</p>
            </div>
        </div>
    </div>

    {{-- Production key status --}}
    <div class="idx-card p-6">
        <div class="flex items-center justify-between gap-3">
            <h3 class="text-base font-semibold text-white">Production key</h3>
            @if ($hasProductionToken)
                <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/20 px-2.5 py-1 text-xs font-semibold text-emerald-200">
                    <svg class="size-3" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path></svg>
                    Active
                </span>
            @else
                <span class="text-xs text-slate-400">Not created yet</span>
            @endif
        </div>
        @if ($hasProductionToken)
            <p class="mt-2 text-sm text-slate-300">Your Production key is active. It was shown once when your first domain was verified.</p>
        @else
            <p class="mt-2 text-sm text-slate-300">Verify a domain to auto-generate your Production key.</p>
        @endif
    </div>

    {{-- Staging key --}}
    <div class="idx-card p-6">
        <div class="flex items-center justify-between gap-3">
            <h3 class="text-base font-semibold text-white">Staging key</h3>
            @if ($hasStagingToken)
                <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/20 px-2.5 py-1 text-xs font-semibold text-emerald-200">
                    <svg class="size-3" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path></svg>
                    Active
                </span>
            @else
                <span class="text-xs text-slate-400">Optional</span>
            @endif
        </div>
        @if ($hasStagingToken)
            <p class="mt-2 text-sm text-slate-300">Your Staging key is active. Use it for preview or staging deployments — same domain slug, different Bearer token.</p>
        @elseif ($canCreateStagingToken)
            <p class="mt-2 text-sm text-slate-300">Use a separate key for staging or preview sites. Same verified domain, different Bearer token.</p>
            <form method="POST" action="{{ route('dashboard.api-tokens.staging', [], false) }}">
                @csrf
                <button type="submit" class="mt-3 inline-flex min-h-10 items-center justify-center rounded-lg border border-cyan-400/40 px-4 py-2 text-xs font-semibold text-cyan-200 hover:bg-cyan-500/10">
                    Generate staging key
                </button>
            </form>
        @endif
    </div>

    {{-- Verified domain list with inline MLS edit --}}
    @if ($verifiedDomains->isNotEmpty())
        <div class="idx-card p-6">
            <h3 class="text-base font-semibold text-white">Verified domains</h3>
            <div class="mt-3 space-y-4">
                @foreach ($verifiedDomains as $domain)
                    <div class="space-y-3 rounded-lg border border-emerald-400/20 bg-slate-950/70 p-4">
                        <div class="flex items-center justify-between gap-3">
                            <span class="truncate text-sm font-medium text-slate-100">{{ $domain->domain_slug }}</span>
                            <div class="flex items-center gap-2">
                                <span class="rounded-full bg-emerald-500/20 px-2 py-1 text-[10px] font-semibold text-emerald-200">VERIFIED</span>
                                <form method="POST" action="{{ route('dashboard.domains.destroy', ['domain' => $domain->id], false) }}" onsubmit="return confirm('Remove this domain?');">
                                    @csrf @method('DELETE')
                                    <button type="submit" class="inline-flex min-h-7 items-center rounded border border-rose-400/40 px-2 py-0.5 text-[10px] font-semibold text-rose-200 hover:bg-rose-500/10">Remove</button>
                                </form>
                            </div>
                        </div>
                        @php
                            $selectedFeeds = $domain->getAllowedMlsDatasets() ?? $mlsCatalogFeedCodes;
                            $defaultFeed = $domain->getMlsDataset() ?? ($selectedFeeds[0] ?? ($mlsCatalogFeedCodes[0] ?? 'stellar'));
                        @endphp
                        <form method="POST" action="{{ route('dashboard.domains.mls.update', ['domain' => $domain->id], false) }}" class="space-y-3 border-t border-white/10 pt-3">
                            @csrf @method('PUT')
                            <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-300">Allowed MLS feeds</p>
                            <div class="flex flex-wrap gap-2">
                                @foreach ($mlsCatalogFeedCodes as $code)
                                    <label class="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-slate-900/80 px-2 py-1">
                                        <input type="checkbox" name="allowed_mls_datasets[]" value="{{ $code }}" @checked(in_array($code, $selectedFeeds, true))>
                                        <span class="text-[11px] text-slate-200">{{ $mlsFeedLabels[$code] ?? $code }}</span>
                                    </label>
                                @endforeach
                            </div>
                            <label class="block text-[11px] font-semibold uppercase tracking-wide text-slate-300">
                                Default feed
                                <select name="mls_dataset" class="mt-1 w-full rounded-md border border-white/20 bg-slate-950 px-2 py-1.5 text-sm text-slate-100">
                                    @foreach ($selectedFeeds as $code)
                                        <option value="{{ $code }}" @selected($defaultFeed === $code)>{{ $mlsFeedLabels[$code] ?? $code }}</option>
                                    @endforeach
                                </select>
                            </label>
                            <button type="submit" class="inline-flex min-h-9 items-center rounded-lg bg-cyan-500/80 px-3 py-1.5 text-xs font-semibold text-slate-950 hover:bg-cyan-400">Save</button>
                        </form>
                    </div>
                @endforeach
            </div>
        </div>
    @endif

    {{-- Admin invite (deferred until ready) --}}
    @if (! empty($canInviteUsers))
        <section class="rounded-2xl border border-cyan-400/25 bg-slate-900/80 p-5 shadow-lg shadow-cyan-900/10">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-cyan-200">Invite a teammate</h2>
            <p class="mt-1 text-sm text-slate-300">Send an email invitation. The recipient completes registration using the same MLS and domain flow.</p>
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
                <button type="submit" class="inline-flex shrink-0 items-center justify-center rounded-xl bg-cyan-500 px-4 py-2.5 text-sm font-semibold text-slate-950 hover:bg-cyan-400">
                    Send invitation
                </button>
            </form>
        </section>
    @endif

    {{-- Link to API keys panel --}}
    <p class="text-center text-xs text-slate-400">
        <a wire:navigate href="{{ \App\Support\DashboardUrl::panel('api') }}" class="font-semibold text-cyan-200 hover:text-cyan-100">Manage all API keys</a> — revoke tokens, generate additional named keys.
    </p>
</section>
@endif
