<x-filament-panels::page>
    <div class="idx-agent-shell space-y-6">
        <div class="idx-panel">
            <h3 class="idx-panel-title">Contacts</h3>
            <p class="idx-panel-subtitle">
                CRM-style grid with quick filters and scoped lead detail.
            </p>

            <div class="mt-4 rounded-lg border border-dashed border-slate-300 p-3 dark:border-slate-700">
                <p class="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">New contact + alert handoff</p>
                <div class="mt-2 grid gap-3 md:grid-cols-5">
                    <input id="idxNewContactName" type="text" class="idx-input" placeholder="Contact name" />
                    <input id="idxNewContactEmail" type="email" class="idx-input" placeholder="Email" />
                    <input id="idxNewContactPhone" type="text" class="idx-input" placeholder="Phone" />
                    <input id="idxNewContactAlertName" type="text" class="idx-input" placeholder="Alert name" />
                    <button id="idxNewContactHandoffBtn" type="button" class="idx-btn-primary">Create contact + alert</button>
                </div>
            </div>

            <div class="mt-4 grid gap-3 md:grid-cols-5">
                <input id="idxContactsSearch" type="text" class="idx-input" placeholder="Search name/email/phone" />
                <select id="idxContactsStatus" class="idx-input">
                    <option value="">All statuses</option>
                    <option value="new">New</option>
                    <option value="hot">Hot</option>
                    <option value="contacted">Contacted</option>
                    <option value="converted">Converted</option>
                </select>
                <select id="idxContactsActivity" class="idx-input">
                    <option value="">All activity</option>
                    <option value="recent">Recent (7d)</option>
                    <option value="stale">Stale (30d+)</option>
                </select>
                <select id="idxContactsTab" class="idx-input">
                    <option value="all">All leads</option>
                    <option value="nurtured">Nurtured</option>
                    <option value="awaiting">Awaiting</option>
                    <option value="archive">Archive</option>
                </select>
                <button id="idxContactsRefreshBtn" type="button" class="idx-btn-secondary">Refresh</button>
            </div>
            <p id="idxContactsStatusText" class="mt-3 text-xs text-slate-500 dark:text-slate-400"></p>

            <div class="mt-3 flex flex-wrap items-center gap-2">
                <label class="flex items-center gap-1 text-xs text-slate-600 dark:text-slate-300">
                    <input id="idxContactsSelectAll" type="checkbox" class="rounded border-slate-300 dark:border-slate-700" />
                    Select all
                </label>
                <button id="idxContactsBulkStatusBtn" type="button" class="rounded-md border border-cyan-300 px-2 py-1 text-xs font-medium text-cyan-700 dark:border-cyan-800 dark:text-cyan-300">Set status</button>
                <select id="idxContactsBulkStatusValue" class="idx-input py-0.5 text-xs">
                    <option value="contacted">Contacted</option>
                    <option value="hot">Hot</option>
                    <option value="converted">Converted</option>
                    <option value="archived">Archived</option>
                </select>
                <button id="idxContactsBulkDeleteBtn" type="button" class="rounded-md border border-rose-300 px-2 py-1 text-xs font-medium text-rose-700 dark:border-rose-800 dark:text-rose-300">Delete selected</button>
                <button id="idxContactsExportBtn" type="button" class="idx-btn-secondary text-xs">Export CSV</button>
                <span id="idxContactsBulkStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
            </div>
        </div>

        <div class="idx-panel">
            <div class="overflow-x-auto">
                <table class="idx-data-table w-full text-left text-sm">
                    <thead class="border-b border-slate-200 text-xs uppercase text-slate-500 dark:border-slate-700 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-1 font-medium w-8"></th>
                            <th class="py-2 pr-4 font-medium">Name</th>
                            <th class="py-2 pr-4 font-medium">Email</th>
                            <th class="py-2 pr-4 font-medium">Status</th>
                            <th class="py-2 pr-4 font-medium">Domain</th>
                            <th class="py-2 pr-4 font-medium">Created</th>
                            <th class="py-2 font-medium">Actions</th>
                        </tr>
                    </thead>
                    <tbody id="idxContactsRows">
                        <tr><td colspan="7" class="py-4 text-slate-500">Loading contacts...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>

        <div id="idxContactDetailPanel" class="idx-panel hidden">
            <div class="mb-4 flex items-center justify-between">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">
                    Contact detail — <span id="idxContactDetailName" class="text-slate-600 dark:text-slate-300"></span>
                </h4>
                <button id="idxContactCloseDetailBtn" type="button" class="idx-btn-secondary">Close</button>
            </div>

            <div class="mb-4 flex flex-wrap gap-1 border-b border-slate-200 dark:border-slate-700">
                <button type="button" data-contact-tab="overview" class="idx-contact-tab-btn border-b-2 border-cyan-600 px-3 py-2 text-xs font-semibold text-cyan-700 dark:text-cyan-400">Overview</button>
                <button type="button" data-contact-tab="alerts" class="idx-contact-tab-btn border-b-2 border-transparent px-3 py-2 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Alerts</button>
                <button type="button" data-contact-tab="email" class="idx-contact-tab-btn border-b-2 border-transparent px-3 py-2 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Email</button>
                <button type="button" data-contact-tab="site" class="idx-contact-tab-btn border-b-2 border-transparent px-3 py-2 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Site</button>
                <button type="button" data-contact-tab="timeline" class="idx-contact-tab-btn border-b-2 border-transparent px-3 py-2 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Timeline</button>
                <button type="button" data-contact-tab="details" class="idx-contact-tab-btn border-b-2 border-transparent px-3 py-2 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400">Details</button>
            </div>

            <div id="idxContactTabOverview">
                <div class="mb-4 grid gap-3 md:grid-cols-4">
                    <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">Status</p>
                        <p id="idxContactOverviewStatus" class="text-sm font-semibold text-slate-900 dark:text-slate-100">—</p>
                    </div>
                    <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">Stage</p>
                        <p id="idxContactOverviewStage" class="text-sm font-semibold text-slate-900 dark:text-slate-100">—</p>
                    </div>
                    <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">Email</p>
                        <p id="idxContactOverviewEmail" class="text-sm font-semibold text-slate-900 dark:text-slate-100">—</p>
                    </div>
                    <div class="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                        <p class="text-[11px] uppercase tracking-wide text-slate-500">Phone</p>
                        <p id="idxContactOverviewPhone" class="text-sm font-semibold text-slate-900 dark:text-slate-100">—</p>
                    </div>
                </div>
                <div class="grid gap-3 md:grid-cols-4">
                    <input id="idxContactUpdateStatus" type="text" class="idx-input" placeholder="status (contacted)" />
                    <input id="idxContactUpdateStage" type="text" class="idx-input" placeholder="stage (qualified)" />
                    <input id="idxContactUpdateTags" type="text" class="idx-input" placeholder="tags comma-separated" />
                    <button id="idxContactUpdateBtn" type="button" class="idx-btn-secondary">Save updates</button>
                </div>
                <textarea id="idxContactUpdateNotes" rows="2" class="mt-3 w-full idx-input" placeholder="Notes"></textarea>
            </div>

            <div id="idxContactTabAlerts" class="hidden">
                <div class="mb-3 grid gap-3 md:grid-cols-4">
                    <select id="idxContactHandoffSearchId" class="idx-input">
                        <option value="">Select saved search</option>
                    </select>
                    <input id="idxContactHandoffAlertName" type="text" class="idx-input" placeholder="Alert name" />
                    <select id="idxContactHandoffCadence" class="idx-input">
                        <option value="daily">Daily</option>
                        <option value="weekly">Weekly</option>
                        <option value="monthly">Monthly</option>
                    </select>
                    <button id="idxContactHandoffBtn" type="button" class="idx-btn-primary">Create alert from contact</button>
                </div>
                <p id="idxContactAlertsEmpty" class="text-xs text-slate-500 dark:text-slate-400">No alerts linked to this contact yet.</p>
                <ul id="idxContactAlertsList" class="space-y-1 text-xs text-slate-600 dark:text-slate-300"></ul>
            </div>

            <div id="idxContactTabEmail" class="hidden">
                <h5 class="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Email activity</h5>
                <ul id="idxContactEmailList" class="space-y-1 text-xs text-slate-600 dark:text-slate-300"></ul>
            </div>

            <div id="idxContactTabSite" class="hidden">
                <h5 class="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Site activity</h5>
                <ul id="idxContactSiteList" class="space-y-1 text-xs text-slate-600 dark:text-slate-300"></ul>
            </div>

            <div id="idxContactTabTimeline" class="hidden">
                <h5 class="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Full timeline</h5>
                <ul id="idxContactTimelineList" class="space-y-1 text-xs text-slate-600 dark:text-slate-300"></ul>
            </div>

            <div id="idxContactTabDetails" class="hidden">
                <pre id="idxContactDetailJson" class="overflow-x-auto rounded-md bg-slate-100 p-3 text-xs text-slate-800 dark:bg-slate-900 dark:text-slate-200"></pre>
            </div>
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const searchEl = document.getElementById('idxContactsSearch');
            const statusEl = document.getElementById('idxContactsStatus');
            const activityEl = document.getElementById('idxContactsActivity');
            const tabEl = document.getElementById('idxContactsTab');
            const refreshBtn = document.getElementById('idxContactsRefreshBtn');
            const rowsEl = document.getElementById('idxContactsRows');
            const statusTextEl = document.getElementById('idxContactsStatusText');
            const detailPanelEl = document.getElementById('idxContactDetailPanel');
            const detailJsonEl = document.getElementById('idxContactDetailJson');
            const detailNameEl = document.getElementById('idxContactDetailName');
            const overviewStatusEl = document.getElementById('idxContactOverviewStatus');
            const overviewStageEl = document.getElementById('idxContactOverviewStage');
            const overviewEmailEl = document.getElementById('idxContactOverviewEmail');
            const overviewPhoneEl = document.getElementById('idxContactOverviewPhone');
            const closeDetailBtn = document.getElementById('idxContactCloseDetailBtn');
            const alertsListEl = document.getElementById('idxContactAlertsList');
            const alertsEmptyEl = document.getElementById('idxContactAlertsEmpty');
            const updateStatusEl = document.getElementById('idxContactUpdateStatus');
            const updateStageEl = document.getElementById('idxContactUpdateStage');
            const updateTagsEl = document.getElementById('idxContactUpdateTags');
            const updateNotesEl = document.getElementById('idxContactUpdateNotes');
            const updateBtn = document.getElementById('idxContactUpdateBtn');
            const handoffSearchEl = document.getElementById('idxContactHandoffSearchId');
            const handoffAlertNameEl = document.getElementById('idxContactHandoffAlertName');
            const handoffCadenceEl = document.getElementById('idxContactHandoffCadence');
            const handoffBtn = document.getElementById('idxContactHandoffBtn');
            const emailListEl = document.getElementById('idxContactEmailList');
            const siteListEl = document.getElementById('idxContactSiteList');
            const timelineListEl = document.getElementById('idxContactTimelineList');
            const newContactNameEl = document.getElementById('idxNewContactName');
            const newContactEmailEl = document.getElementById('idxNewContactEmail');
            const newContactPhoneEl = document.getElementById('idxNewContactPhone');
            const newContactAlertNameEl = document.getElementById('idxNewContactAlertName');
            const newContactHandoffBtn = document.getElementById('idxNewContactHandoffBtn');
            let contacts = [];
            let selectedContactId = null;
            let contactActivityEvents = [];

            const formatActivityLine = (event) => {
                const type = event.type || 'event';
                const title = event.title ? String(event.title) : '';
                const at = event.at ? new Date(event.at).toLocaleString() : 'n/a';
                const channel = event.channel ? ` [${event.channel}]` : '';
                const summary = title !== '' ? title : type;
                return `${at} — ${summary}${channel}`;
            };

            const renderActivityPanels = () => {
                const renderList = (el, items, emptyText) => {
                    if (!el) {
                        return;
                    }
                    if (!items.length) {
                        el.innerHTML = `<li class="text-slate-500">${emptyText}</li>`;
                        return;
                    }
                    el.innerHTML = '';
                    items.forEach((event) => {
                        const li = document.createElement('li');
                        li.textContent = formatActivityLine(event);
                        el.appendChild(li);
                    });
                };

                const byChannel = (ch) => contactActivityEvents.filter((e) => (e.channel || 'other') === ch);
                renderList(emailListEl, byChannel('email'), 'No email activity recorded for this contact yet.');
                renderList(siteListEl, byChannel('site'), 'No site activity recorded for this contact yet.');
                renderList(timelineListEl, contactActivityEvents, 'No timeline events yet.');
            };

            const contactTabIds = ['overview', 'alerts', 'email', 'site', 'timeline', 'details'];

            const switchContactTab = (tabName) => {
                document.querySelectorAll('.idx-contact-tab-btn').forEach((btn) => {
                    const isActive = btn.dataset.contactTab === tabName;
                    btn.classList.toggle('border-cyan-600', isActive);
                    btn.classList.toggle('text-cyan-700', isActive);
                    btn.classList.toggle('dark:text-cyan-400', isActive);
                    btn.classList.toggle('font-semibold', isActive);
                    btn.classList.toggle('border-transparent', !isActive);
                    btn.classList.toggle('text-slate-500', !isActive);
                    btn.classList.toggle('font-medium', !isActive);
                });
                contactTabIds.forEach((t) => {
                    const cap = t.charAt(0).toUpperCase() + t.slice(1);
                    const panel = document.getElementById(`idxContactTab${cap}`);
                    if (panel) panel.classList.toggle('hidden', t !== tabName);
                });
                if (tabName === 'email' || tabName === 'site' || tabName === 'timeline') {
                    renderActivityPanels();
                }
            };

            document.querySelectorAll('.idx-contact-tab-btn').forEach((btn) => {
                btn.addEventListener('click', () => switchContactTab(btn.dataset.contactTab));
            });

            closeDetailBtn?.addEventListener('click', () => {
                detailPanelEl.classList.add('hidden');
                selectedContactId = null;
            });

            const loadSavedSearches = async () => {
                const response = await fetch('/agent/searches', { headers: { Accept: 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading searches');
                }
                const payload = await response.json();
                const searches = payload?.data || [];
                handoffSearchEl.innerHTML = '<option value="">Select saved search</option>';
                searches.forEach((search) => {
                    const option = document.createElement('option');
                    option.value = String(search.id);
                    option.textContent = search.name || `Search #${search.id}`;
                    handoffSearchEl.appendChild(option);
                });
            };

            const renderRows = () => {
                if (contacts.length === 0) {
                    rowsEl.innerHTML = '<tr><td colspan="7" class="py-4 text-slate-500">No contacts found.</td></tr>';
                    return;
                }
                rowsEl.innerHTML = '';
                contacts.forEach((contact) => {
                    const payload = contact.payload || {};
                    const name = payload.name || `${payload.first_name || ''} ${payload.last_name || ''}`.trim() || `Lead #${contact.id}`;
                    const email = payload.email || '—';
                    const leadStatus = payload.status || 'new';
                    const created = contact.created_at ? new Date(contact.created_at).toLocaleString() : 'n/a';
                    const tr = document.createElement('tr');
                    tr.className = 'border-b border-slate-100 dark:border-slate-800';
                    tr.innerHTML = `
                        <td class="py-2 pr-1"><input type="checkbox" data-contact-checkbox="${contact.id}" class="rounded border-slate-300 dark:border-slate-700" /></td>
                        <td class="py-2 pr-4">${name}</td>
                        <td class="py-2 pr-4">${email}</td>
                        <td class="py-2 pr-4">${leadStatus}</td>
                        <td class="py-2 pr-4">${contact.quantyra_domain || '—'}</td>
                        <td class="py-2 pr-4">${created}</td>
                        <td class="py-2">
                            <button type="button" data-contact-id="${contact.id}" class="rounded-md border border-cyan-300 px-2 py-1 text-xs font-medium text-cyan-700 dark:border-cyan-800 dark:text-cyan-300">Open</button>
                        </td>
                    `;
                    rowsEl.appendChild(tr);
                });
            };

            const loadContacts = async () => {
                const params = new URLSearchParams();
                if (searchEl.value.trim() !== '') params.set('search', searchEl.value.trim());
                if (statusEl.value !== '') params.set('status', statusEl.value);
                if (activityEl.value !== '') params.set('activity', activityEl.value);
                if (tabEl.value !== '') params.set('tab', tabEl.value);
                const query = params.toString();
                const response = await fetch(query === '' ? '/agent/contacts' : `/agent/contacts?${query}`, {
                    headers: { Accept: 'application/json' },
                });
                if (!response.ok) {
                    throw new Error('Failed loading contacts');
                }
                const payload = await response.json();
                contacts = payload?.data?.items || [];
                const total = payload?.data?.meta?.total || 0;
                statusTextEl.textContent = `${total} contact(s) in scope`;
                renderRows();
            };

            const openContact = async (contactId) => {
                const response = await fetch(`/agent/contacts/${contactId}`, { headers: { Accept: 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading contact');
                }
                const payload = await response.json();
                selectedContactId = Number(payload?.data?.id || 0) || null;
                const dataPayload = payload?.data?.payload || {};

                if (detailNameEl) detailNameEl.textContent = dataPayload.name || `Contact #${contactId}`;
                if (overviewStatusEl) overviewStatusEl.textContent = dataPayload.status || '—';
                if (overviewStageEl) overviewStageEl.textContent = dataPayload.stage || '—';
                if (overviewEmailEl) overviewEmailEl.textContent = dataPayload.email || '—';
                if (overviewPhoneEl) overviewPhoneEl.textContent = dataPayload.phone || '—';

                updateStatusEl.value = dataPayload.status || '';
                updateStageEl.value = dataPayload.stage || '';
                updateTagsEl.value = Array.isArray(dataPayload.tags) ? dataPayload.tags.join(', ') : '';
                updateNotesEl.value = dataPayload.notes || '';
                detailJsonEl.textContent = JSON.stringify(payload?.data || {}, null, 2);
                detailPanelEl.classList.remove('hidden');
                switchContactTab('overview');
                await loadActivity();
            };

            const loadActivity = async () => {
                if (!selectedContactId) {
                    return;
                }
                const response = await fetch(`/agent/contacts/${selectedContactId}/activity`, { headers: { Accept: 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading activity');
                }
                const payload = await response.json();
                contactActivityEvents = Array.isArray(payload?.data) ? payload.data : [];
                renderActivityPanels();
            };

            const updateContact = async () => {
                if (!selectedContactId) {
                    return;
                }
                const tags = updateTagsEl.value
                    .split(',')
                    .map((v) => v.trim())
                    .filter((v) => v !== '');
                const body = {
                    status: updateStatusEl.value.trim(),
                    stage: updateStageEl.value.trim(),
                    tags,
                    notes: updateNotesEl.value,
                };
                const response = await fetch(`/agent/contacts/${selectedContactId}`, {
                    method: 'PUT',
                    headers: {
                        Accept: 'application/json',
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]')?.content || '',
                    },
                    body: JSON.stringify(body),
                });
                if (!response.ok) {
                    throw new Error('Failed saving contact');
                }
                const payload = await response.json();
                detailJsonEl.textContent = JSON.stringify(payload?.data || {}, null, 2);
                statusTextEl.textContent = 'Contact updated.';
                await loadContacts();
                await loadActivity();
            };

            const handoffContactToAlert = async () => {
                if (!selectedContactId) {
                    return;
                }
                const searchId = Number(handoffSearchEl.value || 0);
                if (!searchId) {
                    throw new Error('Select a saved search first');
                }
                const body = {
                    agent_search_id: searchId,
                    name: handoffAlertNameEl.value.trim() || `Contact ${selectedContactId} Alert`,
                    alert_type: 'listing',
                    cadence: handoffCadenceEl.value || 'daily',
                };
                const response = await fetch(`/agent/contacts/${selectedContactId}/handoff/alert`, {
                    method: 'POST',
                    headers: {
                        Accept: 'application/json',
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]')?.content || '',
                    },
                    body: JSON.stringify(body),
                });
                if (!response.ok) {
                    throw new Error('Failed to create alert from contact');
                }
                statusTextEl.textContent = 'Alert created from contact handoff.';
                await openContact(selectedContactId);
            };

            const createContactAndHandoff = async () => {
                const searchId = Number(handoffSearchEl.value || 0);
                if (!searchId) {
                    throw new Error('Select a saved search first');
                }
                const contactName = newContactNameEl.value.trim();
                if (contactName === '') {
                    throw new Error('Contact name is required');
                }
                const body = {
                    contact: {
                        name: contactName,
                        email: newContactEmailEl.value.trim() || null,
                        phone: newContactPhoneEl.value.trim() || null,
                    },
                    agent_search_id: searchId,
                    name: newContactAlertNameEl.value.trim() || `${contactName} Alert`,
                    alert_type: 'listing',
                    cadence: handoffCadenceEl.value || 'daily',
                };
                const response = await fetch('/agent/contacts/handoff/alert', {
                    method: 'POST',
                    headers: {
                        Accept: 'application/json',
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]')?.content || '',
                    },
                    body: JSON.stringify(body),
                });
                if (!response.ok) {
                    throw new Error('Failed creating contact and alert');
                }
                const payload = await response.json();
                statusTextEl.textContent = 'Created contact and alert handoff.';
                newContactNameEl.value = '';
                newContactEmailEl.value = '';
                newContactPhoneEl.value = '';
                newContactAlertNameEl.value = '';
                const createdContactId = Number(payload?.data?.contact?.id || 0);
                await loadContacts();
                if (createdContactId) {
                    await openContact(createdContactId);
                }
            };

            refreshBtn.addEventListener('click', () => {
                statusTextEl.textContent = 'Refreshing contacts...';
                loadContacts().catch((error) => {
                    statusTextEl.textContent = 'Contacts refresh failed.';
                    console.error(error);
                });
            });

            [searchEl, statusEl, activityEl, tabEl].forEach((el) => {
                el.addEventListener('change', () => {
                    loadContacts().catch((error) => {
                        statusTextEl.textContent = 'Contacts refresh failed.';
                        console.error(error);
                    });
                });
            });

            rowsEl.addEventListener('click', (event) => {
                const button = event.target.closest('button[data-contact-id]');
                if (!(button instanceof HTMLElement)) {
                    return;
                }
                const contactId = Number(button.dataset.contactId || 0);
                if (!contactId) {
                    return;
                }
                openContact(contactId).catch((error) => {
                    statusTextEl.textContent = 'Failed to load contact detail.';
                    console.error(error);
                });
            });

            updateBtn.addEventListener('click', () => {
                updateContact().catch((error) => {
                    statusTextEl.textContent = 'Failed to save contact.';
                    console.error(error);
                });
            });

            handoffBtn.addEventListener('click', () => {
                handoffContactToAlert().catch((error) => {
                    statusTextEl.textContent = 'Failed to create alert handoff.';
                    console.error(error);
                });
            });

            newContactHandoffBtn.addEventListener('click', () => {
                createContactAndHandoff().catch((error) => {
                    statusTextEl.textContent = 'Failed creating contact + alert.';
                    console.error(error);
                });
            });

            loadContacts().catch((error) => {
                statusTextEl.textContent = 'Failed to load contacts.';
                console.error(error);
            });
            loadSavedSearches().catch((error) => {
                console.error(error);
            });

            const selectAllEl = document.getElementById('idxContactsSelectAll');
            const bulkStatusBtn = document.getElementById('idxContactsBulkStatusBtn');
            const bulkStatusValue = document.getElementById('idxContactsBulkStatusValue');
            const bulkDeleteBtn = document.getElementById('idxContactsBulkDeleteBtn');
            const exportBtn = document.getElementById('idxContactsExportBtn');
            const bulkStatusEl = document.getElementById('idxContactsBulkStatus');

            const getSelectedIds = () => {
                return Array.from(rowsEl.querySelectorAll('input[data-contact-checkbox]:checked'))
                    .map((input) => Number(input.dataset.contactCheckbox))
                    .filter((id) => id > 0);
            };

            selectAllEl?.addEventListener('change', () => {
                const checked = selectAllEl.checked;
                rowsEl.querySelectorAll('input[data-contact-checkbox]').forEach((cb) => {
                    cb.checked = checked;
                });
            });

            bulkStatusBtn?.addEventListener('click', async () => {
                const ids = getSelectedIds();
                if (ids.length === 0) {
                    bulkStatusEl.textContent = 'No contacts selected.';
                    return;
                }
                bulkStatusEl.textContent = `Updating ${ids.length} contact(s)...`;
                const newStatus = bulkStatusValue?.value || 'contacted';
                try {
                    const response = await fetch('/agent/contacts/bulk/status', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({ contact_ids: ids, status: newStatus }),
                    });
                    if (!response.ok) {
                        bulkStatusEl.textContent = 'Bulk update failed.';
                        return;
                    }
                    bulkStatusEl.textContent = `Updated ${ids.length} contact(s) to ${newStatus}.`;
                    await loadContacts();
                } catch {
                    bulkStatusEl.textContent = 'Bulk update failed.';
                }
            });

            bulkDeleteBtn?.addEventListener('click', async () => {
                const ids = getSelectedIds();
                if (ids.length === 0) {
                    bulkStatusEl.textContent = 'No contacts selected.';
                    return;
                }
                if (!confirm(`Delete ${ids.length} contact(s)? This cannot be undone.`)) {
                    return;
                }
                bulkStatusEl.textContent = `Deleting ${ids.length} contact(s)...`;
                try {
                    const response = await fetch('/agent/contacts/bulk/delete', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({ contact_ids: ids }),
                    });
                    if (!response.ok) {
                        bulkStatusEl.textContent = 'Bulk delete failed.';
                        return;
                    }
                    bulkStatusEl.textContent = `Deleted ${ids.length} contact(s).`;
                    await loadContacts();
                } catch {
                    bulkStatusEl.textContent = 'Bulk delete failed.';
                }
            });

            exportBtn?.addEventListener('click', () => {
                window.location.href = '/agent/contacts/export.csv';
            });
        });
    </script>
</x-filament-panels::page>
