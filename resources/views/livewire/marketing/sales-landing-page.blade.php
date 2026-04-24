<div
    x-data="{ openFaq: 0 }"
    class="bg-slate-950 text-slate-100 transition-colors duration-300"
    @keydown.escape.window="$wire.closeLoginModal()"
>
    <header class="sticky top-0 z-30 border-b border-white/15 bg-slate-950/95 backdrop-blur supports-[backdrop-filter]:bg-slate-950/80">
        <div class="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-6 sm:py-4 lg:px-8">
            <a href="/" class="text-base font-semibold tracking-tight text-white sm:text-lg">GeoIDX by Quantyra Labs</a>
            <nav class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:justify-end" aria-label="Account and pricing">
                @if (Route::has('register'))
                    <a
                        href="{{ route('register') }}"
                        class="inline-flex min-h-11 min-w-0 items-center justify-center rounded-full border border-white/30 px-4 py-2.5 text-sm font-semibold text-white hover:border-white/50 hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                    >
                        Create account
                    </a>
                @endif
                <button
                    type="button"
                    wire:click="openLoginModal"
                    class="inline-flex min-h-11 min-w-0 items-center justify-center rounded-full border border-white/30 px-4 py-2.5 text-sm font-semibold text-white hover:border-white/50 hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                >
                    Subscriber login
                </button>
                <a
                    href="#pricing"
                    class="inline-flex min-h-11 min-w-0 items-center justify-center rounded-full bg-emerald-400 px-4 py-2.5 text-sm font-semibold text-slate-950 hover:bg-emerald-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-300 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                >
                    See pricing
                </a>
            </nav>
        </div>
    </header>

    <main id="main-content" tabindex="-1" class="outline-none">

        @if (session('flash_billing_error'))
            <div class="border-b border-amber-400/50 bg-amber-950/50 px-4 py-3 text-center text-sm font-medium text-amber-50" role="status">
                {{ session('flash_billing_error') }}
            </div>
        @endif

        @if (request()->query('checkout') === 'success')
            <div class="border-b border-emerald-400/50 bg-emerald-950/40 px-4 py-3 text-center text-sm font-medium text-emerald-50" role="status">
                Thanks — Stripe is finalizing your subscription. Refresh the dashboard in a moment if access is still pending.
            </div>
        @elseif (request()->query('checkout') === 'cancelled')
            <div class="border-b border-slate-500/60 bg-slate-900 px-4 py-3 text-center text-sm font-medium text-slate-100" role="status">
                Checkout cancelled. Your card was not charged.
            </div>
        @endif

    {{-- Revenue Impact: Light hero band lifts headline contrast + centers conversion on GeoIDX brand. --}}
    <section class="border-b border-slate-200 bg-slate-50 px-4 pb-14 pt-12 sm:px-6 sm:pb-16 sm:pt-16 lg:px-8" aria-labelledby="hero-heading">
        <div class="mx-auto max-w-5xl text-center">
            <h1 id="hero-heading" class="text-4xl font-bold tracking-tight text-slate-900 sm:text-5xl md:text-6xl">
                GeoIDX by Quantyra Labs
            </h1>
            <p class="mx-auto mt-4 max-w-3xl text-xl text-slate-700 sm:text-2xl md:text-3xl">
                Multi-MLS IDX platform<br>
                Live MLS Data Proxy + Image Proxy + JS Embed Widgets + LeadConnector App
            </p>
            <p class="mx-auto mt-6 max-w-2xl text-lg text-slate-600 sm:text-xl">
                The fastest, most lead-generating IDX solution for Stellar MLS. Drop-in JS widgets or full REST API.
            </p>
            <div class="mt-10 flex flex-col items-stretch justify-center gap-3 sm:flex-row sm:items-center sm:gap-4">
                <a
                    href="#pricing"
                    class="inline-flex min-h-12 items-center justify-center rounded-full bg-orange-600 px-8 py-4 text-lg font-semibold text-white shadow-md hover:bg-orange-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-50 sm:px-10"
                >
                    Get Started for $39/mo
                </a>
                <a
                    href="#demo"
                    class="inline-flex min-h-12 items-center justify-center rounded-full border-2 border-slate-300 bg-white px-8 py-4 text-lg font-semibold text-slate-800 shadow-sm hover:border-slate-400 hover:bg-slate-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-50 sm:px-10"
                >
                    Watch 47-second Demo
                </a>
            </div>
            <p class="mt-6 text-center text-sm text-slate-500">14-day free trial • Cancel anytime • No contracts</p>
        </div>
    </section>

    {{-- Revenue Impact: Trust bar answers procurement objections in one scan. --}}
    <div class="border-b border-slate-200 bg-white py-4">
        <div class="mx-auto flex max-w-screen-2xl flex-wrap items-center justify-center gap-x-10 gap-y-4 px-4 text-sm text-slate-800 sm:gap-x-12 sm:px-6">
            <div class="flex items-center gap-2">
                <span class="text-emerald-600" aria-hidden="true">✓</span>
                <strong>Official Stellar MLS Consultant</strong> (agreement signed Apr&nbsp;17,&nbsp;2026)
            </div>
            <div class="flex items-center gap-2">
                <span class="text-emerald-600" aria-hidden="true">✓</span>
                <strong>$0 Data License Fee</strong> through our Stellar consultant program
            </div>
            <div class="flex items-center gap-2">
                <span class="text-emerald-600" aria-hidden="true">✓</span>
                <strong>Unlimited JS Widget usage</strong> on Pro &amp; Smart
            </div>
            <div class="flex items-center gap-2">
                <span class="text-emerald-600" aria-hidden="true">✓</span>
                <strong>Hard lead gating + OTP verification</strong> built in
            </div>
            <div class="flex items-center gap-2">
                <span class="text-emerald-600" aria-hidden="true">✓</span>
                <strong>LeadConnector / GoHighLevel embedded app</strong>
            </div>
        </div>
    </div>

    {{-- Revenue Impact: Competitive cards beat long comparison tables on mobile scroll depth. --}}
    <section class="border-b border-slate-200 bg-slate-50 px-4 py-14 sm:px-6 lg:px-8" aria-labelledby="why-geo-heading">
        <h2 id="why-geo-heading" class="mx-auto mb-10 max-w-screen-2xl text-center text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
            Why serious agents, developers &amp; brokerages choose GeoIDX
        </h2>
        <div class="mx-auto grid max-w-screen-2xl gap-8 md:grid-cols-2 lg:grid-cols-3">
            <div class="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
                <h3 class="mb-3 text-xl font-semibold text-slate-900">Official Stellar Coverage. Real Compliance.</h3>
                <p class="text-slate-600">GeoIDX runs under a signed Stellar MLS consultant agreement with policy-ready controls and transparent subscriber economics.</p>
            </div>
            <div class="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
                <h3 class="mb-3 text-xl font-semibold text-slate-900">JS Widgets That Actually Convert</h3>
                <p class="text-slate-600">One-line embed: search bar, interactive map, listing gallery, lead forms. Teaser mode shows only 3 listings → hard gate → email + phone OTP. Leads flow straight into your CRM.</p>
            </div>
            <div class="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
                <h3 class="mb-3 text-xl font-semibold text-slate-900">Unlimited Widgets on Lower Tiers</h3>
                <p class="text-slate-600">Pro and Smart include completely unlimited JS Widget usage so agents can deploy as many lead funnels as needed without monthly API call caps.</p>
            </div>
            <div class="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
                <h3 class="mb-3 text-xl font-semibold text-slate-900">LeadConnector / GHL Embedded App</h3>
                <p class="text-slate-600">One-click install. Verified buyer &amp; seller leads land directly in your GoHighLevel funnels with full journey data.</p>
            </div>
            <div class="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
                <h3 class="mb-3 text-xl font-semibold text-slate-900">Blazing Fast &amp; Future-Proof</h3>
                <p class="text-slate-600">Low-latency data and image proxy architecture keeps search and listing experiences responsive while lead forms stay conversion-focused.</p>
            </div>
            <div class="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
                <h3 class="mb-3 text-xl font-semibold text-slate-900">Revenue-First Lead Gating</h3>
                <p class="text-slate-600">
                    Lead capture forms are woven into search, maps, and listing views so visitors convert when interest is highest—not after a frustrating wall of text.
                </p>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Demo anchor holds hero secondary CTA; embed replaces placeholder when asset ships. --}}
    <section id="demo" class="scroll-mt-24 border-b border-white/10 bg-slate-900/60 py-14" aria-labelledby="demo-heading">
        <div class="mx-auto max-w-3xl px-4 text-center sm:px-6">
            <h2 id="demo-heading" class="text-2xl font-bold tracking-tight text-white sm:text-3xl">47-second product walkthrough</h2>
            <p class="mt-4 text-base text-slate-300 sm:text-lg">
                Hosted demo video embeds here. Until the cut is live, jump straight to plans or sign in to explore widgets in your sandbox.
            </p>
            <div class="mt-8 flex flex-col items-stretch justify-center gap-3 sm:flex-row sm:justify-center">
                <a
                    href="#pricing"
                    class="inline-flex min-h-12 items-center justify-center rounded-full bg-orange-500 px-8 py-3 font-semibold text-white hover:bg-orange-400 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900"
                >
                    View pricing
                </a>
                <button
                    type="button"
                    wire:click="openLoginModal"
                    class="inline-flex min-h-12 items-center justify-center rounded-full border border-white/30 px-8 py-3 font-semibold text-white hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900"
                >
                    Subscriber login
                </button>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Embed demos shorten “time to first widget” in trials. --}}
    <section id="widgets" class="scroll-mt-24 border-y border-white/10 py-14">
        <div class="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <h2 class="text-3xl font-bold tracking-tight">JS embed widgets (search + map + cards)</h2>
            <p class="mt-3 max-w-3xl text-sm text-slate-300">
                Drop-in script tags for authorized domains — teaser galleries, OTP-gated detail, and map-first discovery.
            </p>
            <div class="mt-8 grid gap-6 lg:grid-cols-2">
                <div class="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
                    <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Example loader</p>
                    <pre
                        class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-relaxed text-slate-200"
                        tabindex="0"
                        aria-label="Example script tag for the IDX widget loader"
                    ><code>&lt;script
  src="{{ rtrim(config('app.url'), '/') }}/widgets/idx-loader.js"
  data-quantyra-site-key="YOUR_SITE_KEY"
  data-quantyra-widget="search-bar"
  async
