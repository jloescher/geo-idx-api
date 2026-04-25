<div
    x-data="{ openFaq: 0 }"
    class="bg-slate-950 text-slate-100 transition-colors duration-300"
    @keydown.escape.window="$wire.closeLoginModal()"
>
    <header class="sticky top-0 z-30 border-b border-white/15 bg-slate-950/95 backdrop-blur supports-[backdrop-filter]:bg-slate-950/80">
        <div class="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-6 sm:py-4 lg:px-8">
            <a href="/" class="text-base font-semibold tracking-tight text-white sm:text-lg">GeoIDX by Quantyra Labs</a>
            <nav class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:justify-end" aria-label="Account and pricing">
                @guest
                    @if (Route::has('register'))
                        <a
                            href="{{ route('register', [], false) }}"
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
                @else
                    <a
                        href="{{ route('dashboard.index', [], false) }}"
                        class="inline-flex min-h-11 min-w-0 items-center justify-center rounded-full border border-white/30 px-4 py-2.5 text-sm font-semibold text-white hover:border-white/50 hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                    >
                        Dashboard
                    </a>
                    <form method="POST" action="{{ route('logout', [], false) }}">
                        @csrf
                        <button
                            type="submit"
                            class="inline-flex min-h-11 min-w-0 items-center justify-center rounded-full border border-white/30 px-4 py-2.5 text-sm font-semibold text-white hover:border-white/50 hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                        >
                            Subscriber logout
                        </button>
                    </form>
                @endguest
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

    {{-- Revenue Impact: Hero clarifies demo-to-live policy and accelerates qualified starts. --}}
    <section class="border-b border-slate-800 bg-slate-950 px-4 pb-14 pt-12 sm:px-6 sm:pb-16 sm:pt-16 lg:px-8" aria-labelledby="hero-heading">
        <div class="mx-auto grid max-w-6xl gap-10 lg:grid-cols-2 lg:items-center">
            <div>
                <p class="inline-flex rounded-full border border-cyan-300/40 bg-cyan-400/10 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-cyan-200">
                    Multi-MLS widgets for GHL teams
                </p>
                <h1 id="hero-heading" class="mt-5 text-4xl font-bold tracking-tight text-white sm:text-5xl md:text-6xl">
                    Geography-first IDX widgets that convert faster.
                </h1>
                <p class="mt-5 max-w-2xl text-lg text-slate-200 sm:text-xl">
                    Launch embeddable search, map, and lead-capture widgets in a demo sandbox, then activate paid live data when your funnels are ready.
                </p>
                <p class="mt-4 text-sm font-medium text-amber-200">
                    14-day trial includes demo data only. Paid plan activation is required for live data.
                </p>
                <div class="mt-8 flex flex-col items-stretch gap-3 sm:flex-row sm:items-center sm:gap-4">
                    <a
                        href="{{ route('register', [], false) }}"
                        class="inline-flex min-h-12 items-center justify-center rounded-full bg-cyan-400 px-7 py-3 text-base font-semibold text-slate-950 shadow-md hover:bg-cyan-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-cyan-300 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                        data-event-name="demo_trial_started"
                    >
                        Start Demo Trial
                    </a>
                    <a
                        href="#pricing"
                        class="inline-flex min-h-12 items-center justify-center rounded-full border border-white/25 px-7 py-3 text-base font-semibold text-white hover:border-white/40 hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950"
                        data-event-name="live_data_activation_clicked"
                    >
                        Activate Live Data
                    </a>
                </div>
            </div>

            <div class="rounded-3xl border border-white/10 bg-slate-900/70 p-6 shadow-2xl">
                <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Conversion stack preview</p>
                <div class="mt-4 grid gap-3">
                    <div class="rounded-xl border border-white/10 bg-slate-950/60 p-4">
                        <p class="text-sm font-semibold text-white">Geography-aware search</p>
                        <p class="mt-1 text-sm text-slate-300">Map and boundary context that aligns discovery with local buyer intent.</p>
                    </div>
                    <div class="rounded-xl border border-white/10 bg-slate-950/60 p-4">
                        <p class="text-sm font-semibold text-white">GHL-ready lead capture</p>
                        <p class="mt-1 text-sm text-slate-300">Teaser flow, verification, and routing patterns designed for funnel-first operations.</p>
                    </div>
                    <div class="rounded-xl border border-white/10 bg-slate-950/60 p-4">
                        <p class="text-sm font-semibold text-white">Market-intel overlays</p>
                        <p class="mt-1 text-sm text-slate-300">Context panels that help prospects evaluate movement and timing with confidence.</p>
                    </div>
                </div>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Proof chips lower trust friction before pricing scroll. --}}
    <section class="border-b border-slate-800 bg-slate-900/70 px-4 py-6 sm:px-6 lg:px-8">
        <div class="mx-auto flex max-w-6xl flex-wrap items-center justify-center gap-3">
            @foreach ($proofChips as $chip)
                <div class="inline-flex items-center rounded-full border border-white/15 bg-white/5 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slate-200">
                    {{ $chip['label'] }}
                </div>
            @endforeach
        </div>
    </section>

    <section class="border-b border-slate-800 bg-slate-950 px-4 py-14 sm:px-6 lg:px-8" aria-labelledby="why-geo-heading">
        <h2 id="why-geo-heading" class="mx-auto max-w-4xl text-center text-3xl font-semibold tracking-tight text-white sm:text-4xl">
            Built for geography-led discovery and funnel-ready conversion.
        </h2>
        <p class="mx-auto mt-4 max-w-3xl text-center text-sm text-slate-300 sm:text-base">
            GeoIDX keeps low-tier entry simple for GHL teams while preserving a clear path to paid live data activation when your campaigns are ready.
        </p>
    </section>

    <section id="widgets" class="scroll-mt-24 border-y border-slate-800 bg-slate-900/40 py-14">
        <div class="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <h2 class="text-3xl font-bold tracking-tight text-white">Embeddable JS widgets for GHL-first teams</h2>
            <p class="mt-3 max-w-3xl text-sm text-slate-300">
                Deploy search, map, and lead-capture surfaces in minutes, validate performance in your demo sandbox, then switch to paid live data when ready.
            </p>
            <div class="mt-8 grid gap-6 lg:grid-cols-2">
                <div class="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
                    <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Widget loader example</p>
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
                    <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Live activation path</p>
                    <p class="mt-3 text-sm text-slate-200">
                        Start with demo data during trial to tune UX and funnel events. Activate a paid plan when you want live data on production domains.
                    </p>
                    <a
                        href="#pricing"
                        class="mt-5 inline-flex rounded-full border border-cyan-400/50 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-500/10"
                    >
                        Compare plans
                    </a>
                </div>
            </div>
        </div>
    </section>

    <section class="border-b border-slate-800 bg-slate-950 px-4 py-14 sm:px-6 lg:px-8" aria-labelledby="geography-heading">
        <div class="mx-auto max-w-6xl">
            <h2 id="geography-heading" class="text-3xl font-bold tracking-tight text-white">Geography intelligence that improves lead quality</h2>
            <div class="mt-8 grid gap-6 md:grid-cols-3">
                @foreach ($geographyHighlights as $highlight)
                    <article class="rounded-2xl border border-white/10 bg-slate-900/70 p-6">
                        <h3 class="text-lg font-semibold text-white">{{ $highlight['title'] }}</h3>
                        <p class="mt-3 text-sm leading-relaxed text-slate-300">{{ $highlight['description'] }}</p>
                    </article>
                @endforeach
            </div>
        </div>
    </section>

    <section class="border-b border-slate-800 bg-slate-900/50 px-4 py-14 sm:px-6 lg:px-8" aria-labelledby="market-intel-heading">
        <div class="mx-auto max-w-6xl">
            <h2 id="market-intel-heading" class="text-3xl font-bold tracking-tight text-white">Market-intelligence overlays as conversion proof points</h2>
            <div class="mt-8 grid gap-6 md:grid-cols-3">
                @foreach ($marketIntelHighlights as $highlight)
                    <article class="rounded-2xl border border-white/10 bg-slate-950/60 p-6">
                        <h3 class="text-lg font-semibold text-white">{{ $highlight['title'] }}</h3>
                        <p class="mt-3 text-sm leading-relaxed text-slate-300">{{ $highlight['description'] }}</p>
                    </article>
                @endforeach
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Pricing section clarifies demo-to-live activation and drives qualified checkout. --}}
    <section id="pricing" class="idx-pricing-band scroll-mt-24 py-14" aria-labelledby="pricing-heading">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
            <div class="text-center">
                <div class="mt-2">
                    <span class="inline-flex items-center gap-2 rounded-3xl bg-cyan-100 px-5 py-2 text-sm font-medium text-cyan-800 sm:px-6">
                        Annual subscriptions include <span class="font-bold text-cyan-950">20% savings</span>
                    </span>
                </div>
                <h2 id="pricing-heading" class="mt-6 text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">Simple, Transparent Pricing</h2>
                <p class="mx-auto mt-3 max-w-xl text-lg text-slate-700 sm:text-xl">
                    Pro and Smart are built for GHL-first widget funnels. Start with demo data, then activate paid live data when your campaigns are production-ready.
                </p>
                <p class="mx-auto mt-4 max-w-2xl text-sm text-slate-600">
                    14-day trial includes demo data only • Live data requires paid activation • Overage billing (Ultra &amp; Mega only): $0.001 per extra API call
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
                <a
                    href="#comparison-heading"
                    class="text-xs font-semibold uppercase tracking-wide text-slate-700 underline decoration-slate-400 underline-offset-2 hover:text-slate-900"
                    data-event-name="plan_upgrade_clicked"
                >
                    Compare Pro vs Smart
                </a>
            </div>

            <div class="mt-8 grid gap-6 lg:grid-cols-4">
                @foreach ($plans as $plan)
                    <article @class([
                        'flex flex-col rounded-xl border bg-white shadow-sm',
                        'border-cyan-300 ring-2 ring-cyan-200' => $plan['key'] === 'smart',
                        'border-slate-200' => $plan['key'] !== 'smart',
                    ])>
                        <div class="border-b border-slate-100 px-5 py-4">
                            <div class="flex items-center justify-between gap-3">
                                <h3 class="text-lg font-black text-slate-900">{{ $plan['label'] }}</h3>
                                @if ($plan['key'] === 'smart')
                                    <span class="rounded-full bg-cyan-600 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wide text-white">Most popular</span>
                                @endif
                            </div>
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
                            @if (in_array($plan['key'], ['pro', 'smart'], true))
                                <a
                                    href="{{ route('register', [], false) }}"
                                    class="flex min-h-11 w-full items-center justify-center rounded-md bg-cyan-600 px-4 py-3 text-sm font-bold uppercase tracking-wide text-white shadow hover:bg-cyan-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-cyan-300 focus-visible:ring-offset-2 focus-visible:ring-offset-white"
                                    data-event-name="demo_trial_started"
                                >
                                    Start Demo Trial
                                </a>
                                <a
                                    href="{{ route('billing.checkout', ['plan' => $plan['key'], 'interval' => $billingInterval], false) }}"
                                    class="mt-2 flex min-h-11 w-full items-center justify-center rounded-md border border-slate-300 bg-white px-4 py-3 text-sm font-bold uppercase tracking-wide text-slate-800 shadow-sm hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 focus-visible:ring-offset-2 focus-visible:ring-offset-white"
                                    aria-label="Activate live data on {{ $plan['label'] }} plan with {{ $billingInterval }} billing"
                                    data-event-name="live_data_activation_clicked"
                                >
                                    Activate Live Data
                                </a>
                                <p class="mt-2 text-center text-xs text-slate-600">
                                    Demo data during trial • Paid plan unlocks live data
                                </p>
                            @else
                                <a
                                    href="{{ route('billing.checkout', ['plan' => $plan['key'], 'interval' => $billingInterval], false) }}"
                                    class="flex min-h-11 w-full items-center justify-center rounded-md bg-slate-900 px-4 py-3 text-sm font-bold uppercase tracking-wide text-white shadow hover:bg-slate-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white"
                                    aria-label="Activate {{ $plan['label'] }} plan, {{ $billingInterval }} billing, checkout with Stripe"
                                    data-event-name="subscription_checkout_initiated"
                                >
                                    Activate {{ $plan['label'] }}
                                </a>
                                <p class="mt-2 text-center text-xs text-slate-600">
                                    Secured by Stripe Checkout • 14-day demo-data trial
                                </p>
                            @endif
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

    <section class="border-b border-slate-800 bg-slate-950 px-4 py-14 sm:px-6 lg:px-8" aria-labelledby="comparison-heading">
        <div class="mx-auto max-w-5xl">
            <h2 id="comparison-heading" class="text-3xl font-bold tracking-tight text-white">Pro vs Smart for GHL widget operations</h2>
            <p class="mt-3 text-sm text-slate-300">
                Choose Pro for lean single-operator funnels or Smart for team-level growth. Both start with demo data in trial and require paid activation for live data.
            </p>
            <div class="mt-8 overflow-hidden rounded-2xl border border-white/10">
                <table class="min-w-full divide-y divide-white/10">
                    <thead class="bg-white/5">
                        <tr class="text-left text-xs uppercase tracking-wide text-slate-300">
                            <th scope="col" class="px-4 py-3">Capability</th>
                            <th scope="col" class="px-4 py-3">Pro</th>
                            <th scope="col" class="px-4 py-3">Smart</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-white/10 bg-slate-900/40 text-sm text-slate-200">
                        @foreach ($comparisonRows as $row)
                            <tr>
                                <th scope="row" class="px-4 py-3 font-semibold text-white">{{ $row['category'] }}</th>
                                <td class="px-4 py-3">{{ $row['pro'] }}</td>
                                <td class="px-4 py-3">{{ $row['smart'] }}</td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
        </div>
    </section>

    <section class="border-b border-slate-800 bg-slate-900/50 px-4 py-14 sm:px-6 lg:px-8" aria-labelledby="workflow-heading">
        <div class="mx-auto max-w-6xl">
            <h2 id="workflow-heading" class="text-3xl font-bold tracking-tight text-white">How the demo-to-live journey works</h2>
            <div class="mt-8 grid gap-5 md:grid-cols-3">
                @foreach ($workflowSteps as $step)
                    <article class="rounded-2xl border border-white/10 bg-slate-950/60 p-6">
                        <h3 class="text-lg font-semibold text-white">{{ $step['title'] }}</h3>
                        <p class="mt-3 text-sm leading-relaxed text-slate-300">{{ $step['description'] }}</p>
                    </article>
                @endforeach
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

    {{-- Revenue Impact: Compliance narrative lowers procurement friction without source disclosures. --}}
    <section class="mx-auto mt-12 max-w-screen-2xl scroll-mt-24 rounded-3xl bg-slate-100 px-4 py-12 sm:px-6 lg:px-8" aria-labelledby="compliance-heading">
        <div class="flex flex-col items-center gap-10 md:flex-row md:items-start md:gap-12">
            <div class="min-w-0 flex-1">
                <h2 id="compliance-heading" class="text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl">Compliance-first delivery for production deployments</h2>
                <p class="mt-4 text-base leading-relaxed text-slate-700 sm:text-lg">
                    GeoIDX enforces policy-ready controls, approved-domain requirements, and audit visibility to keep production deployments aligned with applicable IDX standards.
                </p>
                <p class="mt-6 text-base font-medium text-emerald-700">
                    ✓ Approved-domain controls • ✓ Policy-ready disclaimers and attribution • ✓ Auditable deployment activity
                </p>
            </div>
            <div class="flex w-full max-w-md flex-1 flex-col items-center justify-center rounded-2xl border border-emerald-200 bg-white p-8 text-center text-sm text-slate-600 shadow-sm">
                <span class="inline-flex rounded-full bg-emerald-100 px-4 py-1 text-xs font-semibold tracking-wide text-emerald-800">Production-ready controls</span>
                <p class="mt-4 text-base font-semibold text-slate-900">Purpose-built for compliant rollout</p>
                <p class="mt-2 text-xs text-slate-500">Designed for approved domains, clear disclosures, and accountable operations.</p>
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

    @guest
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
                        <a href="{{ route('login', [], false) }}" class="font-medium text-emerald-300 underline decoration-emerald-400/50 hover:text-emerald-200">
                            Open full login
                        </a>
                        @if (Route::has('register'))
                            <span class="mx-1">·</span>
                            <a href="{{ route('register', [], false) }}" class="font-medium text-emerald-300 underline decoration-emerald-400/50 hover:text-emerald-200">
                                Create account
                            </a>
                        @endif
                    </p>
                </div>
            </div>
        </div>
    @endif
    @endguest

    <footer class="border-t border-white/10 py-8">
        <div class="mx-auto flex max-w-6xl flex-col gap-4 px-4 text-xs text-slate-200 sm:px-6 lg:px-8 md:flex-row md:items-center md:justify-between">
            <p>
                Compliance: This is a marketing page only. No live listing data is displayed here. Live data is available only on approved domains after paid activation and under applicable subscriber terms.
            </p>
            <nav class="flex flex-wrap gap-4" aria-label="Footer">
                @guest
                    <button
                        type="button"
                        wire:click="openLoginModal"
                        class="min-h-11 text-left text-slate-200 underline decoration-white/30 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50"
                    >
                        Subscriber login
                    </button>
                @else
                    <form method="POST" action="{{ route('logout', [], false) }}">
                        @csrf
                        <button
                            type="submit"
                            class="min-h-11 text-left text-slate-200 underline decoration-white/30 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50"
                        >
                            Subscriber logout
                        </button>
                    </form>
                @endguest
                <a href="#" class="min-h-11 inline-flex items-center text-slate-200 underline decoration-white/30 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50" aria-label="Privacy policy (coming soon)">Privacy</a>
                <a href="#" class="min-h-11 inline-flex items-center text-slate-200 underline decoration-white/30 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white/50" aria-label="Terms of service (coming soon)">Terms</a>
            </nav>
        </div>
    </footer>
</div>
