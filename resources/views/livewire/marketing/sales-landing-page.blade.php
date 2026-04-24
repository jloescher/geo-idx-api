<div
    x-data="{ openFaq: 0 }"
    class="bg-slate-950 text-slate-100 transition-colors duration-300"
    @keydown.escape.window="$wire.closeLoginModal()"
>
    <header class="sticky top-0 z-30 border-b border-white/15 bg-slate-950/90 backdrop-blur">
        <div class="mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
            <a href="/" class="text-lg font-semibold tracking-tight">Quantyra GeoIDX</a>
            <div class="flex flex-wrap items-center justify-end gap-2">
                <button
                    type="button"
                    wire:click="openLoginModal"
                    class="rounded-full border border-white/25 px-4 py-2 text-sm font-semibold text-slate-100 hover:border-white/40 hover:bg-white/5"
                >
                    Subscriber login
                </button>
                <a href="#pricing" class="rounded-full bg-emerald-400 px-4 py-2 text-sm font-semibold text-slate-900">See Pricing</a>
            </div>
        </div>
    </header>

    {{-- Revenue Impact: Hero positions clear value and pushes top-funnel CTA clicks. --}}
    <section class="relative overflow-hidden">
        <div class="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(16,185,129,0.25),transparent_45%)]"></div>
        <div class="mx-auto max-w-6xl px-4 pb-16 pt-14 sm:px-6 lg:px-8 lg:pb-20 lg:pt-20">
            <div class="grid items-center gap-10 lg:grid-cols-2">
                <div class="space-y-6">
                    <p class="inline-flex rounded-full border border-cyan-400/40 bg-cyan-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-wider text-cyan-300">
                        IDX + geographic intelligence for listing agents
                    </p>
                    <h1 class="text-4xl font-black leading-tight tracking-tight sm:text-5xl">
                        Win Your Farm Area With IDX Buyers Actually Trust
                    </h1>
                    <p class="text-base leading-7 text-slate-200 sm:text-lg">
                        Quantyra GeoIDX helps you turn local search intent into signed representation—without renting leads from portals. Your IDX experience is built around <strong class="font-semibold text-slate-100">real geography</strong>: county lines, city limits, and neighborhood context layered on fast maps, so buyers understand <em>where</em> they are searching before they ever ask you a question. Live MLS data is delivered through a centralized, compliance-aware IDX API; visitors see a short preview, then complete email + phone OTP verification. Verified inquiries route to <strong class="font-semibold text-slate-100">you</strong>—the subscribed agent or lender covering that market.
                    </p>
                    <div class="flex flex-wrap gap-3">
                        <a href="#demo" class="rounded-full bg-emerald-400 px-5 py-3 text-sm font-semibold text-slate-900">Book Demo</a>
                        <button
                            type="button"
                            wire:click="openLoginModal"
                            class="rounded-full border border-white/30 px-5 py-3 text-sm font-semibold text-slate-100 hover:border-white/45 hover:bg-white/5"
                        >
                            Subscriber login
                        </button>
                        <a href="#pricing" class="rounded-full border border-cyan-300/50 px-5 py-3 text-sm font-semibold text-cyan-200">See Pricing</a>
                    </div>
                    <p class="text-sm text-slate-200">
                        Manage embeds, lead routing, billing, and your subscriber dashboard after signing in.
                    </p>
                    <div class="flex flex-wrap gap-2 text-xs font-medium text-slate-200">
                        <span class="rounded-full border border-white/20 px-3 py-1">Stellar MLS–compliant IDX</span>
                        <span class="rounded-full border border-white/20 px-3 py-1">County &amp; city boundary context</span>
                        <span class="rounded-full border border-white/20 px-3 py-1">Tampa Bay &amp; Gulf Coast focus</span>
                    </div>
                </div>
                <div class="rounded-3xl border border-white/10 bg-gradient-to-br from-slate-800 to-slate-900 p-5 shadow-2xl">
                    <div class="h-72 rounded-2xl bg-[url('https://images.unsplash.com/photo-1564013799919-ab600027ffc6?auto=format&fit=crop&w=1200&q=80')] bg-cover bg-center"></div>
                    <p class="mt-4 text-sm text-slate-200">
                        Mobile-first search and map layouts built for agents who prospect by geography—not generic national templates.
                    </p>
                </div>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Problem framing increases urgency and conversion intent. --}}
    <section class="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
        <h2 class="text-3xl font-bold tracking-tight">Why Agents Still Lose Deals They Should Win</h2>
        <div class="mt-8 grid gap-4 md:grid-cols-3">
            <article class="rounded-2xl border border-white/10 p-6">
                <h3 class="text-lg font-semibold">Portal “Leads” Are Not Your Pipeline</h3>
                <p class="mt-2 text-sm text-slate-200">You compete on speed against dozens of agents for the same name—often before you know whether they are serious about your price band or geography.</p>
            </article>
            <article class="rounded-2xl border border-white/10 p-6">
                <h3 class="text-lg font-semibold">IDX That Ignores Geography Confuses Buyers</h3>
                <p class="mt-2 text-sm text-slate-200">When search and maps do not reflect county lines, city limits, and the micro-markets you farm, buyers bounce—or worse, they call you for areas you do not serve.</p>
            </article>
            <article class="rounded-2xl border border-white/10 p-6">
                <h3 class="text-lg font-semibold">Unverified Inquiries Waste Follow-Up</h3>
                <p class="mt-2 text-sm text-slate-200">Traditional forms fill your CRM with junk. You need verified contact data tied to real listing interest before you invest showing time.</p>
            </article>
        </div>
    </section>

    {{-- Revenue Impact: Solution section ties product architecture to margin expansion. --}}
    <section class="border-y border-white/10 py-14">
        <div class="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <h2 class="text-3xl font-bold tracking-tight">Built For Agents Who Sell Place, Not Just Price</h2>
            <p class="mt-4 max-w-3xl text-slate-200">
                Quantyra GeoIDX pairs a compliance-first IDX feed with <strong class="font-semibold text-slate-100">geographic storytelling</strong>: county and municipal boundaries, coastal vs. inland context, and the hyper-local vocabulary buyers already use when they talk about “school side,” “beach side,” or “inside the beltway.” AI-assisted pages and structured data help you rank for the neighborhoods you actually farm—while keeping MLS display rules intact on authorized IDX properties.
            </p>
            <div class="mt-6 rounded-2xl border border-emerald-300/30 bg-emerald-500/10 p-5 text-sm text-emerald-100">
                <strong class="font-semibold text-emerald-50">Geographic intelligence layer:</strong> visualize and filter inventory with county and city boundary context so prospects understand jurisdiction, commute envelopes, and municipal services before they tour.
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Process clarity reduces friction and improves CTA confidence. --}}
    <section class="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
        <h2 class="text-3xl font-bold tracking-tight">How It Works</h2>
        <div class="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <article class="rounded-2xl border border-white/10 p-5">
                <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Step 1</p>
                <h3 class="mt-2 font-semibold">Local SEO &amp; AI Content</h3>
                <p class="mt-2 text-sm text-slate-200">Publish neighborhood- and corridor-specific narratives that mirror how buyers describe where they want to live.</p>
            </article>
            <article class="rounded-2xl border border-white/10 p-5">
                <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Step 2</p>
                <h3 class="mt-2 font-semibold">Boundary-Aware IDX &amp; Maps</h3>
                <p class="mt-2 text-sm text-slate-200">Layer county polygons, city limits, and custom farm boundaries atop MLS-backed search on your authorized IDX sites.</p>
            </article>
            <article class="rounded-2xl border border-white/10 p-5">
                <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Step 3</p>
                <h3 class="mt-2 font-semibold">OTP Lead Gating</h3>
                <p class="mt-2 text-sm text-slate-200">Let shoppers preview a handful of listings, then require verified email + phone before unlocking deeper detail—so your time goes to serious buyers.</p>
            </article>
            <article class="rounded-2xl border border-white/10 p-5">
                <p class="text-xs font-semibold uppercase tracking-wide text-cyan-300">Step 4</p>
                <h3 class="mt-2 font-semibold">Exclusive Agent Routing</h3>
                <p class="mt-2 text-sm text-slate-200">Qualified conversations land in your dashboard, embeds, or connected CRM—owned by the subscribed agent covering that geography.</p>
            </article>
        </div>
    </section>

    {{-- Revenue Impact: Feature proof points support premium pricing and conversion. --}}
    <section class="border-y border-white/10 py-14">
        <div class="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <h2 class="text-3xl font-bold tracking-tight">Everything You Need To Monetize Local Expertise</h2>
            <div class="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                <article class="rounded-2xl border border-white/10 p-5">
                    <h3 class="font-semibold">AI Blog + Programmatic SEO</h3>
                    <p class="mt-2 text-sm text-slate-200">Ship authoritative neighborhood guides, corridor comparisons, and relocation content that reinforces your geographic authority.</p>
                </article>
                <article class="rounded-2xl border border-white/10 p-5">
                    <h3 class="font-semibold">County &amp; City Boundary Maps</h3>
                    <p class="mt-2 text-sm text-slate-200">Highlight municipal services, school attribution, and commute envelopes with Leaflet/OpenStreetMap-ready layers on authorized IDX experiences.</p>
                </article>
                <article class="rounded-2xl border border-white/10 p-5">
                    <h3 class="font-semibold">OTP Lead Forms</h3>
                    <p class="mt-2 text-sm text-slate-200">Stop chasing ghosts—collect verified phone + email from buyers who already engaged with your listings.</p>
                </article>
                <article class="rounded-2xl border border-white/10 p-5">
                    <h3 class="font-semibold">Agent Command Center</h3>
                    <p class="mt-2 text-sm text-slate-200">Track embed performance, manage subscription billing, and coordinate LeadConnector workflows from one dashboard.</p>
                </article>
                <article class="rounded-2xl border border-white/10 p-5">
                    <h3 class="font-semibold">15-Minute MLS Cache</h3>
                    <p class="mt-2 text-sm text-slate-200">Keep IDX pages snappy while honoring refresh rules through the centralized API.</p>
                </article>
                <article class="rounded-2xl border border-white/10 p-5">
                    <h3 class="font-semibold">Farm-Area Packaging</h3>
                    <p class="mt-2 text-sm text-slate-200">Bundle the cities, ZIP clusters, and county segments you already prospect—without forcing a one-size national IDX template on your brand.</p>
                </article>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Pilot-market previews increase trust without compliance risk. --}}
    <section class="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
        <h2 class="text-3xl font-bold tracking-tight">How Geographic Packaging Looks For Agents</h2>
        <p class="mt-3 text-sm text-slate-200">
            Illustrative positioning only—no live MLS listings, boundary files, or map tiles are loaded on this marketing page.
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

    {{-- Revenue Impact: Social proof reduces perceived risk and increases sign-up intent. --}}
    <section class="border-y border-white/10 py-14">
        <div class="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <h2 class="text-3xl font-bold tracking-tight">Social Proof</h2>
            <div class="mt-8 grid gap-4 md:grid-cols-3">
                <blockquote class="rounded-2xl border border-white/10 p-5 text-sm text-slate-200">
                    “Placeholder: Buyers finally stopped asking ‘Is that still Tampa?’ once county lines were obvious on the map.”
                </blockquote>
                <blockquote class="rounded-2xl border border-white/10 p-5 text-sm text-slate-200">
                    “Placeholder: My listing appointments doubled after OTP verification—we talk to people who already proved they are serious.”
                </blockquote>
                <blockquote class="rounded-2xl border border-white/10 p-5 text-sm text-slate-200">
                    “Placeholder: I reclaimed weekends because the IDX experience sells the geography before I repeat the same script.”
                </blockquote>
            </div>
        </div>
    </section>

    {{-- Revenue Impact: Pricing teaser filters intent and drives high-value demo calls. --}}
    <section id="pricing" class="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
        <h2 class="text-3xl font-bold tracking-tight">Invest In The Geography You Already Own</h2>
        <p class="mt-3 max-w-3xl text-sm text-slate-200">
            Final packaging is tailored to your county, city, and farm strategy—book a demo for agent, team, or brokerage pricing.
        </p>
        <div class="mt-8 grid gap-4 md:grid-cols-3">
            <article class="rounded-2xl border border-white/10 p-6">
                <h3 class="text-lg font-semibold">Solo Listing Agent</h3>
                <p class="mt-2 text-sm text-slate-200">Launch IDX, boundary-aware maps, and OTP capture for the corridor you already dominate.</p>
            </article>
            <article class="rounded-2xl border border-emerald-300/40 bg-emerald-500/10 p-6">
                <h3 class="text-lg font-semibold">Top Producer / Small Team</h3>
                <p class="mt-2 text-sm text-slate-100">Coordinate multiple farm pockets, embeds, and lead routing without stitching five vendor tools together.</p>
            </article>
            <article class="rounded-2xl border border-white/10 p-6">
                <h3 class="text-lg font-semibold">Brokerage &amp; ISA Programs</h3>
                <p class="mt-2 text-sm text-slate-200">Enterprise controls, compliance guardrails, and rollout support for multi-office geographic coverage.</p>
            </article>
        </div>
    </section>

    {{-- Revenue Impact: Final CTA captures bottom-funnel intent and books demos. --}}
    <section id="demo" class="border-y border-white/10 py-14">
        <div class="mx-auto flex max-w-4xl flex-col items-center px-4 text-center sm:px-6 lg:px-8">
            <h2 class="text-3xl font-bold tracking-tight">Ready To Turn Geographic Authority Into Signed Buyers?</h2>
            <p class="mt-4 text-slate-200">Book a walkthrough, see boundary-aware IDX in action on authorized domains, and map the farm areas you want to monetize next.</p>
            <div class="mt-6 flex flex-wrap justify-center gap-3">
                <button
                    id="get-started"
                    type="button"
                    wire:click="openLoginModal"
                    class="rounded-full bg-emerald-400 px-5 py-3 text-sm font-semibold text-slate-900 hover:bg-emerald-300"
                >
                    Subscriber login
                </button>
                <a href="#demo" class="rounded-full border border-white/30 px-5 py-3 text-sm font-semibold">Book Demo</a>
                <a href="#pricing" class="rounded-full border border-cyan-300/40 px-5 py-3 text-sm font-semibold text-cyan-200">See Pricing</a>
            </div>
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
                    </p>
                </div>
            </div>
        </div>
    @endif

    <footer class="border-t border-white/10 py-8">
        <div class="mx-auto flex max-w-6xl flex-col gap-4 px-4 text-xs text-slate-300 sm:px-6 lg:px-8 md:flex-row md:items-center md:justify-between">
            <p>
                Compliance: This is a marketing page only. No live Stellar MLS listing data is displayed here. IDX data appears exclusively on authorized domains per signed Stellar MLS IDX agreement.
            </p>
            <div class="flex flex-wrap gap-4">
                <button type="button" wire:click="openLoginModal" class="text-left hover:text-white">Subscriber login</button>
                <a href="#" class="hover:text-white">Privacy</a>
                <a href="#" class="hover:text-white">Terms</a>
            </div>
        </div>
    </footer>
</div>
