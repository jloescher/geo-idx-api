<x-filament-panels::page>
    <div class="idx-agent-shell space-y-6">
        <div class="idx-panel">
            <h3 class="idx-panel-title">Email alerts</h3>
            <p class="idx-panel-subtitle">
                Listing, market activity, and home value alerts with schedule controls and run history preview.
            </p>

            <div class="mt-4 grid gap-4 lg:grid-cols-3">
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Alert name
                    <input id="idxAlertName" type="text" class="idx-input" placeholder="Morning listings pulse" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Type
                    <select id="idxAlertType" class="idx-input">
                        <option value="listing">Listing</option>
                        <option value="market_activity">Market activity</option>
                        <option value="home_value">Home value</option>
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Cadence
                    <select id="idxAlertCadence" class="idx-input">
                        <option value="daily">Daily</option>
                        <option value="weekly">Weekly</option>
                        <option value="monthly">Monthly</option>
                    </select>
                </label>
            </div>
            <p class="mt-2 text-[11px] leading-relaxed text-slate-500 dark:text-slate-400">
                <strong class="text-slate-600 dark:text-slate-300">Listing</strong> alerts use a saved search for matches.
                <strong class="text-slate-600 dark:text-slate-300">Market activity</strong> and <strong class="text-slate-600 dark:text-slate-300">home value</strong> alerts can run on a cadence without a saved search; attach a saved search later when you want MLS-scoped criteria.
            </p>
            <div class="mt-3 flex flex-wrap items-center gap-2">
                <button id="idxAlertCreateBtn" type="button" class="idx-btn-primary">
                    Create alert
                </button>
                <span id="idxAlertStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
            </div>
        </div>

        <div class="grid gap-4 md:grid-cols-5">
            <div class="rounded-xl border border-slate-200/80 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
                <p class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Total alerts</p>
                <p id="idxAlertKpiTotal" class="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">0</p>
            </div>
            <div class="rounded-xl border border-slate-200/80 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
                <p class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Active</p>
                <p id="idxAlertKpiActive" class="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">0</p>
            </div>
            <div class="rounded-xl border border-slate-200/80 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
                <p class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Listing</p>
                <p id="idxAlertKpiListing" class="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">0</p>
            </div>
            <div class="rounded-xl border border-slate-200/80 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
                <p class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Market activity</p>
                <p id="idxAlertKpiMarketActivity" class="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">0</p>
            </div>
            <div class="rounded-xl border border-slate-200/80 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
                <p class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Home value</p>
                <p id="idxAlertKpiHomeValue" class="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">0</p>
            </div>
        </div>

        <div class="idx-panel">
            <div class="mb-2 flex items-center justify-between">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">Run history (30d)</h4>
                <span id="idxAlertHistoryMeta" class="text-xs text-slate-500 dark:text-slate-400">Loading...</span>
            </div>
            <div class="overflow-x-auto">
                <table class="w-full text-left text-sm">
                    <thead class="border-b border-slate-200 text-xs uppercase text-slate-500 dark:border-slate-700 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-4 font-medium">Date</th>
                            <th class="py-2 pr-4 font-medium">Total runs</th>
                            <th class="py-2 pr-4 font-medium">Listing</th>
                            <th class="py-2 pr-4 font-medium">Market activity</th>
                            <th class="py-2 font-medium">Home value</th>
                        </tr>
                    </thead>
                    <tbody id="idxAlertHistoryRows">
                        <tr><td colspan="5" class="py-4 text-slate-500">Loading history...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>

        <div class="idx-panel">
            <div class="mb-4 flex items-center justify-between">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">Alert templates</h4>
                <button id="idxTemplateRefreshBtn" type="button" class="idx-btn-secondary">Refresh templates</button>
            </div>
            <div class="mb-4 grid gap-4 lg:grid-cols-3">
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Template name
                    <input id="idxTemplateName" type="text" class="idx-input" placeholder="Weekly listing digest template" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Template type
                    <select id="idxTemplateType" class="idx-input">
                        <option value="listing">Listing</option>
                        <option value="market_activity">Market activity</option>
                        <option value="home_value">Home value</option>
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Template cadence
                    <select id="idxTemplateCadence" class="idx-input">
                        <option value="daily">Daily</option>
                        <option value="weekly">Weekly</option>
                        <option value="monthly">Monthly</option>
                    </select>
                </label>
            </div>
            <div class="mb-4 flex flex-wrap items-center gap-2">
                <button id="idxTemplateCreateBtn" type="button" class="rounded-md bg-cyan-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-cyan-500">
                    Create template
                </button>
            </div>
            <div class="overflow-x-auto">
                <table class="idx-data-table w-full text-left text-sm">
                    <thead class="border-b border-slate-200 text-xs uppercase text-slate-500 dark:border-slate-700 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-4 font-medium">Name</th>
                            <th class="py-2 pr-4 font-medium">Type</th>
                            <th class="py-2 pr-4 font-medium">Cadence</th>
                            <th class="py-2 pr-4 font-medium">Usage</th>
                            <th class="py-2 pr-4 font-medium">Last used</th>
                            <th class="py-2 font-medium">Actions</th>
                        </tr>
                    </thead>
                    <tbody id="idxTemplateRows">
                        <tr><td colspan="6" class="py-4 text-slate-500">Loading templates...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>

        <div class="idx-panel">
            <div class="mb-3 flex items-center justify-between">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">Current alerts</h4>
                <div class="flex flex-col items-end gap-2 sm:flex-row sm:items-center">
                    <div class="flex flex-wrap items-center gap-2">
                        <span class="text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Type</span>
                        <div class="flex flex-wrap gap-1 border-b border-slate-200 dark:border-slate-700">
                            <button type="button" data-alert-type="all" class="idx-alert-type-tab border-b-2 border-cyan-600 px-2 py-1 text-xs font-semibold text-cyan-700 dark:text-cyan-400">All types</button>
                            <button type="button" data-alert-type="listing" class="idx-alert-type-tab border-b-2 border-transparent px-2 py-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Listing</button>
                            <button type="button" data-alert-type="market_activity" class="idx-alert-type-tab border-b-2 border-transparent px-2 py-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Market</button>
                            <button type="button" data-alert-type="home_value" class="idx-alert-type-tab border-b-2 border-transparent px-2 py-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Home value</button>
                        </div>
                    </div>
                    <div class="flex flex-wrap items-center gap-2">
                        <span class="text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Status</span>
                        <div class="flex gap-1 border-b border-slate-200 dark:border-slate-700">
                            <button type="button" data-alert-status="all" class="idx-alert-status-tab border-b-2 border-cyan-600 px-2 py-1 text-xs font-semibold text-cyan-700 dark:text-cyan-400">All</button>
                            <button type="button" data-alert-status="active" class="idx-alert-status-tab border-b-2 border-transparent px-2 py-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Active</button>
                            <button type="button" data-alert-status="paused" class="idx-alert-status-tab border-b-2 border-transparent px-2 py-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Paused</button>
                        </div>
                    </div>
                    <button id="idxAlertRefreshBtn" type="button" class="idx-btn-secondary text-xs">Refresh</button>
                </div>
            </div>
            <div class="overflow-x-auto">
                <table class="idx-data-table w-full text-left text-sm">
                    <thead class="border-b border-slate-200 text-xs uppercase text-slate-500 dark:border-slate-700 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-4 font-medium">Name</th>
                            <th class="py-2 pr-4 font-medium">Type</th>
                            <th class="py-2 pr-4 font-medium">Status</th>
                            <th class="py-2 pr-4 font-medium">Schedule</th>
                            <th class="py-2 pr-4 font-medium">Next run</th>
                            <th class="py-2 pr-4 font-medium">Runs</th>
                            <th class="py-2 font-medium">Actions</th>
                        </tr>
                    </thead>
                    <tbody id="idxAlertRows">
                        <tr><td colspan="7" class="py-4 text-slate-500">Loading alerts...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>

        <div id="idxAlertRunsOverlay" class="hidden fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4" role="dialog" aria-modal="true" aria-labelledby="idxAlertRunsTitle">
            <div class="max-h-[85vh] w-full max-w-3xl overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl dark:border-slate-700 dark:bg-slate-900">
                <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-700">
                    <h4 id="idxAlertRunsTitle" class="text-sm font-semibold text-slate-900 dark:text-slate-100">Alert runs</h4>
                    <button type="button" id="idxAlertRunsClose" class="rounded-md px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800">Close</button>
                </div>
                <div id="idxAlertRunsBody" class="overflow-y-auto p-4" style="max-height: calc(85vh - 3.5rem);"></div>
            </div>
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const statusEl = document.getElementById('idxAlertStatus');
            const rowsEl = document.getElementById('idxAlertRows');
            const createBtn = document.getElementById('idxAlertCreateBtn');
            const refreshBtn = document.getElementById('idxAlertRefreshBtn');
            const nameEl = document.getElementById('idxAlertName');
            const typeEl = document.getElementById('idxAlertType');
            const cadenceEl = document.getElementById('idxAlertCadence');
            const templateRowsEl = document.getElementById('idxTemplateRows');
            const templateRefreshBtn = document.getElementById('idxTemplateRefreshBtn');
            const templateCreateBtn = document.getElementById('idxTemplateCreateBtn');
            const templateNameEl = document.getElementById('idxTemplateName');
            const templateTypeEl = document.getElementById('idxTemplateType');
            const templateCadenceEl = document.getElementById('idxTemplateCadence');
            const kpiTotalEl = document.getElementById('idxAlertKpiTotal');
            const kpiActiveEl = document.getElementById('idxAlertKpiActive');
            const kpiListingEl = document.getElementById('idxAlertKpiListing');
            const kpiMarketActivityEl = document.getElementById('idxAlertKpiMarketActivity');
            const kpiHomeValueEl = document.getElementById('idxAlertKpiHomeValue');
            const historyMetaEl = document.getElementById('idxAlertHistoryMeta');
            const historyRowsEl = document.getElementById('idxAlertHistoryRows');
            const runsOverlay = document.getElementById('idxAlertRunsOverlay');
            const runsTitle = document.getElementById('idxAlertRunsTitle');
            const runsBody = document.getElementById('idxAlertRunsBody');
            const runsCloseBtn = document.getElementById('idxAlertRunsClose');
            let alerts = [];
            let templates = [];
            let alertStatusFilter = 'all';
            let alertTypeFilter = 'all';

            const switchAlertStatusTab = (status) => {
                alertStatusFilter = status;
                document.querySelectorAll('.idx-alert-status-tab').forEach((btn) => {
                    const active = btn.dataset.alertStatus === status;
                    btn.classList.toggle('border-cyan-600', active);
                    btn.classList.toggle('text-cyan-700', active);
                    btn.classList.toggle('dark:text-cyan-400', active);
                    btn.classList.toggle('font-semibold', active);
                    btn.classList.toggle('border-transparent', !active);
                    btn.classList.toggle('text-slate-500', !active);
                    btn.classList.toggle('font-medium', !active);
                });
                renderFilteredAlerts();
            };

            const switchAlertTypeTab = (type) => {
                alertTypeFilter = type || 'all';
                document.querySelectorAll('.idx-alert-type-tab').forEach((btn) => {
                    const active = btn.dataset.alertType === alertTypeFilter;
                    btn.classList.toggle('border-cyan-600', active);
                    btn.classList.toggle('text-cyan-700', active);
                    btn.classList.toggle('dark:text-cyan-400', active);
                    btn.classList.toggle('font-semibold', active);
                    btn.classList.toggle('border-transparent', !active);
                    btn.classList.toggle('text-slate-500', !active);
                    btn.classList.toggle('font-medium', !active);
                });
                refreshAlerts().catch((error) => {
                    statusEl.textContent = 'Failed to load alerts for type filter.';
                    console.error(error);
                });
            };

            document.querySelectorAll('.idx-alert-status-tab').forEach((btn) => {
                btn.addEventListener('click', () => switchAlertStatusTab(btn.dataset.alertStatus));
            });

            document.querySelectorAll('.idx-alert-type-tab').forEach((btn) => {
                btn.addEventListener('click', () => switchAlertTypeTab(btn.dataset.alertType));
            });

            const renderFilteredAlerts = () => {
                const filtered = alertStatusFilter === 'all'
                    ? alerts
                    : alerts.filter((a) => (a.status || 'active') === alertStatusFilter);
                renderAlertRows(filtered);
            };

            const runCountText = (alert) => {
                const total = typeof alert.runs_count === 'number'
                    ? alert.runs_count
                    : (Array.isArray(alert.runs) ? alert.runs.length : 0);
                if (total === 0) {
                    return '0';
                }
                const runs = Array.isArray(alert.runs) ? alert.runs : [];
                const last = runs.length > 0 ? runs[0] : null;
                const ranAt = last?.ran_at ? new Date(last.ran_at).toLocaleString() : 'n/a';
                return `${total} (last: ${ranAt})`;
            };

            const renderAlertRows = (filteredAlerts) => {
                if (!filteredAlerts || filteredAlerts.length === 0) {
                    rowsEl.innerHTML = '<tr><td colspan="7" class="py-4 text-slate-500">No alerts match this filter.</td></tr>';
                    return;
                }
                rowsEl.innerHTML = '';
                filteredAlerts.forEach((alert) => {
                    const schedule = alert.schedule_json?.cadence ?? 'n/a';
                    const nextRun = alert.next_run_at ? new Date(alert.next_run_at).toLocaleString() : 'n/a';
                    const isActive = alert.status === 'active';
                    const tr = document.createElement('tr');
                    tr.className = 'border-b border-slate-100 dark:border-slate-800';
                    tr.innerHTML = `
                        <td class="py-2 pr-4">${alert.name ?? ''}</td>
                        <td class="py-2 pr-4">${alert.alert_type ?? ''}</td>
                        <td class="py-2 pr-4">${alert.status ?? ''}</td>
                        <td class="py-2 pr-4">${schedule}</td>
                        <td class="py-2 pr-4">${nextRun}</td>
                        <td class="py-2 pr-4">${runCountText(alert)}</td>
                        <td class="py-2">
                            <div class="flex flex-wrap gap-2">
                                <button type="button" data-action="view-runs" data-id="${alert.id}" class="rounded-md border border-slate-300 px-2 py-1 text-xs font-medium text-slate-700 dark:border-slate-600 dark:text-slate-200">View runs</button>
                                <button type="button" data-action="toggle" data-id="${alert.id}" class="rounded-md px-2 py-1 text-xs font-medium ${isActive ? 'bg-amber-600 text-white' : 'bg-emerald-600 text-white'}">${isActive ? 'Pause' : 'Resume'}</button>
                                <button type="button" data-action="delete" data-id="${alert.id}" class="rounded-md border border-rose-300 px-2 py-1 text-xs font-medium text-rose-700 dark:border-rose-800 dark:text-rose-300">Delete</button>
                            </div>
                        </td>
                    `;
                    rowsEl.appendChild(tr);
                });
            };

            const renderTemplateRows = () => {
                if (templates.length === 0) {
                    templateRowsEl.innerHTML = '<tr><td colspan="6" class="py-4 text-slate-500">No templates yet.</td></tr>';
                    return;
                }
                templateRowsEl.innerHTML = '';
                templates.forEach((template) => {
                    const cadence = template.schedule_json?.cadence ?? template.body_json?.schedule?.cadence ?? 'n/a';
                    const usage = template.usage_count ?? 0;
                    const lastUsed = template.last_used_at ? new Date(template.last_used_at).toLocaleDateString() : '—';
                    const tr = document.createElement('tr');
                    tr.className = 'border-b border-slate-100 dark:border-slate-800';
                    tr.innerHTML = `
                        <td class="py-2 pr-4">${template.name ?? ''}</td>
                        <td class="py-2 pr-4">${template.template_type ?? ''}</td>
                        <td class="py-2 pr-4">${cadence}</td>
                        <td class="py-2 pr-4">${usage}</td>
                        <td class="py-2 pr-4">${lastUsed}</td>
                        <td class="py-2">
                            <div class="flex flex-wrap gap-2">
                                <button type="button" data-template-action="use" data-id="${template.id}" class="rounded-md bg-emerald-600 px-2 py-1 text-xs font-medium text-white">Use template</button>
                                <button type="button" data-template-action="delete" data-id="${template.id}" class="rounded-md border border-rose-300 px-2 py-1 text-xs font-medium text-rose-700 dark:border-rose-800 dark:text-rose-300">Delete</button>
                            </div>
                        </td>
                    `;
                    templateRowsEl.appendChild(tr);
                });
            };

            const refreshAlerts = async () => {
                const params = new URLSearchParams();
                if (alertTypeFilter !== 'all') {
                    params.set('alert_type', alertTypeFilter);
                }
                const url = params.toString() === '' ? '/agent/alerts' : `/agent/alerts?${params.toString()}`;
                const response = await fetch(url, { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading alerts');
                }
                const payload = await response.json();
                alerts = payload?.data || [];
                renderFilteredAlerts();
            };

            const refreshTemplates = async () => {
                const response = await fetch('/agent/alert-templates', { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading templates');
                }
                const payload = await response.json();
                templates = payload?.data || [];
                renderTemplateRows();
            };

            const refreshSummary = async () => {
                const response = await fetch('/agent/alerts/summary', { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading alert summary');
                }
                const payload = await response.json();
                const summary = payload?.data || {};
                const byType = summary.by_type || {};
                kpiTotalEl.textContent = String(summary.total || 0);
                kpiActiveEl.textContent = String(summary.active || 0);
                kpiListingEl.textContent = String(byType.listing || 0);
                kpiMarketActivityEl.textContent = String(byType.market_activity || 0);
                kpiHomeValueEl.textContent = String(byType.home_value || 0);
            };

            const closeRunsOverlay = () => {
                runsOverlay.classList.add('hidden');
            };

            const showAlertRunsDetail = async (alertId, alertName) => {
                runsTitle.textContent = alertName ? `Runs — ${alertName}` : `Runs — alert #${alertId}`;
                runsBody.textContent = '';
                const loading = document.createElement('p');
                loading.className = 'text-sm text-slate-500';
                loading.textContent = 'Loading...';
                runsBody.appendChild(loading);
                runsOverlay.classList.remove('hidden');

                const response = await fetch(`/agent/alerts/${alertId}/runs?limit=50`, {
                    headers: { Accept: 'application/json' },
                });
                runsBody.textContent = '';
                if (!response.ok) {
                    const err = document.createElement('p');
                    err.className = 'text-sm text-rose-600 dark:text-rose-400';
                    err.textContent = 'Could not load runs.';
                    runsBody.appendChild(err);
                    return;
                }
                const payload = await response.json();
                const items = Array.isArray(payload?.data) ? payload.data : [];
                if (items.length === 0) {
                    const empty = document.createElement('p');
                    empty.className = 'text-sm text-slate-500';
                    empty.textContent = 'No runs recorded yet.';
                    runsBody.appendChild(empty);
                    return;
                }
                const table = document.createElement('table');
                table.className = 'w-full text-left text-xs text-slate-800 dark:text-slate-200';
                const thead = document.createElement('thead');
                thead.className = 'border-b border-slate-200 text-[10px] uppercase text-slate-500 dark:border-slate-700 dark:text-slate-400';
                thead.innerHTML = '<tr><th class="py-2 pr-3 font-medium">Ran at</th><th class="py-2 pr-3 font-medium">Status</th><th class="py-2 font-medium">Metadata</th></tr>';
                table.appendChild(thead);
                const tbody = document.createElement('tbody');
                items.forEach((run) => {
                    const tr = document.createElement('tr');
                    tr.className = 'border-b border-slate-100 align-top dark:border-slate-800';
                    const tdWhen = document.createElement('td');
                    tdWhen.className = 'py-2 pr-3 whitespace-nowrap';
                    tdWhen.textContent = run.ran_at ? new Date(run.ran_at).toLocaleString() : 'n/a';
                    const tdStatus = document.createElement('td');
                    tdStatus.className = 'py-2 pr-3';
                    tdStatus.textContent = run.status ?? '';
                    const tdMeta = document.createElement('td');
                    tdMeta.className = 'py-2';
                    const pre = document.createElement('pre');
                    pre.className = 'max-h-48 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-slate-600 dark:text-slate-400';
                    pre.textContent = JSON.stringify(run.metadata ?? {}, null, 2);
                    tdMeta.appendChild(pre);
                    tr.appendChild(tdWhen);
                    tr.appendChild(tdStatus);
                    tr.appendChild(tdMeta);
                    tbody.appendChild(tr);
                });
                table.appendChild(tbody);
                runsBody.appendChild(table);
            };

            runsCloseBtn.addEventListener('click', closeRunsOverlay);
            runsOverlay.addEventListener('click', (event) => {
                if (event.target === runsOverlay) {
                    closeRunsOverlay();
                }
            });

            const refreshHistory = async () => {
                const response = await fetch('/agent/alerts/history?days=30', { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading alert history');
                }
                const payload = await response.json();
                const data = payload?.data || {};
                const buckets = data.buckets || [];
                const meta = data.meta || {};
                historyMetaEl.textContent = `${meta.total_runs || 0} total runs`;
                const rows = buckets.slice(-7).reverse();
                if (rows.length === 0) {
                    historyRowsEl.innerHTML = '<tr><td colspan="5" class="py-4 text-slate-500">No history yet.</td></tr>';
                    return;
                }
                historyRowsEl.innerHTML = '';
                rows.forEach((bucket) => {
                    const tr = document.createElement('tr');
                    tr.className = 'border-b border-slate-100 dark:border-slate-800';
                    tr.innerHTML = `
                        <td class="py-2 pr-4">${bucket.date ?? ''}</td>
                        <td class="py-2 pr-4">${bucket.total_runs ?? 0}</td>
                        <td class="py-2 pr-4">${bucket.listing_runs ?? 0}</td>
                        <td class="py-2 pr-4">${bucket.market_activity_runs ?? 0}</td>
                        <td class="py-2">${bucket.home_value_runs ?? 0}</td>
                    `;
                    historyRowsEl.appendChild(tr);
                });
            };

            createBtn.addEventListener('click', async () => {
                const name = nameEl.value.trim();
                if (name === '') {
                    statusEl.textContent = 'Provide an alert name.';
                    return;
                }
                statusEl.textContent = 'Creating alert...';
                try {
                    const response = await fetch('/agent/alerts', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({
                            name,
                            alert_type: typeEl.value,
                            status: 'active',
                            schedule_json: { cadence: cadenceEl.value },
                        }),
                    });
                    if (!response.ok) {
                        const payload = await response.json();
                        statusEl.textContent = 'Create failed.';
                        console.error(payload);
                        return;
                    }
                    nameEl.value = '';
                    statusEl.textContent = 'Alert created.';
                    await refreshAlerts();
                    await refreshSummary();
                    await refreshHistory();
                } catch (error) {
                    statusEl.textContent = 'Create failed.';
                    console.error(error);
                }
            });

            refreshBtn.addEventListener('click', () => {
                statusEl.textContent = 'Refreshing...';
                refreshAlerts()
                    .then(() => {
                        return Promise.all([refreshSummary(), refreshHistory()]);
                    })
                    .then(() => {
                        statusEl.textContent = 'Refreshed.';
                    })
                    .catch((error) => {
                        statusEl.textContent = 'Refresh failed.';
                        console.error(error);
                    });
            });

            templateRefreshBtn.addEventListener('click', () => {
                statusEl.textContent = 'Refreshing templates...';
                refreshTemplates()
                    .then(() => {
                        statusEl.textContent = 'Templates refreshed.';
                    })
                    .catch((error) => {
                        statusEl.textContent = 'Template refresh failed.';
                        console.error(error);
                    });
            });

            templateCreateBtn.addEventListener('click', async () => {
                const name = templateNameEl.value.trim();
                if (name === '') {
                    statusEl.textContent = 'Provide a template name.';
                    return;
                }
                statusEl.textContent = 'Creating template...';
                try {
                    const response = await fetch('/agent/alert-templates', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({
                            name,
                            template_type: templateTypeEl.value,
                            body_json: {
                                status: 'active',
                                schedule: { cadence: templateCadenceEl.value },
                            },
                            schedule_json: {
                                cadence: templateCadenceEl.value,
                            },
                        }),
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Template create failed.';
                        return;
                    }
                    templateNameEl.value = '';
                    statusEl.textContent = 'Template created.';
                    await refreshTemplates();
                } catch (error) {
                    statusEl.textContent = 'Template create failed.';
                    console.error(error);
                }
            });

            templateRowsEl.addEventListener('click', async (event) => {
                const button = event.target.closest('button[data-template-action]');
                if (!(button instanceof HTMLElement)) {
                    return;
                }
                const action = button.dataset.templateAction;
                const id = Number(button.dataset.id || 0);
                if (!id) {
                    return;
                }
                const template = templates.find((row) => Number(row.id) === id);
                if (!template) {
                    return;
                }

                if (action === 'delete') {
                    statusEl.textContent = 'Deleting template...';
                    const response = await fetch(`/agent/alert-templates/${id}`, {
                        method: 'DELETE',
                        headers: {
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Template delete failed.';
                        return;
                    }
                    statusEl.textContent = 'Template deleted.';
                    await refreshTemplates();
                    return;
                }

                if (action === 'use') {
                    const alertName = window.prompt('New alert name from template', `${template.name} alert`);
                    if (!alertName || alertName.trim() === '') {
                        return;
                    }
                    statusEl.textContent = 'Creating alert from template...';
                    const response = await fetch('/agent/alerts/from-template', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({
                            template_id: id,
                            name: alertName.trim(),
                        }),
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Template apply failed.';
                        return;
                    }
                    statusEl.textContent = 'Alert created from template.';
                    await refreshAlerts();
                    await refreshSummary();
                    await refreshHistory();
                }
            });

            rowsEl.addEventListener('click', async (event) => {
                const button = event.target.closest('button[data-action]');
                if (!(button instanceof HTMLElement)) {
                    return;
                }
                const action = button.dataset.action;
                const id = Number(button.dataset.id || 0);
                if (!id) {
                    return;
                }
                const alert = alerts.find((row) => Number(row.id) === id);
                if (!alert) {
                    return;
                }

                if (action === 'view-runs') {
                    statusEl.textContent = 'Loading run history...';
                    try {
                        await showAlertRunsDetail(id, alert.name);
                        statusEl.textContent = 'Runs loaded.';
                    } catch (error) {
                        statusEl.textContent = 'Failed to load runs.';
                        console.error(error);
                    }
                    return;
                }

                if (action === 'delete') {
                    statusEl.textContent = 'Deleting alert...';
                    const response = await fetch(`/agent/alerts/${id}`, {
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
                    statusEl.textContent = 'Alert deleted.';
                    await refreshAlerts();
                    await refreshSummary();
                    await refreshHistory();
                    return;
                }

                if (action === 'toggle') {
                    const nextStatus = alert.status === 'active' ? 'paused' : 'active';
                    statusEl.textContent = 'Updating status...';
                    const response = await fetch(`/agent/alerts/${id}`, {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({
                            name: alert.name,
                            alert_type: alert.alert_type,
                            status: nextStatus,
                            schedule_json: alert.schedule_json ?? { cadence: 'daily' },
                            agent_search_id: alert.agent_search_id ?? null,
                            next_run_at: alert.next_run_at ?? null,
                        }),
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Status update failed.';
                        return;
                    }
                    statusEl.textContent = 'Status updated.';
                    await refreshAlerts();
                    await refreshSummary();
                    await refreshHistory();
                }
            });

            Promise.all([refreshAlerts(), refreshTemplates(), refreshSummary(), refreshHistory()])
                .then(() => {
                    statusEl.textContent = 'Alerts loaded.';
                })
                .catch((error) => {
                    statusEl.textContent = 'Failed to load alerts.';
                    console.error(error);
                });
        });
    </script>
</x-filament-panels::page>
