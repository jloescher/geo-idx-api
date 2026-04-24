<div
    x-data="{ openFaq: 0 }"
    class="bg-slate-950 text-slate-100 transition-colors duration-300"
    @keydown.escape.window="$wire.closeLoginModal()"
>
    <header class="sticky top-0 z-30 border-b border-white/15 bg-slate-950/90 backdrop-blur">
        <div class="mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
            <a href="/" class="text-lg font-semibold tracking-tight">Quantyra Labs IDX API</a>
            <div class="flex flex-wrap items-center justify-end gap-2">
                @if (Route::has('register'))
                    <a
                        href="{{ route('register') }}"
                        class="rounded-full border border-white/25 px-4 py-2 text-sm font-semibold text-slate-100 hover:border-white/40 hover:bg-white/5"
                    >
                        Create account
                    </a>
                @endif
                <button
                    type="button"
                    wire:click="openLoginModal"
                    class="rounded-full border border-white/25 px-4 py-2 text-sm font-semibold text-slate-100 hover:border-white/40 hover:bg-white/5"
                >
                    Subscriber login
                </button>
                <a href="#pricing" class="rounded-full bg-emerald-400 px-4 py-2 text-sm font-semibold text-slate-900">See pricing</a>
            </div>
        </div>
    </header>

    @if (session('flash_billing_error'))
        <div class="border-b border-amber-400/40 bg-amber-500/15 px-4 py-3 text-center text-sm text-amber-100">
            {{ session('flash_billing_error') }}
        </div>
    @endif

    @if (request()->query('checkout') === 'success')
        <div class="border-b border-emerald-400/40 bg-emerald-500/15 px-4 py-3 text-center text-sm text-emerald-50">
            Thanks — Stripe is finalizing your subscription. Refresh the dashboard in a moment if access is still pending.
        </div>
    @elseif (request()->query('checkout') === 'cancelled')
        <div class="border-b border-slate-600/60 bg-slate-800/80 px-4 py-3 text-center text-sm text-slate-200">
            Checkout cancelled. Your card was not charged.
        </div>
    @endif

    {{-- Revenue Impact: Hero compresses differentiation into one scan + routes to pricing. --}}
    <section class="relative overflow-hidden">
        <div class="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(59,130,246,0.22),transparent_45%)]"></div>
        <div class="mx-auto max-w-6xl px-4 pb-16 pt-14 sm:px-6 lg:px-8 lg:pb-20 lg:pt-20">
            <div class="grid items-center gap-10 lg:grid-cols-2">
                <div class="space-y-6">
                    <div class="flex flex-wrap items-center gap-2">
                        <span class="inline-flex rounded-full border border-blue-400/50 bg-blue-500/15 px-3 py-1 text-xs font-semibold uppercase tracking-wider text-blue-200">
                            Official Stellar MLS Consultant
                        </span>
                        <span class="inline-flex rounded-full border border-emerald-400/40 bg-emerald-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-wider text-emerald-200">
                            $0 data license fee
                        </span>
                    </div>
                    <h1 class="text-4xl font-black leading-tight tracking-tight sm:text-5xl">
                        Quantyra Labs IDX API
                    </h1>
                    <p class="text-lg font-semibold text-blue-100/90">
                        Official Stellar MLS Consultant • $0 Data License Fee
                    </p>
                    <p class="text-base leading-7 text-slate-200 sm:text-lg">
                        Live MLS Data Proxy + Image Proxy + JS Embed Widgets + LeadConnector App<br>
                        Build high-conversion IDX websites or embed directly into GoHighLevel / custom sites.<br>
                        <span class="font-semibold text-slate-50">Deeper SEO. Blazing speed. Hard lead gating included.</span>
                    </p>
                    <div class="flex flex-wrap gap-3">
                        <a href="#pricing" class="rounded-full bg-orange-500 px-5 py-3 text-sm font-semibold text-white shadow-lg shadow-orange-900/30 hover:bg-orange-400">
                            Add to cart — see plans
                        </a>
                        <a href="#widgets" class="rounded-full border border-white/30 px-5 py-3 text-sm font-semibold text-slate-100 hover:border-white/45 hover:bg-white/5">
                            Widget embeds
                        </a>
                        <button
                            type="button"
                            wire:click="openLoginModal"
                            class="rounded-full border border-white/30 px-5 py-3 text-sm font-semibold text-slate-100 hover:border-white/45 hover:bg-white/5"
                        >
                            Subscriber login
                        </button>
                    </div>
                </div>
                <div class="rounded-3xl border border-white/10 bg-gradient-to-br from-slate-800 to-slate-900 p-5 shadow-2xl">
                    <div class="rounded-2xl border border-white/10 bg-slate-950/60 p-5 text-sm text-slate-200">
                        <p class="text-xs font-semibold uppercase tracking-wide text-blue-300">Stack preview</p>
                        <ul class="mt-3 list-inside list-disc space-y-2">
                            <li>Laravel 13 + Octane + FrankenPHP edge</li>
                            <li>PostgreSQL freshness cache (~15 minutes)</li>
                            <li>Leaflet + OpenStreetMap maps</li>
                            <li>TextGrid phone + email OTP hard gating</li>
                        </ul>
                    </div>
                    <p class="mt-4 text-xs text-slate-400">
                        Illustrative only — no live MLS rows on this marketing host.
                    </p>
                </div>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Compliance proof reduces enterprise/legal friction on sales calls. --}}
    <section class="border-y border-white/10 bg-slate-900/40 py-12">
        <div class="mx-auto flex max-w-6xl flex-col gap-6 px-4 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
            <div class="flex items-start gap-4">
                <div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-2xl border border-blue-400/40 bg-blue-500/20 text-lg font-black text-blue-100">
                    S
                </div>
                <div>
                    <h2 class="text-xl font-bold text-white">Official Stellar MLS Consultant</h2>
                    <p class="mt-2 max-w-2xl text-sm text-slate-200">
                        Participant Data Access Agreement (signed 4/17/2026): Quantyra Labs LLC acts as Consultant;
                        Firm on file: <span class="font-semibold text-white">Realty Of America, LLC</span>;
                        IDX display permitted on authorized domains (example: <span class="font-mono text-blue-200">searchtampabayhouses.com</span>).
                        <span class="font-semibold text-emerald-200">$0 annual Stellar MLS data license fee</span> for qualifying IDX use under the agreement.
                    </p>
                </div>
            </div>
            <div class="rounded-2xl border border-white/15 bg-slate-950/80 px-5 py-4 text-xs leading-relaxed text-slate-300">
                Proof points: Consultant = Quantyra Labs LLC • Firm = Realty Of America, LLC • Agreement date 4/17/2026 •
                Consultant badge reserved for authorized marketing + onboarding materials tied to the signed program.
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Comparison grid accelerates “why switch from Realtyna / BuyingBuddy / AeroIDX”. --}}
    <section class="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
        <h2 class="text-3xl font-bold tracking-tight">Why teams switch from Realtyna, BuyingBuddy, AeroIDX…</h2>
        <p class="mt-3 max-w-3xl text-sm text-slate-300">
            Quantyra Labs IDX API is engineered for conversion: verified leads, faster edge delivery, and a single Laravel surface for REST + widgets + GHL.
        </p>
        <div class="mt-8 overflow-x-auto rounded-2xl border border-white/10">
            <table class="min-w-full divide-y divide-white/10 text-left text-sm">
                <thead class="bg-white/5 text-xs font-semibold uppercase tracking-wide text-slate-300">
                    <tr>
                        <th class="px-4 py-3">Capability</th>
                        <th class="px-4 py-3 text-emerald-300">Quantyra Labs IDX API</th>
                        <th class="px-4 py-3">Typical legacy IDX</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-white/10 text-slate-200">
                    <tr>
                        <td class="px-4 py-3 font-medium text-white">Stellar MLS data fee posture</td>
                        <td class="px-4 py-3 text-emerald-200">$0 data license fee (signed Consultant)</td>
                        <td class="px-4 py-3">Often bundled + opaque pass-throughs</td>
                    </tr>
                    <tr>
                        <td class="px-4 py-3 font-medium text-white">Edge performance</td>
                        <td class="px-4 py-3 text-emerald-200">Octane + FrankenPHP, sub-100ms proxy targets</td>
                        <td class="px-4 py-3">Monolithic PHP / cold boots</td>
                    </tr>
                    <tr>
                        <td class="px-4 py-3 font-medium text-white">Lead gating</td>
                        <td class="px-4 py-3 text-emerald-200">Hard gating: 3 listings → email + phone OTP (TextGrid)</td>
                        <td class="px-4 py-3">Soft forms / easy to fake emails</td>
                    </tr>
                    <tr>
                        <td class="px-4 py-3 font-medium text-white">Go-to-market surfaces</td>
                        <td class="px-4 py-3 text-emerald-200">JS widgets + LeadConnector embedded app + REST</td>
                        <td class="px-4 py-3">Often iframe-only or API bolt-ons</td>
                    </tr>
                </tbody>
            </table>
        </div>
        <ul class="mt-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-3 text-sm text-slate-200">
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span><strong class="text-white">$0 Stellar MLS data fee</strong> (we are the official signed Consultant)</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>Centralized Laravel Octane + FrankenPHP proxy API (sub-100ms responses)</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>Built-in <strong class="text-white">JS Embed Widgets</strong> (search bar, Leaflet map, listing cards, gallery, lead forms with teaser gating)</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>Official <strong class="text-white">LeadConnector / GoHighLevel</strong> embedded app</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>Full REST API for custom websites (perfect for deeper programmatic SEO)</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>Hard lead gating (3 listings → email + phone OTP via TextGrid)</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>Image proxy served from NVMe SSDs (no hotlinking)</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>Leaflet + OpenStreetMap interactive maps</span></li>
            <li class="flex gap-2"><span class="text-emerald-400">✓</span> <span>PostgreSQL 15-minute freshness cache</span></li>
        </ul>
    </section>

    {{-- Revenue Impact: Embed demos shorten “time to first widget” in trials. --}}
    <section id="widgets" class="border-y border-white/10 py-14">
        <div class="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <h2 class="text-3xl font-bold tracking-tight">JS embed widgets (search + map + cards)</h2>
            <p class="mt-3 max-w-3xl text-sm text-slate-300">
                Drop-in script tags for authorized domains — teaser galleries, OTP-gated detail, and map-first discovery.
            </p>
            <div class="mt-8 grid gap-6 lg:grid-cols-2">
                <div class="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
                    <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Example loader</p>
                    <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-relaxed text-slate-200"><code>&lt;script
  src="{{ rtrim(config('app.url'), '/') }}/widgets/idx-loader.js"
  data-quantyra-site-key="YOUR_SITE_KEY"
  data-quantyra-widget="search-bar"
  async