&gt;&lt;/script&gt;</code></pre>
                </div>
                <div class="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
                    <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">API docs teaser</p>
                    <p class="mt-3 text-sm text-slate-200">
                        Authenticate with API tokens or domain allow-lists, then pull normalized listing payloads, media references, and geography helpers for programmatic SEO pages.
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
    <section id="pricing" class="idx-pricing-band scroll-mt-24 py-14" aria-labelledby="pricing-heading">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
            <div class="text-center">
                <div class="mt-2">
                    <span class="inline-flex items-center gap-2 rounded-3xl bg-blue-100 px-5 py-2 text-sm font-medium text-blue-800 sm:px-6">
                        <span class="text-2xl" role="img" aria-label="Savings">💰</span>
                        Annual Subscriptions with <span class="font-bold text-orange-600">20% Discount</span>
                    </span>
                </div>
                <h2 id="pricing-heading" class="mt-6 text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">Simple, Transparent Pricing</h2>
                <p class="mx-auto mt-3 max-w-xl text-lg text-slate-700 sm:text-xl">
                    Choose the plan that matches your volume and tech needs. Pro &amp; Smart plans include completely unlimited JS Widget usage. No monthly API call limits.
                </p>
                <p class="mx-auto mt-4 max-w-2xl text-sm text-slate-600">
                    14-day free trial + money-back guarantee • Overage billing (Ultra &amp; Mega only): $0.001 per extra API call • Soft rate limits (60-100 req/sec/domain) + fair-usage policy on all plans
                </p>
            </div>

            <div class="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
                <span id="billing-toggle-label" class="text-sm font-semibold text-slate-800">Billing</span>
                <div
                    class="inline-flex rounded-full border border-slate-300 bg-white p-1 shadow-sm"
                    role="group"
                    aria-labelledby="billing-toggle-label"
                >
                    <button
                        type="button"
                        wire:click="setBillingInterval('monthly')"
                        @class([
                            'min-h-11 min-w-[7.5rem] rounded-full px-4 py-2.5 text-sm font-semibold transition focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600 focus-visible:ring-offset-2',
                            'bg-slate-900 text-white' => $billingInterval === 'monthly',
                            'text-slate-700 hover:bg-slate-100 hover:text-slate-900' => $billingInterval !== 'monthly',
                        ])
                        aria-pressed="{{ $billingInterval === 'monthly' ? 'true' : 'false' }}"
                    >
                        Monthly
                    </button>
                    <button
                        type="button"
                        wire:click="setBillingInterval('annual')"
                        @class([
                            'min-h-11 min-w-[7.5rem] rounded-full px-4 py-2.5 text-sm font-semibold transition focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600 focus-visible:ring-offset-2',
                            'bg-slate-900 text-white' => $billingInterval === 'annual',
                            'text-slate-700 hover:bg-slate-100 hover:text-slate-900' => $billingInterval !== 'annual',
                        ])
                        aria-pressed="{{ $billingInterval === 'annual' ? 'true' : 'false' }}"
                    >
                        Annual (20% off)
                    </button>
                </div>
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
                                class="flex min-h-11 w-full items-center justify-center rounded-md bg-orange-700 px-4 py-3 text-sm font-bold uppercase tracking-wide text-white shadow hover:bg-orange-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300 focus-visible:ring-offset-2 focus-visible:ring-offset-white"
                                aria-label="Add {{ $plan['label'] }} plan to cart, {{ $billingInterval }} billing, checkout with Stripe"
                            >
                                Add to cart
                            </a>
                            <p class="mt-2 text-center text-xs text-slate-600">
                                Secured by Stripe Checkout • 14-day trial
                            </p>
                        </div>
                    </article>
                @endforeach
            </div>

            <div class="mx-auto mt-10 max-w-3xl rounded-xl border border-slate-200 bg-white px-5 py-4 text-center text-xs text-slate-600">
                <strong class="text-slate-900">Overage (Ultra &amp; Mega only):</strong> $0.001 per additional API call.
                <span class="mx-2">•</span>
                <strong class="text-slate-900">Policy controls:</strong> Soft rate limits (60-100 req/sec/domain) + fair-usage protection
            </div>
        </div>
    </section>

    {{-- Revenue Impact: FAQ removes objections late in the buying decision. --}}
    <section class="mx-auto max-w-4xl px-4 py-14 sm:px-6 lg:px-8" aria-labelledby="faq-heading">
        <h2 id="faq-heading" class="text-3xl font-bold tracking-tight">FAQ</h2>
        <div class="mt-8 space-y-3">
            @foreach ($faqs as $index => $faq)
                <article class="rounded-2xl border border-white/10">
                    <button
                        type="button"
                        id="faq-trigger-{{ $index }}"
                        class="flex min-h-12 w-full items-center justify-between gap-3 px-5 py-4 text-left text-base font-medium text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                        @click="openFaq = openFaq === {{ $index }} ? null : {{ $index }}"
                        x-bind:aria-expanded="openFaq === {{ $index }}"
                        aria-controls="faq-panel-{{ $index }}"
                    >
                        <span>{{ $faq['question'] }}</span>
                        <span class="shrink-0 text-cyan-300" aria-hidden="true" x-text="openFaq === {{ $index }} ? '−' : '+'"></span>
                    </button>
                    <div
                        id="faq-panel-{{ $index }}"
                        role="region"
                        x-show="openFaq === {{ $index }}"
                        x-transition
                        class="px-5 pb-5 text-sm leading-relaxed text-slate-200"
                    >
                        {{ $faq['answer'] }}
                    </div>
                </article>
            @endforeach
        </div>
    </section>

    {{-- Revenue Impact: MLS compliance story closes enterprise deals + lowers legal anxiety. --}}
    <section class="mx-auto mt-12 max-w-screen-2xl scroll-mt-24 rounded-3xl bg-slate-100 px-4 py-12 sm:px-6 lg:px-8" aria-labelledby="compliance-heading">
        <div class="flex flex-col items-center gap-10 md:flex-row md:items-start md:gap-12">
            <div class="min-w-0 flex-1">
                <h2 id="compliance-heading" class="text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl">Official Stellar MLS Consultant badge &amp; compliance proof</h2>
                <p class="mt-4 text-base leading-relaxed text-slate-700 sm:text-lg">
                    Quantyra Labs LLC is an <strong class="font-medium text-slate-900">Official Stellar MLS Consultant</strong> under a signed agreement effective April 17, 2026. GeoIDX enforces required disclaimers, approved-domain controls, and audit logging so subscriber deployments stay inside Stellar policy.
                </p>
                <p class="mt-6 text-base font-medium text-emerald-700">
                    ✓ Signed consultant agreement on file • ✓ IDX only on approved domains • ✓ Policy-ready attribution, disclaimers, and audit trail
                </p>
            </div>
            <div class="flex w-full max-w-md flex-1 flex-col items-center justify-center rounded-2xl border border-emerald-200 bg-white p-8 text-center text-sm text-slate-600 shadow-sm">
                <span class="inline-flex rounded-full bg-emerald-100 px-4 py-1 text-xs font-semibold tracking-wide text-emerald-800">Official Stellar MLS Consultant</span>
                <p class="mt-4 text-base font-semibold text-slate-900">Agreement Signed: April 17, 2026</p>
                <p class="mt-2 text-xs text-slate-500">Quantyra Labs LLC • Consultant of record for compliant Stellar IDX delivery.</p>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Terminal CTA captures scroll exhausters who skipped mid-page checkout. --}}
    <section class="mx-auto mt-12 max-w-6xl px-4 pb-6 sm:px-6 lg:px-8" aria-labelledby="final-cta-heading">
        <div class="rounded-3xl bg-gradient-to-br from-blue-600 to-indigo-800 px-6 py-16 text-center text-white sm:px-10 sm:py-20">
            <h2 id="final-cta-heading" class="text-3xl font-bold tracking-tight sm:text-4xl">Ready to dominate your market with GeoIDX?</h2>
            <p class="mt-4 text-xl sm:text-2xl">Start your 14-day free trial today. No credit card required.</p>
            <a
                href="#pricing"
                class="mt-8 inline-flex min-h-14 items-center justify-center rounded-2xl bg-white px-10 py-4 text-lg font-semibold text-indigo-800 transition-colors hover:bg-amber-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-indigo-800 sm:px-14 sm:text-xl"
            >
                Claim Your Plan →
            </a>
        </div>
    </section>

    </main>

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
        <div class="mx-auto flex max-w-6xl flex-col gap-4 px-4 text-xs text-slate-200 sm:px-6 lg:px-8 md:flex-row md:items-center md:justify-between">
            <p>
                Compliance: This is a marketing page only. No live MLS listing data is displayed here. GeoIDX data is delivered only on approved domains under the signed Stellar MLS consultant agreement and applicable subscriber terms.
            </p>
            <nav class="flex flex-wrap gap-4" aria-label="Footer">
                <button
                    type="button"
                    wire:click="openLoginModal"
                    class="min-h-11 text-left text-slate-200 underline decoration-white/30 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50"
                >
                    Subscriber login
                </button>
                <a href="#" class="min-h-11 inline-flex items-center text-slate-200 underline decoration-white/30 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50" aria-label="Privacy policy (coming soon)">Privacy</a>
                <a href="#" class="min-h-11 inline-flex items-center text-slate-200 underline decoration-white/30 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50" aria-label="Terms of service (coming soon)">Terms</a>
            </nav>
        </div>
    </footer>
</div>
