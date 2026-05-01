<x-filament-panels::page>
    @php
        /** @var array{widgets: bool, seo_landing_pages: bool} $mff */
        $mff = $marketingFeatureFlags ?? ['widgets' => false, 'seo_landing_pages' => false];
        /** @var list<string> $compsDatasetCodes */
        $compsDatasetCodes = $compsDatasetCodes ?? [(string) config('bridge.dataset', 'stellar')];
    @endphp
    <div class="idx-agent-shell space-y-6">
        <div class="idx-panel">
            <h3 class="idx-panel-title">Marketing</h3>
            <p class="idx-panel-subtitle">
                Generate and manage shareable links tied to saved searches.
            </p>

            <div class="mt-4 grid gap-4 lg:grid-cols-4">
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Saved search
                    <select id="idxShareSearchSelect" class="idx-input">
                        <option value="">Optional: link to a saved search</option>
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Link type
                    <select id="idxShareTemplateKind" class="idx-input">
                        <option value="standard">Standard share link</option>
                        @if ($mff['seo_landing_pages'])
                            <option value="seo_landing">SEO landing template</option>
                        @endif
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    UTM source
                    <input id="idxShareUtmSource" type="text" class="idx-input" placeholder="newsletter" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    UTM campaign
                    <input id="idxShareUtmCampaign" type="text" class="idx-input" placeholder="spring-buyers" />
                </label>
            </div>

            <div class="mt-3 flex flex-wrap items-center gap-2">
                <button id="idxShareCreateBtn" type="button" class="idx-btn-primary">
                    Generate share link
                </button>
                <button id="idxShareRefreshBtn" type="button" class="idx-btn-secondary">
                    Refresh
                </button>
                <span id="idxShareStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
            </div>
        </div>

        <div class="idx-panel">
            <h4 class="mb-2 text-sm font-semibold text-slate-900 dark:text-slate-100">Home value quick estimate</h4>
            <p class="mb-4 text-xs text-slate-500 dark:text-slate-400">
                Run an MLS comps-backed point estimate using the same Bridge pipeline as Quantyra home-value mode.
                Useful for sellers and test-the-market conversations; not an appraisal or broker price opinion.
            </p>
            <div class="mb-4 grid gap-4 lg:grid-cols-4">
                <label class="block text-xs text-slate-600 dark:text-slate-300 lg:col-span-2">
                    Street address
                    <input id="idxHomeValueAddress" type="text" class="idx-input" placeholder="123 Main St, Tampa, FL" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    MLS dataset
                    <select id="idxCompsDataset" class="idx-input">
                        @foreach ($compsDatasetCodes as $datasetCode)
                            <option value="{{ $datasetCode }}">{{ $datasetCode }}</option>
                        @endforeach
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    &nbsp;
                    <button id="idxHomeValueRunBtn" type="button" class="idx-btn-primary w-full">Run estimate</button>
                </label>
            </div>
            <div class="mb-4 grid gap-4 md:grid-cols-4">
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Bedrooms
                    <input id="idxHomeValueBedrooms" type="number" min="0" max="20" class="idx-input" placeholder="Optional" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Full baths
                    <input id="idxHomeValueFullBaths" type="number" min="0" max="20" class="idx-input" placeholder="Optional" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Half baths
                    <input id="idxHomeValueHalfBaths" type="number" min="0" max="10" value="0" class="idx-input" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Living area (sq ft)
                    <input id="idxHomeValueSqft" type="number" min="0" class="idx-input" placeholder="Optional" />
                </label>
            </div>
            <p id="idxHomeValueStatus" class="text-xs text-slate-500 dark:text-slate-400"></p>
            <pre id="idxHomeValueResult" class="mt-3 hidden overflow-x-auto rounded-md bg-slate-100 p-3 text-xs text-slate-800 dark:bg-slate-900 dark:text-slate-200"></pre>
        </div>

        <div class="idx-panel">
            <h4 class="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">Share links</h4>
            <div class="mb-3 flex flex-wrap items-center gap-2">
                <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">Metrics window</span>
                <button type="button" data-metrics-days="7" class="idx-metrics-window-btn idx-btn-secondary">
                    7d
                </button>
                <button type="button" data-metrics-days="30" class="idx-metrics-window-btn idx-btn-secondary">
                    30d
                </button>
            </div>
            <div class="mb-4 grid gap-3 md:grid-cols-3">
                @if ($mff['seo_landing_pages'])
                    <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">SEO templates</p>
                        <p id="idxMetricSeoTotal" class="text-lg font-semibold text-slate-900 dark:text-slate-100">0</p>
                    </div>
                    <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">SEO active</p>
                        <p id="idxMetricSeoActive" class="text-lg font-semibold text-emerald-700 dark:text-emerald-300">0</p>
                    </div>
                    <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">SEO inactive</p>
                        <p id="idxMetricSeoInactive" class="text-lg font-semibold text-rose-700 dark:text-rose-300">0</p>
                    </div>
                @endif
                <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                    <p class="text-[11px] uppercase tracking-wide text-slate-500">Created (window)</p>
                    <p id="idxMetricCreatedInWindow" class="text-lg font-semibold text-slate-900 dark:text-slate-100">0</p>
                    <p id="idxMetricCreatedDelta" class="text-[11px] text-slate-500">Delta: 0</p>
                </div>
                <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                    <p class="text-[11px] uppercase tracking-wide text-slate-500">Visits (window)</p>
                    <p id="idxMetricVisitsInWindow" class="text-lg font-semibold text-cyan-700 dark:text-cyan-300">0</p>
                    <p id="idxMetricVisitsDelta" class="text-[11px] text-slate-500">Delta: 0</p>
                </div>
            </div>
            <div class="mb-4 rounded-lg border border-slate-200 px-3 py-3 dark:border-slate-700">
                <div class="mb-2 flex items-center justify-between gap-2">
                    <p class="text-[11px] uppercase tracking-wide text-slate-500">Trend (created / visits)</p>
                    <button id="idxExportTrendCsvBtn" type="button" class="rounded-md border border-emerald-300 px-2 py-1 text-[11px] font-semibold text-emerald-700 dark:border-emerald-800 dark:text-emerald-300">
                        Export trend CSV
                    </button>
                </div>
                <div id="idxMetricTrendRows" class="space-y-1 text-xs text-slate-600 dark:text-slate-300">
                    <p>No trend data yet.</p>
                </div>
            </div>
            <div class="mb-4 rounded-lg border border-slate-200 px-3 py-3 text-xs text-slate-600 dark:border-slate-700 dark:text-slate-300">
                <p class="mb-2 text-[11px] uppercase tracking-wide text-slate-500">Retention operations</p>
                <div class="mb-3 flex flex-wrap items-center gap-2">
                    <label class="text-[11px]">
                        Estimate days
                        <input id="idxOpsEstimateDays" type="number" min="1" max="3650" value="90" class="ml-1 w-20 rounded border border-slate-300 px-1 py-0.5 dark:border-slate-700 dark:bg-slate-900" />
                    </label>
                    <button id="idxOpsEstimateBtn" type="button" class="rounded-md border border-cyan-300 px-2 py-1 text-[11px] font-semibold text-cyan-700 dark:border-cyan-800 dark:text-cyan-300">
                        Estimate prune
                    </button>
                    <span id="idxOpsEstimateStatus" class="text-[11px] text-slate-500"></span>
                </div>
                <div class="grid gap-2 md:grid-cols-2">
                    <p>Prune days: <span id="idxOpsPruneDays" class="font-semibold text-slate-900 dark:text-slate-100">-</span></p>
                    <p>Candidate rows: <span id="idxOpsPruneCandidates" class="font-semibold text-slate-900 dark:text-slate-100">-</span></p>
                    <p>Schedule: <span id="idxOpsSchedule" class="font-semibold text-slate-900 dark:text-slate-100">-</span></p>
                    <p>Command: <code id="idxOpsCommand" class="rounded bg-slate-100 px-1 py-0.5 text-[11px] dark:bg-slate-800">-</code></p>
                </div>
            </div>
            <div class="mb-3 grid gap-3 md:grid-cols-3">
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Filter type
                    <select id="idxShareFilterKind" class="idx-input">
                        <option value="">All</option>
                        <option value="standard">Standard</option>
                        @if ($mff['seo_landing_pages'])
                            <option value="seo_landing">SEO landing</option>
                        @endif
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Filter status
                    <select id="idxShareFilterStatus" class="idx-input">
                        <option value="">All</option>
                        <option value="active">Active</option>
                        <option value="inactive">Inactive</option>
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Search token
                    <input id="idxShareFilterQuery" type="text" class="idx-input" placeholder="token fragment" />
                </label>
            </div>
            <div class="mb-3 flex flex-wrap items-center gap-2">
                @if ($mff['seo_landing_pages'])
                    <button id="idxSeoTemplatesPresetBtn" type="button" class="rounded-md border border-cyan-300 px-2 py-1 text-xs font-semibold text-cyan-700 dark:border-cyan-800 dark:text-cyan-300">
                        SEO templates only
                    </button>
                @endif
                <button id="idxExportShareCsvBtn" type="button" class="rounded-md border border-emerald-300 px-2 py-1 text-xs font-semibold text-emerald-700 dark:border-emerald-800 dark:text-emerald-300">
                    Export CSV
                </button>
                <button id="idxClearShareFiltersBtn" type="button" class="idx-btn-secondary">
                    Clear filters
                </button>
            </div>
            <div class="overflow-x-auto">
                <table class="idx-data-table w-full text-left text-sm">
                    <thead class="border-b border-slate-200 text-xs uppercase text-slate-500 dark:border-slate-700 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-4 font-medium">Search</th>
                            <th class="py-2 pr-4 font-medium">Type</th>
                            <th class="py-2 pr-4 font-medium">URL</th>
                            <th class="py-2 pr-4 font-medium">Status</th>
                            <th class="py-2 pr-4 font-medium">Token</th>
                            <th class="py-2 pr-4 font-medium">Created</th>
                            <th class="py-2 font-medium">Actions</th>
                        </tr>
                    </thead>
                    <tbody id="idxShareRows">
                        <tr><td colspan="7" class="py-4 text-slate-500">Loading share links...</td></tr>
                    </tbody>
                </table>
            </div>
            @if ($mff['seo_landing_pages'])
                <div class="mt-4 rounded-lg border border-slate-200 px-3 py-3 dark:border-slate-700">
                    <div class="mb-2 flex items-center justify-between">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">SEO landing pages</p>
                        <span id="idxSeoLandingCount" class="text-xs text-slate-500">0 pages</span>
                    </div>
                    <div id="idxSeoLandingRows" class="space-y-1 text-xs text-slate-600 dark:text-slate-300">
                        <p>No SEO landing pages yet.</p>
                    </div>
                </div>
            @else
                <div class="mt-4 rounded-lg border border-dashed border-slate-300 bg-slate-50/60 px-3 py-3 text-sm text-slate-600 dark:border-slate-600 dark:bg-slate-950/30 dark:text-slate-400">
                    <p class="font-medium text-slate-800 dark:text-slate-200">SEO landing pages</p>
                    <p class="mt-1 text-xs leading-relaxed">
                        This workspace is hidden until you enable <span class="font-mono text-[11px]">seo_landing_pages</span> under Agent Settings → feature flags.
                    </p>
                </div>
            @endif
        </div>

        @if ($mff['widgets'])
            <div class="idx-panel">
                <h4 class="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">Widget embed code generator</h4>
                <p class="mb-3 text-xs text-slate-500 dark:text-slate-400">Generate embeddable widget snippets for your website. Copy the code and paste it into your site's HTML.</p>
                <div class="mb-4 grid gap-3 md:grid-cols-4">
                    <label class="block text-xs text-slate-600 dark:text-slate-300">
                        Widget type
                        <select id="idxWidgetType" class="idx-input">
                            <option value="search">Property search</option>
                            <option value="lead_form">Lead capture form</option>
                            <option value="showcase">Listing showcase</option>
                        </select>
                    </label>
                    <label class="block text-xs text-slate-600 dark:text-slate-300">
                        Linked search
                        <select id="idxWidgetSearchSelect" class="idx-input">
                            <option value="">None (default)</option>
                        </select>
                    </label>
                    <label class="block text-xs text-slate-600 dark:text-slate-300">
                        Max listings
                        <input id="idxWidgetMaxListings" type="number" min="1" max="50" value="6" class="idx-input" />
                    </label>
                    <label class="block text-xs text-slate-600 dark:text-slate-300">
                        Theme
                        <select id="idxWidgetTheme" class="idx-input">
                            <option value="light">Light</option>
                            <option value="dark">Dark</option>
                            <option value="auto">Auto (system)</option>
                        </select>
                    </label>
                </div>
                <div class="mb-3 flex items-center gap-2">
                    <button id="idxWidgetGenerateBtn" type="button" class="idx-btn-primary">Generate embed code</button>
                    <button id="idxWidgetCopyBtn" type="button" class="idx-btn-secondary hidden">Copy embed code</button>
                    <span id="idxWidgetStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
                </div>
                <div id="idxWidgetPreviewContainer" class="hidden">
                    <p class="mb-1 text-[11px] uppercase tracking-wide text-slate-500">Preview</p>
                    <pre id="idxWidgetPreviewCode" class="overflow-x-auto rounded-md bg-slate-100 p-3 text-xs text-slate-800 dark:bg-slate-900 dark:text-slate-200"></pre>
                </div>
            </div>
        @else
            <div class="idx-panel">
                <h4 class="mb-2 text-sm font-semibold text-slate-900 dark:text-slate-100">Widget embeds</h4>
                <p class="text-sm text-slate-600 dark:text-slate-400">
                    Widget embed code generation is turned off. Enable <span class="font-mono text-xs">widgets</span> under Agent Settings → feature flags to use the embed builder and server-side embed validation.
                </p>
            </div>
        @endif
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const featureFlags = @json($mff);
            const searchSelect = document.getElementById('idxShareSearchSelect');
            const utmSource = document.getElementById('idxShareUtmSource');
            const utmCampaign = document.getElementById('idxShareUtmCampaign');
            const templateKindEl = document.getElementById('idxShareTemplateKind');
            const createBtn = document.getElementById('idxShareCreateBtn');
            const refreshBtn = document.getElementById('idxShareRefreshBtn');
            const statusEl = document.getElementById('idxShareStatus');
            const rowsEl = document.getElementById('idxShareRows');
            const filterKindEl = document.getElementById('idxShareFilterKind');
            const filterStatusEl = document.getElementById('idxShareFilterStatus');
            const filterQueryEl = document.getElementById('idxShareFilterQuery');
            const metricSeoTotalEl = document.getElementById('idxMetricSeoTotal');
            const metricSeoActiveEl = document.getElementById('idxMetricSeoActive');
            const metricSeoInactiveEl = document.getElementById('idxMetricSeoInactive');
            const metricCreatedInWindowEl = document.getElementById('idxMetricCreatedInWindow');
            const metricVisitsInWindowEl = document.getElementById('idxMetricVisitsInWindow');
            const metricCreatedDeltaEl = document.getElementById('idxMetricCreatedDelta');
            const metricVisitsDeltaEl = document.getElementById('idxMetricVisitsDelta');
            const metricTrendRowsEl = document.getElementById('idxMetricTrendRows');
            const seoLandingRowsEl = document.getElementById('idxSeoLandingRows');
            const seoLandingCountEl = document.getElementById('idxSeoLandingCount');
            const opsPruneDaysEl = document.getElementById('idxOpsPruneDays');
            const opsPruneCandidatesEl = document.getElementById('idxOpsPruneCandidates');
            const opsScheduleEl = document.getElementById('idxOpsSchedule');
            const opsCommandEl = document.getElementById('idxOpsCommand');
            const opsEstimateDaysEl = document.getElementById('idxOpsEstimateDays');
            const opsEstimateBtn = document.getElementById('idxOpsEstimateBtn');
            const opsEstimateStatusEl = document.getElementById('idxOpsEstimateStatus');
            const exportTrendCsvBtn = document.getElementById('idxExportTrendCsvBtn');
            const metricsWindowButtons = Array.from(document.querySelectorAll('.idx-metrics-window-btn'));
            const seoTemplatesPresetBtn = document.getElementById('idxSeoTemplatesPresetBtn');
            const exportShareCsvBtn = document.getElementById('idxExportShareCsvBtn');
            const clearShareFiltersBtn = document.getElementById('idxClearShareFiltersBtn');
            const compsAddressEl = document.getElementById('idxHomeValueAddress');
            const compsDatasetEl = document.getElementById('idxCompsDataset');
            const compsBedroomsEl = document.getElementById('idxHomeValueBedrooms');
            const compsFullBathsEl = document.getElementById('idxHomeValueFullBaths');
            const compsHalfBathsEl = document.getElementById('idxHomeValueHalfBaths');
            const compsSqftEl = document.getElementById('idxHomeValueSqft');
            const compsRunBtnEl = document.getElementById('idxHomeValueRunBtn');
            const compsStatusEl = document.getElementById('idxHomeValueStatus');
            const compsResultEl = document.getElementById('idxHomeValueResult');
            let shareLinks = [];
            let metricsWindowDays = 7;

            compsRunBtnEl?.addEventListener('click', async () => {
                const address = String(compsAddressEl?.value || '').trim();
                if (address === '') {
                    if (compsStatusEl) {
                        compsStatusEl.textContent = 'Enter an address.';
                    }

                    return;
                }
                if (compsStatusEl) {
                    compsStatusEl.textContent = 'Running estimate…';
                }
                if (compsResultEl) {
                    compsResultEl.classList.add('hidden');
                    compsResultEl.textContent = '';
                }
                const subject = {
                    address,
                    bedrooms: compsBedroomsEl?.value !== '' ? Number(compsBedroomsEl.value) : null,
                    full_bathrooms: compsFullBathsEl?.value !== '' ? Number(compsFullBathsEl.value) : null,
                    half_bathrooms: compsHalfBathsEl?.value !== '' ? Number(compsHalfBathsEl.value) : 0,
                    living_area_sqft: compsSqftEl?.value !== '' ? Number(compsSqftEl.value) : null,
                    pool: null,
                    waterfront: null,
                    lot_size_sqft: null,
                    hoa_monthly_fee: null,
                };
                const body = {
                    subject,
                    mode: 'home_value',
                    scope: { type: 'radius', radius_miles: 3 },
                    filters: {
                        sold_months_back: 12,
                        max_sold_comps: 10,
                        living_area_pct: 20,
                        beds_tolerance: 1,
                        baths_tolerance: 1,
                        year_built_tolerance: 15,
                    },
                    home_value_params: { sold_months_back: 12, max_comps: 8 },
                };
                const dataset = compsDatasetEl?.value ? String(compsDatasetEl.value) : 'stellar';
                try {
                    const response = await fetch(`/agent/comps/run?dataset=${encodeURIComponent(dataset)}`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            Accept: 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                        },
                        body: JSON.stringify(body),
                    });
                    const payload = await response.json();
                    if (!response.ok || payload?.success === false) {
                        if (compsStatusEl) {
                            compsStatusEl.textContent =
                                typeof payload?.message === 'string'
                                    ? payload.message
                                    : String(payload?.error ?? 'Estimate failed.');
                        }
                        if (compsResultEl && payload) {
                            compsResultEl.classList.remove('hidden');
                            compsResultEl.textContent = JSON.stringify(payload, null, 2);
                        }

                        return;
                    }
                    const hv = payload?.home_value_result ?? {};
                    const point = hv?.point_estimate;
                    const range = hv?.range ?? {};
                    if (compsStatusEl) {
                        const lowLabel = range?.low != null ? Number(range.low).toLocaleString() : '–';
                        const highLabel = range?.high != null ? Number(range.high).toLocaleString() : '–';
                        compsStatusEl.textContent =
                            point != null
                                ? `Estimate: ${Number(point).toLocaleString()} (${lowLabel}–${highLabel})`
                                : 'Estimate completed.';
                    }
                    if (compsResultEl) {
                        compsResultEl.classList.remove('hidden');
                        compsResultEl.textContent = JSON.stringify(payload, null, 2);
                    }
                } catch {
                    if (compsStatusEl) {
                        compsStatusEl.textContent = 'Request failed.';
                    }
                }
            });

            const renderRows = () => {
                if (shareLinks.length === 0) {
                    rowsEl.innerHTML = '<tr><td colspan="7" class="py-4 text-slate-500">No share links yet.</td></tr>';
                    return;
                }
                rowsEl.innerHTML = '';
                shareLinks.forEach((link) => {
                    const created = link.created_at ? new Date(link.created_at).toLocaleString() : 'n/a';
                    const templateKind = link.attribution_json?.template_kind || 'standard';
                    const row = document.createElement('tr');
                    row.className = 'border-b border-slate-100 dark:border-slate-800';
                    row.innerHTML = `
                        <td class="py-2 pr-4">${link.agent_search_name || 'General'}</td>
                        <td class="py-2 pr-4">${templateKind}</td>
                        <td class="py-2 pr-4">
                            <a class="text-cyan-700 hover:underline dark:text-cyan-300" href="${link.canonical_url || link.url}" target="_blank" rel="noopener">${link.canonical_url || link.url}</a>
                            ${link.reused_existing ? '<div class="mt-1 text-[11px] text-emerald-600 dark:text-emerald-300">Reused canonical template link</div>' : ''}
                        </td>
                        <td class="py-2 pr-4">${link.status || 'active'}</td>
                        <td class="py-2 pr-4 font-mono text-xs">${link.token}</td>
                        <td class="py-2 pr-4">${created}</td>
                        <td class="py-2">
                            <div class="flex flex-wrap gap-2">
                                <button type="button" data-action="open-canonical" data-id="${link.id}" class="rounded-md border border-cyan-300 px-2 py-1 text-xs font-medium text-cyan-700 dark:border-cyan-800 dark:text-cyan-300">Open canonical</button>
                                <button type="button" data-action="copy-canonical-path" data-id="${link.id}" class="idx-btn-secondary">Copy path</button>
                                <button type="button" data-action="toggle" data-id="${link.id}" data-status="${link.status || 'active'}" class="idx-btn-secondary">${(link.status || 'active') === 'inactive' ? 'Activate' : 'Deactivate'}</button>
                                <button type="button" data-action="copy" data-id="${link.id}" class="idx-btn-secondary">Copy</button>
                                <button type="button" data-action="delete" data-id="${link.id}" class="rounded-md border border-rose-300 px-2 py-1 text-xs font-medium text-rose-700 dark:border-rose-800 dark:text-rose-300">Delete</button>
                            </div>
                        </td>
                    `;
                    rowsEl.appendChild(row);
                });
            };

            const loadSearches = async () => {
                const response = await fetch('/agent/searches', { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading searches');
                }
                const payload = await response.json();
                const searches = payload?.data || [];
                searchSelect.innerHTML = '<option value="">Optional: link to a saved search</option>';
                searches.forEach((search) => {
                    const option = document.createElement('option');
                    option.value = String(search.id);
                    option.textContent = search.name || `Search #${search.id}`;
                    searchSelect.appendChild(option);
                });
            };

            const loadShareLinks = async () => {
                const params = new URLSearchParams();
                if (String(filterKindEl?.value || '') !== '') {
                    params.set('template_kind', String(filterKindEl.value));
                }
                if (String(filterStatusEl?.value || '') !== '') {
                    params.set('status', String(filterStatusEl.value));
                }
                if (String(filterQueryEl?.value || '').trim() !== '') {
                    params.set('q', String(filterQueryEl.value).trim());
                }
                const queryString = params.toString();
                const url = queryString === '' ? '/agent/share-links' : `/agent/share-links?${queryString}`;
                const response = await fetch(url, { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading share links');
                }
                const payload = await response.json();
                shareLinks = payload?.data || [];
                renderRows();
            };

            const loadMetrics = async () => {
                const response = await fetch(`/agent/share-links/metrics?days=${metricsWindowDays}`, { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading metrics');
                }
                const payload = await response.json();
                const data = payload?.data || {};
                if (metricSeoTotalEl) {
                    metricSeoTotalEl.textContent = String(data.seo_total || 0);
                }
                if (metricSeoActiveEl) {
                    metricSeoActiveEl.textContent = String(data.seo_active_total || 0);
                }
                if (metricSeoInactiveEl) {
                    metricSeoInactiveEl.textContent = String(data.seo_inactive_total || 0);
                }
                metricCreatedInWindowEl.textContent = String(data.created_in_window || 0);
                metricVisitsInWindowEl.textContent = String(data.visits_in_window || 0);
                const createdDelta = Number(data.created_delta || 0);
                const visitsDelta = Number(data.visits_delta || 0);
                metricCreatedDeltaEl.textContent = `Delta: ${createdDelta >= 0 ? '+' : ''}${createdDelta}`;
                metricVisitsDeltaEl.textContent = `Delta: ${visitsDelta >= 0 ? '+' : ''}${visitsDelta}`;
                metricCreatedDeltaEl.classList.toggle('text-emerald-600', createdDelta > 0);
                metricCreatedDeltaEl.classList.toggle('text-rose-600', createdDelta < 0);
                metricVisitsDeltaEl.classList.toggle('text-emerald-600', visitsDelta > 0);
                metricVisitsDeltaEl.classList.toggle('text-rose-600', visitsDelta < 0);
                metricsWindowButtons.forEach((btn) => {
                    const days = Number(btn.dataset.metricsDays || 7);
                    btn.classList.toggle('bg-cyan-600', days === Number(data.window_days || 7));
                    btn.classList.toggle('text-white', days === Number(data.window_days || 7));
                });
            };

            const loadMetricsHistory = async () => {
                const response = await fetch(`/agent/share-links/metrics/history?days=${metricsWindowDays}`, { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading metrics history');
                }
                const payload = await response.json();
                const buckets = payload?.data?.buckets || [];
                if (buckets.length === 0) {
                    metricTrendRowsEl.innerHTML = '<p>No trend data yet.</p>';
                    return;
                }

                const maxCreated = Math.max(1, ...buckets.map((row) => Number(row.created || 0)));
                const maxVisits = Math.max(1, ...buckets.map((row) => Number(row.visits || 0)));
                metricTrendRowsEl.innerHTML = '';
                buckets.slice(-8).forEach((row) => {
                    const created = Number(row.created || 0);
                    const visits = Number(row.visits || 0);
                    const createdWidth = Math.max(4, Math.round((created / maxCreated) * 100));
                    const visitsWidth = Math.max(4, Math.round((visits / maxVisits) * 100));
                    const item = document.createElement('div');
                    item.className = 'grid grid-cols-[80px_1fr_1fr] items-center gap-2';
                    item.innerHTML = `
                        <span class="font-mono text-[11px]">${row.date.slice(5)}</span>
                        <span class="inline-block h-2 rounded bg-slate-400/70" style="width:${createdWidth}%"></span>
                        <span class="inline-block h-2 rounded bg-cyan-500/80" style="width:${visitsWidth}%"></span>
                    `;
                    metricTrendRowsEl.appendChild(item);
                });
            };

            const loadOperations = async () => {
                const response = await fetch('/agent/share-links/operations', { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading operations data');
                }
                const payload = await response.json();
                const data = payload?.data || {};
                opsPruneDaysEl.textContent = String(data.prune_days ?? '-');
                opsPruneCandidatesEl.textContent = String(data.prune_candidate_count ?? '-');
                opsScheduleEl.textContent = String(data.schedule ?? '-');
                opsCommandEl.textContent = String(data.prune_command ?? '-');
            };

            const loadSeoLandings = async () => {
                if (!featureFlags.seo_landing_pages || !seoLandingRowsEl || !seoLandingCountEl) {
                    return;
                }
                const response = await fetch('/agent/share-links/seo-landings', { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading SEO landing pages');
                }
                const payload = await response.json();
                const rows = payload?.data || [];
                seoLandingCountEl.textContent = `${rows.length} page${rows.length === 1 ? '' : 's'}`;
                if (rows.length === 0) {
                    seoLandingRowsEl.innerHTML = '<p>No SEO landing pages yet.</p>';
                    return;
                }
                seoLandingRowsEl.innerHTML = '';
                rows.slice(0, 10).forEach((row) => {
                    const item = document.createElement('div');
                    item.className = 'grid grid-cols-[1fr_auto] items-center gap-2 rounded border border-slate-200 px-2 py-1 dark:border-slate-700';
                    item.innerHTML = `
                        <a class="truncate text-cyan-700 hover:underline dark:text-cyan-300" href="${row.canonical_url}" target="_blank" rel="noopener">${row.agent_search_name || row.slug || row.canonical_path}</a>
                        <span class="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-300">${row.status || 'active'}</span>
                    `;
                    seoLandingRowsEl.appendChild(item);
                });
            };

            const estimateOperations = async () => {
                const days = Number(opsEstimateDaysEl?.value || 0);
                opsEstimateStatusEl.textContent = 'Estimating...';
                const response = await fetch(`/agent/share-links/operations/estimate?days=${days}`, { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    opsEstimateStatusEl.textContent = 'Estimate failed.';
                    return;
                }
                const payload = await response.json();
                const data = payload?.data || {};
                opsPruneCandidatesEl.textContent = String(data.prune_candidate_count ?? '-');
                opsEstimateStatusEl.textContent = `Cutoff: ${String(data.cutoff || '')}`;
            };

            createBtn.addEventListener('click', async () => {
                statusEl.textContent = 'Generating link...';
                try {
                    const chosenKind = String(templateKindEl?.value || 'standard');
                    if (chosenKind === 'seo_landing' && !featureFlags.seo_landing_pages) {
                        statusEl.textContent = 'Canonical landing links are disabled for this account.';
                        return;
                    }
                    const searchId = Number(searchSelect.value || 0);
                    const response = await fetch('/agent/share-links', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({
                            agent_search_id: searchId || null,
                            template_kind: chosenKind,
                            attribution_json: {
                                utm_source: utmSource.value.trim() || null,
                                utm_campaign: utmCampaign.value.trim() || null,
                            },
                        }),
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Generate failed.';
                        return;
                    }
                    const payload = await response.json();
                    statusEl.textContent = payload?.data?.reused_existing ? 'Reused existing SEO template link.' : 'Share link generated.';
                    await loadMetrics();
                    await loadMetricsHistory();
                    await loadOperations();
                    await loadShareLinks();
                    if (featureFlags.seo_landing_pages && seoLandingRowsEl) {
                        await loadSeoLandings();
                    }
                } catch (error) {
                    statusEl.textContent = 'Generate failed.';
                    console.error(error);
                }
            });

            refreshBtn.addEventListener('click', () => {
                statusEl.textContent = 'Refreshing...';
                const refreshLoads = [loadSearches(), loadShareLinks()];
                if (featureFlags.seo_landing_pages && seoLandingRowsEl) {
                    refreshLoads.push(loadSeoLandings());
                }
                Promise.all(refreshLoads)
                    .then(() => {
                        statusEl.textContent = 'Refreshed.';
                    })
                    .catch((error) => {
                        statusEl.textContent = 'Refresh failed.';
                        console.error(error);
                    });
            });

            rowsEl.addEventListener('click', async (event) => {
                const button = event.target.closest('button[data-action]');
                if (!(button instanceof HTMLElement)) {
                    return;
                }
                const id = Number(button.dataset.id || 0);
                if (!id) {
                    return;
                }
                const link = shareLinks.find((row) => Number(row.id) === id);
                if (!link) {
                    return;
                }

                if (button.dataset.action === 'copy') {
                    try {
                        await navigator.clipboard.writeText(link.url);
                        statusEl.textContent = 'Link copied.';
                    } catch {
                        statusEl.textContent = 'Copy failed.';
                    }
                    return;
                }

                if (button.dataset.action === 'copy-canonical-path') {
                    try {
                        await navigator.clipboard.writeText(link.canonical_path || '');
                        statusEl.textContent = 'Canonical path copied.';
                    } catch {
                        statusEl.textContent = 'Copy failed.';
                    }
                    return;
                }

                if (button.dataset.action === 'open-canonical') {
                    if (typeof link.canonical_url === 'string' && link.canonical_url !== '') {
                        window.open(link.canonical_url, '_blank', 'noopener');
                        statusEl.textContent = 'Opened canonical URL.';
                    } else {
                        statusEl.textContent = 'No canonical URL available.';
                    }
                    return;
                }

                if (button.dataset.action === 'delete') {
                    statusEl.textContent = 'Deleting...';
                    const response = await fetch(`/agent/share-links/${id}`, {
                        method: 'DELETE',
                        headers: {
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Delete failed.';
                        return;
                    }
                    statusEl.textContent = 'Deleted.';
                    await loadMetrics();
                    await loadMetricsHistory();
                    await loadOperations();
                    await loadShareLinks();
                    if (featureFlags.seo_landing_pages && seoLandingRowsEl) {
                        await loadSeoLandings();
                    }
                }

                if (button.dataset.action === 'toggle') {
                    const current = button.dataset.status || 'active';
                    const next = current === 'inactive' ? 'active' : 'inactive';
                    statusEl.textContent = `${next === 'active' ? 'Activating' : 'Deactivating'}...`;
                    const response = await fetch(`/agent/share-links/${id}`, {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({ status: next }),
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Status update failed.';
                        return;
                    }
                    statusEl.textContent = `Link ${next}.`;
                    await loadMetrics();
                    await loadMetricsHistory();
                    await loadOperations();
                    await loadShareLinks();
                    if (featureFlags.seo_landing_pages && seoLandingRowsEl) {
                        await loadSeoLandings();
                    }
                }
            });

            const initialLoads = [loadSearches(), loadShareLinks(), loadMetrics(), loadMetricsHistory(), loadOperations()];
            if (featureFlags.seo_landing_pages && seoLandingRowsEl) {
                initialLoads.push(loadSeoLandings());
            }
            Promise.all(initialLoads)
                .then(() => {
                    statusEl.textContent = 'Marketing links ready.';
                })
                .catch((error) => {
                    statusEl.textContent = 'Failed to load marketing data.';
                    console.error(error);
                });

            [filterKindEl, filterStatusEl, filterQueryEl].forEach((el) => {
                el?.addEventListener('change', () => {
                    loadShareLinks().catch((error) => {
                        statusEl.textContent = 'Filter refresh failed.';
                        console.error(error);
                    });
                });
            });

            if (seoTemplatesPresetBtn) {
                seoTemplatesPresetBtn.addEventListener('click', () => {
                    if (filterKindEl) {
                        filterKindEl.value = 'seo_landing';
                    }
                    if (filterStatusEl) {
                        filterStatusEl.value = 'active';
                    }
                    loadShareLinks().catch((error) => {
                        statusEl.textContent = 'Filter preset failed.';
                        console.error(error);
                    });
                });
            }

            clearShareFiltersBtn?.addEventListener('click', () => {
                if (filterKindEl) {
                    filterKindEl.value = '';
                }
                if (filterStatusEl) {
                    filterStatusEl.value = '';
                }
                if (filterQueryEl) {
                    filterQueryEl.value = '';
                }
                loadShareLinks().catch((error) => {
                    statusEl.textContent = 'Clear filters failed.';
                    console.error(error);
                });
            });

            exportShareCsvBtn?.addEventListener('click', () => {
                const params = new URLSearchParams();
                if (String(filterKindEl?.value || '') !== '') {
                    params.set('template_kind', String(filterKindEl.value));
                }
                if (String(filterStatusEl?.value || '') !== '') {
                    params.set('status', String(filterStatusEl.value));
                }
                if (String(filterQueryEl?.value || '').trim() !== '') {
                    params.set('q', String(filterQueryEl.value).trim());
                }
                const queryString = params.toString();
                const url = queryString === '' ? '/agent/share-links/export.csv' : `/agent/share-links/export.csv?${queryString}`;
                window.location.href = url;
            });

            metricsWindowButtons.forEach((btn) => {
                btn.addEventListener('click', () => {
                    metricsWindowDays = Number(btn.dataset.metricsDays || 7);
                    Promise.all([loadMetrics(), loadMetricsHistory()]).catch((error) => {
                        statusEl.textContent = 'Metrics refresh failed.';
                        console.error(error);
                    });
                });
            });

            exportTrendCsvBtn?.addEventListener('click', () => {
                window.location.href = `/agent/share-links/metrics/history.csv?days=${metricsWindowDays}`;
            });

            opsEstimateBtn?.addEventListener('click', () => {
                estimateOperations().catch((error) => {
                    opsEstimateStatusEl.textContent = 'Estimate failed.';
                    console.error(error);
                });
            });

            if (featureFlags.widgets) {
                const widgetTypeEl = document.getElementById('idxWidgetType');
                const widgetSearchSelect = document.getElementById('idxWidgetSearchSelect');
                const widgetMaxListings = document.getElementById('idxWidgetMaxListings');
                const widgetThemeEl = document.getElementById('idxWidgetTheme');
                const widgetGenerateBtn = document.getElementById('idxWidgetGenerateBtn');
                const widgetCopyBtn = document.getElementById('idxWidgetCopyBtn');
                const widgetStatusEl = document.getElementById('idxWidgetStatus');
                const widgetPreviewContainer = document.getElementById('idxWidgetPreviewContainer');
                const widgetPreviewCode = document.getElementById('idxWidgetPreviewCode');
                let widgetEmbedCode = '';

                const loadWidgetSearches = async () => {
                    if (!widgetSearchSelect) {
                        return;
                    }
                    const response = await fetch('/agent/searches', { headers: { 'Accept': 'application/json' } });
                    if (!response.ok) return;
                    const payload = await response.json();
                    const searches = payload?.data || [];
                    widgetSearchSelect.innerHTML = '<option value="">None (default)</option>';
                    searches.forEach((search) => {
                        const option = document.createElement('option');
                        option.value = String(search.id);
                        option.textContent = search.name || `Search #${search.id}`;
                        widgetSearchSelect.appendChild(option);
                    });
                };

                widgetGenerateBtn?.addEventListener('click', () => {
                    const type = widgetTypeEl?.value || 'search';
                    const searchId = widgetSearchSelect?.value || '';
                    const maxListings = widgetMaxListings?.value || '6';
                    const theme = widgetThemeEl?.value || 'light';
                    const baseUrl = '{{ rtrim(config("app.url"), "/") }}';

                    const attrs = [
                        `data-quantyra-widget="${type}"`,
                        `data-theme="${theme}"`,
                        `data-max-listings="${maxListings}"`,
                    ];
                    if (searchId) attrs.push(`data-search-id="${searchId}"`);

                    widgetEmbedCode = `<div ${attrs.join(' ')}></div>\n<script src="${baseUrl}/widget/loader.js" async><\/script>`;

                    if (widgetPreviewCode) {
                        widgetPreviewCode.textContent = widgetEmbedCode;
                    }
                    widgetPreviewContainer?.classList.remove('hidden');
                    widgetCopyBtn?.classList.remove('hidden');
                    if (widgetStatusEl) {
                        widgetStatusEl.textContent = 'Embed code generated.';
                    }
                });

                widgetCopyBtn?.addEventListener('click', async () => {
                    try {
                        await navigator.clipboard.writeText(widgetEmbedCode);
                        if (widgetStatusEl) {
                            widgetStatusEl.textContent = 'Embed code copied to clipboard.';
                        }
                    } catch {
                        if (widgetStatusEl) {
                            widgetStatusEl.textContent = 'Copy failed.';
                        }
                    }
                });

                loadWidgetSearches().catch(() => {});
            }
        });
    </script>
</x-filament-panels::page>