&gt;&lt;/script&gt;</code></pre>
                </div>
                <div class="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
                    <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">API docs teaser</p>
                    <p class="mt-3 text-sm text-slate-200">
                        Authenticate with Sanctum-issued tokens or domain allow-lists, then pull normalized listing JSON, media references, and geography helpers for programmatic SEO pages.
                    </p>
                    <a
                        href="#pricing"
                        class="mt-5 inline-flex rounded-full border border-blue-400/50 px-4 py-2 text-sm font-semibold text-blue-100 hover:bg-blue-500/10"
                    >
                        Unlock API keys on Ultra+
                    </a>
                </div>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Realtyna-style pricing grid maximizes plan comparison + cart clicks. --}}
    <section id="pricing" class="bg-slate-100 py-14 text-slate-900">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
            <div class="text-center">
                <h2 class="text-3xl font-black tracking-tight text-slate-900 sm:text-4xl">IDX API subscriptions</h2>
                <p class="mx-auto mt-3 max-w-2xl text-sm text-slate-600">
                    14-day free trial + money-back guarantee • Overage billing $0.001 per extra API call •
                    Add-ons: extra domains $19/mo • priority support $49/mo
                </p>
            </div>

            <div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
                <span class="text-sm font-semibold text-slate-700">Billing</span>
                <div class="inline-flex rounded-full border border-slate-300 bg-white p-1 shadow-sm">
                    <button
                        type="button"
                        wire:click="setBillingInterval('monthly')"
                        @class([
                            'rounded-full px-4 py-2 text-sm font-semibold transition',
                            'bg-slate-900 text-white' => $billingInterval === 'monthly',
                            'text-slate-600 hover:text-slate-900' => $billingInterval !== 'monthly',
                        ])
                    >
                        Monthly
                    </button>
                    <button
                        type="button"
                        wire:click="setBillingInterval('annual')"
                        @class([
                            'rounded-full px-4 py-2 text-sm font-semibold transition',
                            'bg-slate-900 text-white' => $billingInterval === 'annual',
                            'text-slate-600 hover:text-slate-900' => $billingInterval !== 'annual',
                        ])
                    >
                        Annual (20% off)
                    </button>
                </div>
            </div>

            <div class="mt-6 rounded-xl bg-blue-600 px-4 py-3 text-center text-sm font-bold uppercase tracking-wide text-white shadow-md">
                Annual Subscriptions with 20% Discount
            </div>

            <div class="mt-8 grid gap-6 lg:grid-cols-4">
                @foreach ($plans as $plan)
                    <article class="flex flex-col rounded-xl border border-slate-200 bg-white shadow-sm">
                        <div class="border-b border-slate-100 px-5 py-4">
                            <h3 class="text-lg font-black text-slate-900">{{ $plan['label'] }}</h3>
                            <p class="mt-1 text-xs font-medium uppercase tracking-wide text-slate-500">{{ $plan['best_for'] }}</p>
                            <div class="mt-4">
                                @if ($billingInterval === 'monthly')
                                    <p class="text-3xl font-black text-slate-900">{{ $plan['monthly_display'] }}<span class="text-base font-semibold text-slate-500">/mo</span></p>
                                    <p class="mt-1 text-xs text-slate-500">Billed monthly</p>
                                @else
                                    <p class="text-3xl font-black text-slate-900">{{ $plan['annual_display'] }}<span class="text-base font-semibold text-slate-500">/yr</span></p>
                                    <p class="mt-1 text-xs font-semibold text-emerald-700">{{ $plan['annual_note'] }}</p>
                                @endif
                            </div>
                            <p class="mt-4 rounded-lg bg-blue-50 px-3 py-2 text-center text-xs font-semibold text-blue-900">
                                Your market already generated {{ number_format($teaserLeads) }} leads this month
                            </p>
                        </div>
                        <ul class="flex-1 space-y-2 px-5 py-4 text-sm text-slate-700">
                            @foreach ($plan['features'] as $feature)
                                <li class="flex gap-2">
                                    <span class="mt-0.5 text-emerald-600" aria-hidden="true">
                                        <svg class="size-4 shrink-0" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.704 5.29a1 1 0 010 1.42l-7.995 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.42l3.293 3.3 7.288-7.3a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>
                                    </span>
                                    <span>{{ $feature }}</span>
                                </li>
                            @endforeach
                        </ul>
                        <div class="px-5 pb-5">
                            <a
                                href="{{ route('billing.checkout', ['plan' => $plan['key'], 'interval' => $billingInterval]) }}"
                                class="flex w-full items-center justify-center rounded-md bg-orange-600 px-4 py-3 text-sm font-bold uppercase tracking-wide text-white shadow hover:bg-orange-500"
                            >
                                Add to cart
                            </a>
                            <p class="mt-2 text-center text-[11px] text-slate-500">
                                Secured by Stripe Checkout • 14-day trial
                            </p>
                        </div>
                    </article>
                @endforeach
            </div>

            <div class="mx-auto mt-10 max-w-3xl rounded-xl border border-slate-200 bg-white px-5 py-4 text-center text-xs text-slate-600">
                <strong class="text-slate-900">Overage:</strong> $0.001 per additional API call beyond plan buckets.
                <span class="mx-2">•</span>
                <strong class="text-slate-900">Add-ons:</strong> Extra domains $19/mo · Priority support $49/mo
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Pilot-market previews increase trust without compliance risk. --}}
    <section class="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
        <h2 class="text-3xl font-bold tracking-tight">How geographic packaging looks for agents</h2>
        <p class="mt-3 text-sm text-slate-200">
            Illustrative positioning only — no live MLS listings, boundary files, or map tiles are loaded on this marketing page.
        </p>
        <div class="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            @foreach ($illustrativeMarkets as $row)
                <article class="rounded-2xl border border-white/10 p-4">
                    <div class="h-24 rounded-xl bg-gradient-to-br from-cyan-500/40 to-emerald-400/20"></div>
                    <p class="mt-3 text-sm font-semibold text-slate-100">{{ $row['label'] }}</p>
                    <p class="mt-1 text-xs leading-relaxed text-slate-200">{{ $row['geography'] }}</p>
                </article>
            @endforeach
        </div>
    </section>

    {{-- Revenue Impact: FAQ removes objections late in the buying decision. --}}
    <section class="mx-auto max-w-4xl px-4 py-14 sm:px-6 lg:px-8">
        <h2 class="text-3xl font-bold tracking-tight">FAQ</h2>
        <div class="mt-8 space-y-3">
            @foreach ($faqs as $index => $faq)
                <article class="rounded-2xl border border-white/10">
                    <button
                        type="button"
                        class="flex w-full items-center justify-between px-5 py-4 text-left"
                        @click="openFaq = openFaq === {{ $index }} ? null : {{ $index }}"
                    >
                        <span class="font-medium">{{ $faq['question'] }}</span>
                        <span class="text-cyan-300" x-text="openFaq === {{ $index }} ? '-' : '+'"></span>
                    </button>
                    <p x-show="openFaq === {{ $index }}" x-transition class="px-5 pb-5 text-sm text-slate-200">
                        {{ $faq['answer'] }}
                    </p>
                </article>
            @endforeach
        </div>
    </section>

    @if ($showLoginModal)
        <div
            class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
            aria-modal="true"
            role="dialog"
            aria-labelledby="login-modal-title"
        >
            <div class="absolute inset-0 bg-slate-950/80" wire:click="closeLoginModal"></div>

            <div class="relative w-full max-w-md overflow-hidden rounded-2xl border border-white/10 bg-slate-900 shadow-2xl">
                <div class="flex items-start justify-between gap-4 border-b border-white/10 px-6 py-4">
                    <h2 id="login-modal-title" class="text-lg font-bold tracking-tight text-slate-100">Subscriber login</h2>
                    <button
                        type="button"
                        class="rounded-lg p-1.5 text-slate-300 hover:bg-white/10 hover:text-white focus:outline-none focus:ring-2 focus:ring-emerald-400/50"
                        wire:click="closeLoginModal"
                        aria-label="Close login dialog"
                    >
                        <span aria-hidden="true" class="text-xl leading-none">&times;</span>
                    </button>
                </div>

                <div class="px-6 py-6">
                    <x-auth.login-form :autofocus-email="true" :show-intro="false" />

                    <p class="mt-6 border-t border-white/10 pt-4 text-center text-xs text-slate-300">
                        Prefer a dedicated page?
                        <a href="{{ route('login') }}" class="font-medium text-emerald-300 underline decoration-emerald-400/50 hover:text-emerald-200">
                            Open full login
                        </a>
                        @if (Route::has('register'))
                            <span class="mx-1">·</span>
                            <a href="{{ route('register') }}" class="font-medium text-emerald-300 underline decoration-emerald-400/50 hover:text-emerald-200">
                                Create account
                            </a>
                        @endif
                    </p>
                </div>
            </div>
        </div>
    @endif

    <footer class="border-t border-white/10 py-8">
        <div class="mx-auto flex max-w-6xl flex-col gap-4 px-4 text-xs text-slate-300 sm:px-6 lg:px-8 md:flex-row md:items-center md:justify-between">
            <p>
                Compliance: This is a marketing page only. No live Stellar MLS listing data is displayed here. IDX data appears exclusively on authorized domains per signed Stellar MLS Participant Data Access Agreement (Consultant: Quantyra Labs LLC; Firm: Realty Of America, LLC).
            </p>
            <div class="flex flex-wrap gap-4">
                <button type="button" wire:click="openLoginModal" class="text-left hover:text-white">Subscriber login</button>
                <a href="#" class="hover:text-white">Privacy</a>
                <a href="#" class="hover:text-white">Terms</a>
            </div>
        </div>
    </footer>
</div>
