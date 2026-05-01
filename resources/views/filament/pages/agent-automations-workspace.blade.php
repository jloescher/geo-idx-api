<x-filament-panels::page>
    <div class="idx-agent-shell space-y-6">
        <div class="idx-panel">
            <h3 class="idx-panel-title">Automations</h3>
            <p class="idx-panel-subtitle">
                Auto nurture settings and integration health controls.
            </p>

            <div class="mt-4 grid gap-4 lg:grid-cols-3">
                <label class="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200">
                    <input id="idxAutoNurtureEnabled" type="checkbox" class="rounded border-slate-300 dark:border-slate-700" />
                    Enable nurture
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Nurture mode
                    <select id="idxAutoNurtureMode" class="idx-input">
                        <option value="off">Off</option>
                        <option value="basic">Basic</option>
                        <option value="aggressive">Aggressive</option>
                    </select>
                </label>
                <label class="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200">
                    <input id="idxAutoDryRun" type="checkbox" class="rounded border-slate-300 dark:border-slate-700" />
                    Dry run only
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300 lg:col-span-2">
                    Eligibility tags (comma-separated)
                    <input id="idxAutoTags" type="text" class="idx-input" placeholder="buyer,hot_lead" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Integration health
                    <select id="idxAutoHealth" class="idx-input">
                        <option value="connected">Connected</option>
                        <option value="degraded">Degraded</option>
                        <option value="disconnected">Disconnected</option>
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300 lg:col-span-3">
                    Notes
                    <textarea id="idxAutoNotes" rows="3" class="idx-input" placeholder="Operational notes..."></textarea>
                </label>
            </div>

            <div class="mt-3 flex flex-wrap items-center gap-2">
                <button id="idxAutoSaveBtn" type="button" class="idx-btn-primary">
                    Save settings
                </button>
                <button id="idxAutoReloadBtn" type="button" class="idx-btn-secondary">
                    Reload
                </button>
                <button id="idxAutoConnectBtn" type="button" class="idx-btn-secondary">
                    Connect GHL
                </button>
                <button id="idxAutoReconnectBtn" type="button" class="idx-btn-secondary">
                    Reconnect GHL
                </button>
                <button id="idxAutoDisconnectBtn" type="button" class="idx-btn-secondary">
                    Disconnect GHL
                </button>
                <span id="idxAutoStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
            </div>
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const enabledEl = document.getElementById('idxAutoNurtureEnabled');
            const modeEl = document.getElementById('idxAutoNurtureMode');
            const dryRunEl = document.getElementById('idxAutoDryRun');
            const tagsEl = document.getElementById('idxAutoTags');
            const healthEl = document.getElementById('idxAutoHealth');
            const notesEl = document.getElementById('idxAutoNotes');
            const saveBtn = document.getElementById('idxAutoSaveBtn');
            const reloadBtn = document.getElementById('idxAutoReloadBtn');
            const connectBtn = document.getElementById('idxAutoConnectBtn');
            const reconnectBtn = document.getElementById('idxAutoReconnectBtn');
            const disconnectBtn = document.getElementById('idxAutoDisconnectBtn');
            const statusEl = document.getElementById('idxAutoStatus');

            const hydrate = (data) => {
                enabledEl.checked = Boolean(data?.nurture_enabled);
                modeEl.value = data?.nurture_mode ?? 'off';
                dryRunEl.checked = Boolean(data?.dry_run);
                tagsEl.value = Array.isArray(data?.eligibility_tags) ? data.eligibility_tags.join(',') : '';
                healthEl.value = data?.integration_health ?? 'disconnected';
                notesEl.value = data?.notes ?? '';
            };

            const load = async () => {
                const response = await fetch('/agent/automations/settings', { headers: { 'Accept': 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading automation settings');
                }
                const payload = await response.json();
                hydrate(payload?.data || {});
            };

            saveBtn.addEventListener('click', async () => {
                statusEl.textContent = 'Saving...';
                const payload = {
                    nurture_enabled: enabledEl.checked,
                    nurture_mode: modeEl.value,
                    dry_run: dryRunEl.checked,
                    eligibility_tags: tagsEl.value
                        .split(',')
                        .map((tag) => tag.trim())
                        .filter((tag) => tag !== ''),
                    integration_health: healthEl.value,
                    notes: notesEl.value.trim() || null,
                };

                try {
                    const response = await fetch('/agent/automations/settings', {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify(payload),
                    });
                    if (!response.ok) {
                        statusEl.textContent = 'Save failed.';
                        return;
                    }
                    statusEl.textContent = 'Settings saved.';
                } catch (error) {
                    statusEl.textContent = 'Save failed.';
                    console.error(error);
                }
            });

            reloadBtn.addEventListener('click', () => {
                statusEl.textContent = 'Loading...';
                load()
                    .then(() => {
                        statusEl.textContent = 'Settings loaded.';
                    })
                    .catch((error) => {
                        statusEl.textContent = 'Load failed.';
                        console.error(error);
                    });
            });

            const runIntegrationAction = async (action) => {
                statusEl.textContent = `${action}...`;
                const response = await fetch(`/agent/automations/settings/integrations/${action}`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': '{{ csrf_token() }}',
                        'Accept': 'application/json',
                    },
                    body: JSON.stringify({ provider: 'ghl' }),
                });
                if (!response.ok) {
                    statusEl.textContent = `${action} failed.`;
                    return;
                }
                await load();
                statusEl.textContent = `Integration ${action} complete.`;
            };

            connectBtn.addEventListener('click', () => {
                runIntegrationAction('connect').catch((error) => {
                    statusEl.textContent = 'Connect failed.';
                    console.error(error);
                });
            });
            reconnectBtn.addEventListener('click', () => {
                runIntegrationAction('reconnect').catch((error) => {
                    statusEl.textContent = 'Reconnect failed.';
                    console.error(error);
                });
            });
            disconnectBtn.addEventListener('click', () => {
                runIntegrationAction('disconnect').catch((error) => {
                    statusEl.textContent = 'Disconnect failed.';
                    console.error(error);
                });
            });

            load()
                .then(() => {
                    statusEl.textContent = 'Settings loaded.';
                })
                .catch((error) => {
                    statusEl.textContent = 'Load failed.';
                    console.error(error);
                });
        });
    </script>
</x-filament-panels::page>
