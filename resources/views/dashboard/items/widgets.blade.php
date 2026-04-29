@if ($activePanel === 'widgets')
    <section id="widget-library" class="idx-card mt-6 p-5 shadow-xl shadow-cyan-950/20 sm:p-6">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
                <h2 class="text-2xl font-semibold tracking-tight text-white">Widget Library</h2>
                <p class="mt-1 text-sm text-slate-300">Install search and listing widgets in minutes with one-click embed code copy.</p>
            </div>
            <p class="text-xs text-cyan-200">Loader URL: <span class="font-mono">{{ $widgetLoaderBaseUrl }}/widget/loader.js</span></p>
        </div>
        @if ($hasWidgetAccess && $widgetPreviewApiKey !== '')
            <div class="mt-4 rounded-xl border border-cyan-400/25 bg-slate-950/60 p-4">
                <p class="text-sm font-semibold text-cyan-100">Your widget site key</p>
                <p class="mt-1 text-xs text-slate-400">Put this in <span class="font-mono text-slate-200">?token=</span> on your site. Allowed origins are the hostnames under <span class="font-semibold text-slate-300">My Approved Domains</span>. GoHighLevel marketplace is optional.</p>
                <div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
                    <code class="flex-1 break-all rounded-lg border border-white/15 bg-slate-900 px-3 py-2 font-mono text-xs text-slate-100" x-text="previewApiKey"></code>
                    <button type="button" @click="navigator.clipboard.writeText(previewApiKey); toast = 'Site key copied'; setTimeout(() => toast = '', 2200)" class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg border border-cyan-400/40 px-3 py-2 text-xs font-semibold text-cyan-100 hover:bg-cyan-500/10">Copy key</button>
                </div>
            </div>
        @endif
        <div class="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            @foreach ($widgetCards as $widget)
                <article class="group rounded-2xl border border-white/10 bg-slate-950/70 p-4 transition-all hover:-translate-y-0.5 hover:border-cyan-400/40 hover:shadow-lg hover:shadow-cyan-900/20">
                    <div class="rounded-xl border border-white/10 bg-gradient-to-br from-slate-900 to-slate-800 p-3 text-xs text-slate-300">
                        <p class="font-semibold text-slate-100">{{ $widget['preview'] }}</p>
                        <p class="mt-1">Use Preview Demo for a live proxy-backed render.</p>
                    </div>
                    <h3 class="mt-3 text-base font-semibold text-white">{{ $widget['label'] }}</h3>
                    <p class="mt-1 text-sm text-slate-300">{{ $widget['description'] }}</p>
                    <div class="mt-4 flex flex-wrap gap-2">
                        <button type="button" @click="copyEmbed('{{ $widget['slug'] }}')" class="inline-flex min-h-10 items-center rounded-lg bg-cyan-500 px-3 py-2 text-xs font-semibold text-slate-950 hover:bg-cyan-400">Copy Embed Code</button>
                        <button type="button" @click="openPreview('{{ $widget['slug'] }}')" class="inline-flex min-h-10 items-center rounded-lg border border-white/20 px-3 py-2 text-xs font-semibold text-slate-200 hover:bg-white/10">Preview Demo</button>
                    </div>
                </article>
            @endforeach
        </div>
    </section>

    @if ($hasWidgetAccess)
        <section class="idx-card mt-8 rounded-3xl p-6 shadow-xl sm:p-8">
            <h2 class="text-xl font-semibold tracking-tight text-white">Widget appearance</h2>
            <p class="mt-1 text-sm text-slate-300">Set colors once for every embed (search, map, property, footer). Per-page query parameters on the loader still override these when you need a one-off.</p>
            <form method="POST" action="{{ route('dashboard.widget-appearance', [], false) }}" class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                @csrf
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Primary
                    <input type="color" name="primary" value="{{ old('primary', $widgetPaletteForm['primary'] ?? '#2563eb') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Secondary
                    <input type="color" name="secondary" value="{{ old('secondary', $widgetPaletteForm['secondary'] ?? '#1e40af') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Accent (optional)
                    <input type="color" name="accent" value="{{ old('accent', $widgetPaletteForm['accent'] ?? '#10b981') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Text
                    <input type="color" name="text" value="{{ old('text', $widgetPaletteForm['text'] ?? '#0f172a') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Background
                    <input type="color" name="background" value="{{ old('background', $widgetPaletteForm['background'] ?? '#ffffff') }}" class="mt-2 h-10 w-full cursor-pointer rounded-lg border border-white/20 bg-slate-950" />
                </label>
                <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Theme
                    <select name="theme" class="mt-2 w-full rounded-lg border border-white/20 bg-slate-950 px-3 py-2 text-sm text-slate-100">
                        <option value="light" @selected(old('theme', $widgetPaletteForm['theme'] ?? 'light') === 'light')>Light</option>
                        <option value="dark" @selected(old('theme', $widgetPaletteForm['theme'] ?? 'light') === 'dark')>Dark</option>
                    </select>
                </label>
                <div class="sm:col-span-2 lg:col-span-3">
                    <button type="submit" class="inline-flex min-h-11 items-center rounded-lg bg-violet-500 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-400">Save widget colors</button>
                </div>
            </form>
            @if ($errors->any())
                <div class="mt-4 rounded-lg border border-rose-400/30 bg-rose-900/20 px-3 py-2 text-xs text-rose-100">{{ $errors->first() }}</div>
            @endif
        </section>
    @endif
@endif
