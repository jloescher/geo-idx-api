<x-filament-panels::page>
    <div class="idx-agent-shell space-y-6">
        @if (! empty($onboardingChecklist) && ($showOnboardingChecklist ?? true))
            <div class="rounded-xl border border-slate-200/80 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900/40" data-agent-onboarding-checklist>
                <h3 class="text-base font-semibold text-slate-900 dark:text-slate-100">Getting started</h3>
                <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">
                    Data-driven checklist: steps check off automatically as you use the portal. Links respect modules disabled in Agent Settings.
                </p>
                <ol class="mt-4 space-y-3">
                    @foreach ($onboardingChecklist as $step)
                        <li class="rounded-lg border border-slate-200/80 px-3 py-2 dark:border-slate-700 {{ ($step['done'] ?? false) ? 'bg-emerald-50/60 dark:bg-emerald-950/20' : 'bg-slate-50 dark:bg-slate-950/40' }}">
                            <div class="flex flex-wrap items-start justify-between gap-2">
                                <div>
                                    <span class="text-sm font-semibold text-slate-900 dark:text-slate-100">
                                        @if (! empty($step['done']))
                                            <span class="text-emerald-600 dark:text-emerald-400" aria-hidden="true">✓</span>
                                        @else
                                            <span class="text-slate-400 dark:text-slate-500" aria-hidden="true">○</span>
                                        @endif
                                        {{ $step['title'] }}
                                    </span>
                                    <p class="mt-1 text-xs text-slate-600 dark:text-slate-400">{{ $step['description'] }}</p>
                                    @if (! empty($step['blocked']))
                                        <p class="mt-1 text-xs font-medium text-amber-700 dark:text-amber-300">This step’s module is off — open Agent Settings to enable it.</p>
                                    @endif
                                </div>
                                @if (empty($step['done']) && ! empty($step['href']))
                                    <a
                                        href="{{ $step['href'] }}"
                                        class="shrink-0 rounded-md border border-cyan-500/40 bg-cyan-50 px-2 py-1 text-xs font-semibold text-cyan-800 hover:bg-cyan-100 dark:border-cyan-700 dark:bg-cyan-950/60 dark:text-cyan-200 dark:hover:bg-cyan-900/80"
                                        @if (\Illuminate\Support\Str::contains((string) $step['href'], '/filament-dashboard'))
                                            wire:navigate
                                        @endif
                                    >
                                        {{ $step['cta_label'] ?? 'Open' }}
                                    </a>
                                @endif
                            </div>
                        </li>
                    @endforeach
                </ol>
                <div class="mt-4 flex flex-wrap items-center gap-2 border-t border-slate-200 pt-4 dark:border-slate-700">
                    <button
                        id="idxDismissAgentOnboardingChecklist"
                        type="button"
                        class="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                    >
                        Dismiss checklist
                    </button>
                    <span id="idxDismissOnboardingStatus" class="text-xs text-slate-500 dark:text-slate-400"></span>
                </div>
            </div>
            <script>
                document.addEventListener('DOMContentLoaded', function () {
                    const btn = document.getElementById('idxDismissAgentOnboardingChecklist');
                    const statusEl = document.getElementById('idxDismissOnboardingStatus');
                    const panel = document.querySelector('[data-agent-onboarding-checklist]');
                    btn?.addEventListener('click', async () => {
                        if (!panel) return;
                        statusEl.textContent = 'Saving…';
                        try {
                            const response = await fetch('{{ url('/agent/settings/onboarding-checklist/dismiss') }}', {
                                method: 'POST',
                                headers: {
                                    Accept: 'application/json',
                                    'X-CSRF-TOKEN': '{{ csrf_token() }}',
                                },
                            });
                            if (!response.ok) {
                                statusEl.textContent = 'Could not dismiss.';
                                return;
                            }
                            panel.remove();
                        } catch {
                            statusEl.textContent = 'Could not dismiss.';
                        }
                    });
                });
            </script>
        @endif

        @if (! empty($workspaceShortcuts))
            <div class="rounded-xl border border-slate-200/80 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
                <h3 class="text-base font-semibold text-slate-900 dark:text-slate-100">Workspace shortcuts</h3>
                <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">
                    Jump to Agent Portal modules. Items reflect your feature flags; disabled shortcuts stay visible so you know what’s available after enabling modules in Agent Settings.
                </p>
                <ul class="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    @foreach ($workspaceShortcuts as $link)
                        <li>
                            @if ($link['enabled'])
                                <a
                                    href="{{ $link['url'] }}"
                                    data-agent-workspace-shortcut="{{ $link['slug'] }}"
                                    class="flex h-full flex-col rounded-lg border border-slate-200/80 bg-slate-50 px-4 py-3 text-sm shadow-sm transition hover:border-cyan-500/40 hover:bg-white dark:border-slate-700 dark:bg-slate-950/50 dark:hover:bg-slate-900/60"
                                    wire:navigate
                                >
                                    <span class="font-medium text-slate-900 dark:text-slate-100">{{ $link['label'] }}</span>
                                    @if (! empty($link['badges']))
                                        <span class="mt-2 flex flex-wrap gap-1.5">
                                            @foreach ($link['badges'] as $badge)
                                                @if ($badge['enabled'])
                                                    <span data-agent-subfeature-badge="{{ $badge['key'] }}" class="rounded border border-emerald-400/70 bg-emerald-50 px-2 py-0.5 font-mono text-[10px] font-medium uppercase tracking-wide text-emerald-800 dark:bg-emerald-950/70 dark:text-emerald-200">{{ $badge['label'] }}</span>
                                                @else
                                                    <span data-agent-subfeature-badge="{{ $badge['key'] }}" class="rounded border border-dashed border-slate-300 bg-transparent px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide text-slate-500 dark:border-slate-600 dark:text-slate-400">{{ $badge['label'] }}</span>
                                                @endif
                                            @endforeach
                                        </span>
                                    @endif
                                    <span class="mt-auto pt-2 text-xs text-emerald-600 dark:text-emerald-400">Open</span>
                                </a>
                            @else
                                <span
                                    data-agent-workspace-shortcut="{{ $link['slug'] }}"
                                    class="flex h-full flex-col rounded-lg border border-dashed border-slate-300/80 bg-slate-50/60 px-4 py-3 text-sm opacity-85 dark:border-slate-600 dark:bg-slate-950/30"
                                    title="Enable this module in Agent Settings → feature flags."
                                >
                                    <span class="font-medium text-slate-700 dark:text-slate-300">{{ $link['label'] }}</span>
                                    @if (! empty($link['badges']))
                                        <span class="mt-2 flex flex-wrap gap-1.5 opacity-95">
                                            @foreach ($link['badges'] as $badge)
                                                @if ($badge['enabled'])
                                                    <span data-agent-subfeature-badge="{{ $badge['key'] }}" class="rounded border border-emerald-400/70 bg-emerald-50 px-2 py-0.5 font-mono text-[10px] font-medium uppercase tracking-wide text-emerald-800 dark:bg-emerald-950/70 dark:text-emerald-200">{{ $badge['label'] }}</span>
                                                @else
                                                    <span data-agent-subfeature-badge="{{ $badge['key'] }}" class="rounded border border-dashed border-slate-400/80 px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide text-slate-600 dark:border-slate-500 dark:text-slate-400">{{ $badge['label'] }}</span>
                                                @endif
                                            @endforeach
                                        </span>
                                    @endif
                                    <span class="mt-auto pt-2 text-xs font-medium uppercase tracking-wide text-amber-600 dark:text-amber-400">Off</span>
                                </span>
                            @endif
                        </li>
                    @endforeach
                </ul>
            </div>
        @endif

        <div class="rounded-xl border border-slate-200/80 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
            <h3 class="text-base font-semibold text-slate-900 dark:text-slate-100">MLS &amp; dataset access</h3>
            <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">
                Resolved from your account assignments and active domains. Used to constrain search, alerts, and widgets.
            </p>
            <div class="mt-4 overflow-x-auto">
                <table class="w-full text-left text-sm">
                    <thead class="border-b border-slate-200 text-xs uppercase text-slate-500 dark:border-slate-700 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-4 font-medium">MLS code</th>
                            <th class="py-2 pr-4 font-medium">Dataset</th>
                            <th class="py-2 pr-4 font-medium">Feed</th>
                            <th class="py-2 font-medium">Status</th>
                        </tr>
                    </thead>
                    <tbody>
                        @forelse ($feedScopes as $row)
                            <tr class="border-b border-slate-100 dark:border-slate-800">
                                <td class="py-2 pr-4 font-mono text-xs">{{ $row['mls_code'] }}</td>
                                <td class="py-2 pr-4 font-mono text-xs">{{ $row['dataset_code'] }}</td>
                                <td class="py-2 pr-4 font-mono text-xs">{{ $row['feed_id'] }}</td>
                                <td class="py-2 text-emerald-600 dark:text-emerald-400">{{ $row['status'] }}</td>
                            </tr>
                        @empty
                            <tr>
                                <td colspan="4" class="py-4 text-slate-500">No feed rows yet. Assign MLS datasets on your account or activate a domain with a dataset.</td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>

        <div class="rounded-xl border border-slate-200/80 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
            <h3 class="text-base font-semibold text-slate-900 dark:text-slate-100">Core search fields (registry sample)</h3>
            <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">
                Canonical keys shared across MLS adapters; enum values load from lookups cache when wired.
            </p>
            <ul class="mt-4 grid gap-2 sm:grid-cols-2">
                @foreach ($registrySample as $field)
                    <li class="rounded-lg border border-slate-200/80 bg-slate-50 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-950/50">
                        <span class="font-mono text-xs text-cyan-700 dark:text-cyan-300">{{ $field['key'] }}</span>
                        <span class="mt-1 block text-slate-700 dark:text-slate-200">{{ $field['label'] }}</span>
                        <span class="text-xs text-slate-500">{{ $field['category'] }} · {{ $field['type'] }}</span>
                    </li>
                @endforeach
            </ul>
        </div>
    </div>
</x-filament-panels::page>
