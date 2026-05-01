<x-filament-panels::page>
    <div class="idx-agent-shell space-y-6">
        <div class="idx-panel">
            <div class="flex items-center justify-between">
                <div>
                    <h3 class="idx-panel-title">Agent Dashboard</h3>
                    <p class="idx-panel-subtitle">Live KPI overview and recent activity stream.</p>
                </div>
                <div class="flex items-center gap-2">
                    <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">Period</span>
                    <button type="button" data-period="7" class="idx-period-btn idx-btn-secondary">7D</button>
                    <button type="button" data-period="30" class="idx-period-btn idx-btn-secondary">30D</button>
                    <button type="button" data-period="90" class="idx-period-btn idx-btn-secondary">90D</button>
                    <button id="idxAgentDashboardRefresh" type="button" class="idx-btn-secondary">Refresh</button>
                </div>
            </div>
            <p id="idxAgentDashboardStatus" class="mt-2 text-xs text-slate-500 dark:text-slate-400"></p>
        </div>

        <div class="grid gap-4 md:grid-cols-4">
            <div class="idx-kpi-card">
                <p class="idx-kpi-label">Contacts</p>
                <p id="idxAgentKpiContacts" class="idx-kpi-value">0</p>
            </div>
            <div class="idx-kpi-card">
                <p class="idx-kpi-label">Active Alerts</p>
                <p id="idxAgentKpiAlerts" class="idx-kpi-value">0</p>
            </div>
            <div class="idx-kpi-card">
                <p class="idx-kpi-label">Saved Searches</p>
                <p id="idxAgentKpiSearches" class="idx-kpi-value">0</p>
            </div>
            <div class="idx-kpi-card">
                <p class="idx-kpi-label">Active Share Links</p>
                <p id="idxAgentKpiShareLinks" class="idx-kpi-value">0</p>
            </div>
        </div>

        <div class="idx-panel">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                <div>
                    <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">Upcoming alerts</h4>
                    <p class="text-xs text-slate-500 dark:text-slate-400">Active alerts with the soonest scheduled run first.</p>
                </div>
                <a href="{{ url('/filament-dashboard/agent-alerts-workspace') }}" class="text-xs font-semibold text-cyan-700 hover:text-cyan-600 dark:text-cyan-400 dark:hover:text-cyan-300">
                    Manage in Email alerts →
                </a>
            </div>
            <ul id="idxAgentUpcomingAlerts" class="space-y-2 text-sm text-slate-700 dark:text-slate-200">
                <li class="text-slate-500">Loading alerts...</li>
            </ul>
        </div>

        <div class="grid gap-4 lg:grid-cols-3">
            <div class="idx-panel lg:col-span-2">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">Recent activity</h4>
                <ul id="idxAgentActivityFeed" class="mt-3 space-y-2 text-sm text-slate-700 dark:text-slate-200">
                    <li class="text-slate-500">Loading activity...</li>
                </ul>
            </div>

            <div class="idx-panel">
                <div class="mb-3 flex gap-1 border-b border-slate-200 dark:border-slate-700">
                    <button type="button" data-contact-tab="recent" class="idx-contact-highlight-tab border-b-2 border-cyan-600 px-2 py-1 text-xs font-semibold text-cyan-700 dark:text-cyan-400">Recent</button>
                    <button type="button" data-contact-tab="hot" class="idx-contact-highlight-tab border-b-2 border-transparent px-2 py-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Hot</button>
                    <button type="button" data-contact-tab="new" class="idx-contact-highlight-tab border-b-2 border-transparent px-2 py-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">New</button>
                </div>
                <ul id="idxAgentContactHighlights" class="space-y-1 text-xs text-slate-600 dark:text-slate-300">
                    <li class="text-slate-500">Loading contacts...</li>
                </ul>
            </div>
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const refreshBtn = document.getElementById('idxAgentDashboardRefresh');
            const statusEl = document.getElementById('idxAgentDashboardStatus');
            const contactsEl = document.getElementById('idxAgentKpiContacts');
            const alertsEl = document.getElementById('idxAgentKpiAlerts');
            const searchesEl = document.getElementById('idxAgentKpiSearches');
            const shareLinksEl = document.getElementById('idxAgentKpiShareLinks');
            const feedEl = document.getElementById('idxAgentActivityFeed');
            const upcomingAlertsEl = document.getElementById('idxAgentUpcomingAlerts');
            const contactHighlightsEl = document.getElementById('idxAgentContactHighlights');
            const periodButtons = Array.from(document.querySelectorAll('.idx-period-btn'));
            let currentPeriod = 7;
            let currentContactTab = 'recent';

            const switchPeriod = (days) => {
                currentPeriod = days;
                periodButtons.forEach((btn) => {
                    const active = Number(btn.dataset.period) === days;
                    btn.classList.toggle('bg-cyan-600', active);
                    btn.classList.toggle('text-white', active);
                });
                loadDashboard();
            };

            periodButtons.forEach((btn) => {
                btn.addEventListener('click', () => switchPeriod(Number(btn.dataset.period)));
            });

            const switchContactTab = (tab) => {
                currentContactTab = tab;
                document.querySelectorAll('.idx-contact-highlight-tab').forEach((btn) => {
                    const active = btn.dataset.contactTab === tab;
                    btn.classList.toggle('border-cyan-600', active);
                    btn.classList.toggle('text-cyan-700', active);
                    btn.classList.toggle('dark:text-cyan-400', active);
                    btn.classList.toggle('font-semibold', active);
                    btn.classList.toggle('border-transparent', !active);
                    btn.classList.toggle('text-slate-500', !active);
                    btn.classList.toggle('font-medium', !active);
                });
                loadContactHighlights();
            };

            document.querySelectorAll('.idx-contact-highlight-tab').forEach((btn) => {
                btn.addEventListener('click', () => switchContactTab(btn.dataset.contactTab));
            });

            const renderUpcomingAlerts = (items) => {
                if (!Array.isArray(items) || items.length === 0) {
                    upcomingAlertsEl.innerHTML = '<li class="text-slate-500">No active alerts. Create one under Email alerts.</li>';
                    return;
                }
                upcomingAlertsEl.innerHTML = '';
                items.forEach((alert) => {
                    const li = document.createElement('li');
                    const when = alert.next_run_at ? new Date(alert.next_run_at).toLocaleString() : 'Schedule pending';
                    li.className = 'flex flex-wrap items-center justify-between gap-2 rounded-md border border-slate-200 px-3 py-2 dark:border-slate-700';
                    li.innerHTML = `<span class="font-medium text-slate-900 dark:text-slate-100">${alert.name ?? 'Alert'}</span><span class="text-xs text-slate-500 dark:text-slate-400">${alert.alert_type ?? ''} · ${alert.cadence ?? ''}</span><span class="text-xs text-slate-600 dark:text-slate-300">${when}</span>`;
                    upcomingAlertsEl.appendChild(li);
                });
            };

            const feedTypeLabel = (type) => {
                const labels = {
                    lead: 'Lead',
                    alert: 'Alert',
                    alert_run: 'Alert run',
                    search: 'Saved search',
                    share_link: 'Share link',
                };
                return labels[type] || type || 'Event';
            };

            const renderFeed = (items) => {
                if (!Array.isArray(items) || items.length === 0) {
                    feedEl.innerHTML = '<li class="text-slate-500">No recent activity.</li>';
                    return;
                }
                feedEl.innerHTML = '';
                items.slice(0, 15).forEach((item) => {
                    const li = document.createElement('li');
                    const when = item.at ? new Date(item.at).toLocaleString() : 'n/a';
                    const kind = feedTypeLabel(item.type);
                    const extra = item.alert_type ? ` · ${item.alert_type}` : '';
                    li.className = 'rounded-md border border-slate-200 px-3 py-2 dark:border-slate-700';
                    li.textContent = `${when} — ${kind}${extra} — ${item.title || 'Untitled'} (${item.status || 'n/a'})`;
                    feedEl.appendChild(li);
                });
            };

            const renderContactHighlights = (contacts) => {
                if (!Array.isArray(contacts) || contacts.length === 0) {
                    contactHighlightsEl.innerHTML = '<li class="text-slate-500">No contacts in this segment.</li>';
                    return;
                }
                contactHighlightsEl.innerHTML = '';
                contacts.slice(0, 8).forEach((contact) => {
                    const li = document.createElement('li');
                    li.className = 'flex items-center justify-between rounded-md border border-slate-200 px-2 py-1.5 dark:border-slate-700';
                    const status = contact.payload?.status || contact.status || 'new';
                    const name = contact.payload?.name || contact.name || `Contact #${contact.id}`;
                    li.innerHTML = `<span class="truncate font-medium">${name}</span><span class="ml-2 rounded-full bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-300">${status}</span>`;
                    contactHighlightsEl.appendChild(li);
                });
            };

            const loadDashboard = async () => {
                const response = await fetch(`/agent/dashboard/summary?period=${currentPeriod}`, { headers: { Accept: 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading dashboard summary');
                }
                const payload = await response.json();
                const kpis = payload?.data?.kpis || {};
                contactsEl.textContent = String(kpis.contacts || 0);
                alertsEl.textContent = String(kpis.active_alerts || 0);
                searchesEl.textContent = String(kpis.saved_searches || 0);
                shareLinksEl.textContent = String(kpis.active_share_links || 0);
                renderFeed(payload?.data?.activity_feed || []);
                renderUpcomingAlerts(payload?.data?.upcoming_alerts || []);
                statusEl.textContent = `Updated now (${currentPeriod}D window)`;
            };

            const loadContactHighlights = async () => {
                const response = await fetch(`/agent/contacts?tab=${currentContactTab}&per_page=8`, { headers: { Accept: 'application/json' } });
                if (!response.ok) {
                    return;
                }
                const payload = await response.json();
                renderContactHighlights(payload?.data?.items || []);
            };

            refreshBtn.addEventListener('click', () => {
                statusEl.textContent = 'Refreshing...';
                Promise.all([loadDashboard(), loadContactHighlights()]).catch((error) => {
                    statusEl.textContent = 'Refresh failed.';
                    console.error(error);
                });
            });

            switchPeriod(7);
            loadContactHighlights().catch(() => {});
        });
    </script>
</x-filament-panels::page>
