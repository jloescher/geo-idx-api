@if ($activePanel === 'leads')
    @if ($leadsEligible)
        <section class="idx-leads-canvas mt-6">
            <div class="idx-leads-metric-grid">
            <article class="idx-leads-metric">
                <p class="text-xs uppercase tracking-wide text-slate-400">Total Leads</p>
                <svg class="idx-leads-metric-icon" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M16 11c1.66 0 2.99-1.57 2.99-3.5S17.66 4 16 4s-3 1.57-3 3.5S14.34 11 16 11Zm-8 0c1.66 0 2.99-1.57 2.99-3.5S9.66 4 8 4 5 5.57 5 7.5 6.34 11 8 11Zm0 2c-2.33 0-7 1.17-7 3.5V20h14v-3.5C15 14.17 10.33 13 8 13Zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.98 1.97 3.45V20h6v-3.5c0-2.33-4.67-3.5-7-3.5Z"/></svg>
                <p class="idx-leads-kpi">{{ number_format($totalLeads) }}</p>
            </article>
            <article class="idx-leads-metric">
                <p class="text-xs uppercase tracking-wide text-slate-400">Leads This Month</p>
                <svg class="idx-leads-metric-icon" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="m14 6 7 7-1.41 1.41L15 9.83V20h-2V9.83l-4.59 4.58L7 13l7-7Z"/></svg>
                <p class="idx-leads-kpi">{{ number_format((int) ($leadsThisMonth ?? 0)) }} <span class="idx-leads-trend-up text-3xl">↑</span></p>
            </article>
            <article class="idx-leads-metric">
                <p class="text-xs uppercase tracking-wide text-slate-400">Avg. Conversion (Visitor to Lead)</p>
                <svg class="idx-leads-metric-icon" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M16 6h5v5h-2V9.41l-5.29 5.3-4-4L2 18.41 3.41 20 9.71 13.7l4 4L20 11.41V14h2V6h-6Z"/></svg>
                <p class="idx-leads-kpi">{{ number_format($conversionRate, 1) }}% <span class="idx-leads-trend-up text-3xl">↑</span></p>
            </article>
            <article class="idx-leads-metric">
                <p class="text-xs uppercase tracking-wide text-slate-400">Hot Leads (24h)</p>
                <svg class="idx-leads-metric-icon" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M13.5.67s.74 2.65-.8 5.64c-1.4 2.7-4.24 3.94-5.44 6.74-1.3 3.04.26 6.95 3.5 8.47 3.2 1.5 7.14.12 8.8-3.06 2.56-4.9-1.2-10.88-6.06-17.79Z"/></svg>
                <p class="idx-leads-kpi">{{ number_format($hotLeads24h) }} <span class="idx-leads-trend-down text-3xl">↑</span></p>
                <p class="mt-1 text-xs text-slate-400">Verified leads received in the last 24 hours.</p>
            </article>
            </div>
        </section>

        <section class="idx-leads-canvas mt-8">
            <h2 class="text-3xl font-semibold text-white">LEADS</h2>
            <livewire:dashboard.leads.leads-table />
        </section>

        <div class="mt-4 grid gap-4 xl:grid-cols-2">
            <livewire:dashboard.leads.lead-saved-searches-panel />
            <livewire:dashboard.leads.lead-alert-settings-panel />
        </div>
    @else
        <section class="mt-8 rounded-2xl border border-amber-400/30 bg-amber-900/20 p-5 text-amber-100">
            <h2 class="text-lg font-semibold">Leads access is locked</h2>
            <p class="mt-2 text-sm">Requirements: active paid subscription, eligible plan, and verified MLS membership.</p>
            <a wire:navigate href="{{ route('dashboard.index', ['panel' => 'onboarding'], false) }}" class="mt-3 inline-flex text-sm font-semibold text-amber-200 underline">Review setup checklist</a>
        </section>
    @endif
@endif
