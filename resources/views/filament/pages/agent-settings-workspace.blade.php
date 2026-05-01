<x-filament-panels::page>
    <div class="idx-agent-shell space-y-6">
        <div class="idx-panel">
            <h3 class="idx-panel-title">Settings</h3>
            <p class="idx-panel-subtitle">Notification, cadence, UX, and feature preferences.</p>

            <div class="mt-4 grid gap-4 lg:grid-cols-3">
                <label class="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200">
                    <input id="idxSettingsEmail" type="checkbox" class="rounded border-slate-300 dark:border-slate-700" />
                    Email notifications enabled
                </label>
                <label class="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200">
                    <input id="idxSettingsSms" type="checkbox" class="rounded border-slate-300 dark:border-slate-700" />
                    SMS notifications enabled
                </label>
                <label class="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200">
                    <input id="idxSettingsWeeklyDigest" type="checkbox" class="rounded border-slate-300 dark:border-slate-700" />
                    Weekly digest enabled
                </label>

                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Default alert cadence
                    <select id="idxSettingsCadence" class="idx-input">
                        <option value="daily">Daily</option>
                        <option value="weekly">Weekly</option>
                        <option value="monthly">Monthly</option>
                    </select>
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Timezone
                    <input id="idxSettingsTimezone" type="text" class="idx-input" placeholder="America/New_York" />
                </label>
                <label class="block text-xs text-slate-600 dark:text-slate-300">
                    Theme density
                    <select id="idxSettingsDensity" class="idx-input">
                        <option value="compact">Compact</option>
                        <option value="comfortable">Comfortable</option>
                    </select>
                </label>
                <label class="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200 lg:col-span-2">
                    <input id="idxSettingsTips" type="checkbox" class="rounded border-slate-300 dark:border-slate-700" />
                    Show onboarding tips inside agent workspace
                </label>
            </div>
            <div class="mt-4 flex flex-wrap items-center gap-2">
                <button id="idxSettingsSaveBtn" type="button" class="idx-btn-primary">Save settings</button>
                <button id="idxSettingsReloadBtn" type="button" class="idx-btn-secondary">Reload</button>
                <span id="idxSettingsStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
            </div>
        </div>

        @if ($onboardingChecklistDismissed ?? false)
            <div id="idxOnboardingChecklistSettings" class="idx-panel">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">Getting started checklist</h4>
                <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    The overview &ldquo;Getting started&rdquo; card is hidden. You can show it again from here.
                </p>
                <div class="mt-3 flex flex-wrap items-center gap-2">
                    <button id="idxOnboardingChecklistRestoreBtn" type="button" class="idx-btn-secondary">
                        Show checklist on overview
                    </button>
                    <span id="idxOnboardingChecklistSettingsStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
                </div>
            </div>
        @endif

        <div class="idx-panel">
            <h4 class="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">Feature flags</h4>
            <p class="mb-3 text-xs text-slate-500 dark:text-slate-400">Toggle modules on or off for your portal. Disabled modules are hidden from navigation.</p>
            <div id="idxFeatureFlagsToggles" class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                <p class="text-xs text-slate-500">Loading flags...</p>
            </div>
            <div class="mt-3 flex items-center gap-2">
                <button id="idxFlagsSaveBtn" type="button" class="idx-btn-primary">Save flags</button>
                <span id="idxFlagsStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
            </div>
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const emailEl = document.getElementById('idxSettingsEmail');
            const smsEl = document.getElementById('idxSettingsSms');
            const weeklyDigestEl = document.getElementById('idxSettingsWeeklyDigest');
            const cadenceEl = document.getElementById('idxSettingsCadence');
            const timezoneEl = document.getElementById('idxSettingsTimezone');
            const densityEl = document.getElementById('idxSettingsDensity');
            const tipsEl = document.getElementById('idxSettingsTips');
            const saveBtn = document.getElementById('idxSettingsSaveBtn');
            const reloadBtn = document.getElementById('idxSettingsReloadBtn');
            const statusEl = document.getElementById('idxSettingsStatus');

            const onboardingChecklistPanel = document.getElementById('idxOnboardingChecklistSettings');
            const onboardingRestoreBtn = document.getElementById('idxOnboardingChecklistRestoreBtn');
            const onboardingChecklistStatus = document.getElementById('idxOnboardingChecklistSettingsStatus');

            const hydrate = (data) => {
                emailEl.checked = Boolean(data?.notification_email_enabled);
                smsEl.checked = Boolean(data?.notification_sms_enabled);
                weeklyDigestEl.checked = Boolean(data?.weekly_digest_enabled);
                cadenceEl.value = data?.alert_default_cadence || 'daily';
                timezoneEl.value = data?.timezone || 'America/New_York';
                densityEl.value = data?.theme_density || 'compact';
                tipsEl.checked = Boolean(data?.onboarding_tips_enabled);
                const checklistHidden = Boolean(data?.hide_agent_onboarding_checklist);
                if (onboardingChecklistPanel) {
                    onboardingChecklistPanel.classList.toggle('hidden', !checklistHidden);
                }
            };

            const load = async () => {
                const response = await fetch('/agent/settings', { headers: { Accept: 'application/json' } });
                if (!response.ok) {
                    throw new Error('Failed loading settings');
                }
                const payload = await response.json();
                hydrate(payload?.data || {});
            };

            saveBtn.addEventListener('click', async () => {
                statusEl.textContent = 'Saving...';
                const body = {
                    notification_email_enabled: emailEl.checked,
                    notification_sms_enabled: smsEl.checked,
                    weekly_digest_enabled: weeklyDigestEl.checked,
                    alert_default_cadence: cadenceEl.value,
                    timezone: timezoneEl.value.trim() || 'America/New_York',
                    theme_density: densityEl.value,
                    onboarding_tips_enabled: tipsEl.checked,
                };

                try {
                    const response = await fetch('/agent/settings', {
                        method: 'PUT',
                        headers: {
                            Accept: 'application/json',
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                        },
                        body: JSON.stringify(body),
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
                statusEl.textContent = 'Reloading...';
                load().then(() => {
                    statusEl.textContent = 'Settings loaded.';
                }).catch((error) => {
                    statusEl.textContent = 'Load failed.';
                    console.error(error);
                });
            });

            onboardingRestoreBtn?.addEventListener('click', async () => {
                onboardingChecklistStatus.textContent = 'Updating...';
                try {
                    const response = await fetch('{{ url('/agent/settings/onboarding-checklist/restore') }}', {
                        method: 'POST',
                        headers: {
                            Accept: 'application/json',
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                        },
                        body: JSON.stringify({}),
                    });
                    if (!response.ok) {
                        onboardingChecklistStatus.textContent = 'Request failed.';
                        return;
                    }
                    await load();
                    onboardingChecklistStatus.textContent = 'Checklist will show on Overview.';
                } catch (error) {
                    onboardingChecklistStatus.textContent = 'Request failed.';
                    console.error(error);
                }
            });

            load().then(() => {
                statusEl.textContent = 'Settings loaded.';
            }).catch((error) => {
                statusEl.textContent = 'Load failed.';
                console.error(error);
            });

            const flagsTogglesEl = document.getElementById('idxFeatureFlagsToggles');
            const flagsSaveBtn = document.getElementById('idxFlagsSaveBtn');
            const flagsStatusEl = document.getElementById('idxFlagsStatus');
            let currentFlags = {};

            const loadFlags = async () => {
                const response = await fetch('/agent/settings/feature-flags', { headers: { Accept: 'application/json' } });
                if (!response.ok) return;
                const payload = await response.json();
                currentFlags = payload?.data?.flags || {};
                const defaults = payload?.data?.global_defaults || {};
                flagsTogglesEl.innerHTML = '';
                Object.entries(defaults).forEach(([module, defaultVal]) => {
                    const enabled = Boolean(currentFlags[module] ?? defaultVal);
                    const label = document.createElement('label');
                    label.className = 'flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:text-slate-200';
                    label.innerHTML = `<input type="checkbox" data-flag="${module}" ${enabled ? 'checked' : ''} class="rounded border-slate-300 dark:border-slate-700" /> ${module.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}`;
                    flagsTogglesEl.appendChild(label);
                });
            };

            flagsSaveBtn?.addEventListener('click', async () => {
                flagsStatusEl.textContent = 'Saving flags...';
                const flags = {};
                flagsTogglesEl.querySelectorAll('input[data-flag]').forEach((input) => {
                    flags[input.dataset.flag] = input.checked;
                });
                try {
                    const response = await fetch('/agent/settings/feature-flags', {
                        method: 'PUT',
                        headers: {
                            Accept: 'application/json',
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                        },
                        body: JSON.stringify({ flags }),
                    });
                    if (!response.ok) {
                        flagsStatusEl.textContent = 'Save failed.';
                        return;
                    }
                    flagsStatusEl.textContent = 'Feature flags saved.';
                } catch {
                    flagsStatusEl.textContent = 'Save failed.';
                }
            });

            loadFlags().catch(() => {});
        });
    </script>
</x-filament-panels::page>
