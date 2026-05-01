<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ $seoTitle }}</title>
    <meta name="description" content="{{ $seoDescription }}" />
    <meta name="robots" content="{{ $robotsDirective }}" />
    <link rel="canonical" href="{{ $canonicalUrl }}" />
    <meta property="og:title" content="{{ $seoTitle }}" />
    <meta property="og:description" content="{{ $seoDescription }}" />
    <meta property="og:url" content="{{ $canonicalUrl }}" />
    @vite(['resources/css/app.css'])
</head>
<body class="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
    <main class="mx-auto max-w-4xl px-4 py-8">
        <div class="rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <p class="text-xs font-semibold uppercase tracking-wide text-cyan-700 dark:text-cyan-300">Shared search</p>
            <h1 class="mt-2 text-2xl font-semibold">{{ $search?->name ?? 'General market search' }}</h1>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">
                Explore listings from this shared search and run a live snapshot.
            </p>

            <div class="mt-5 grid gap-4 md:grid-cols-2">
                <div class="rounded-lg border border-slate-200 p-4 text-sm dark:border-slate-800">
                    <p class="font-medium">Search filters</p>
                    <ul class="mt-2 space-y-1 text-slate-600 dark:text-slate-300">
                        @forelse ($search?->filters ?? [] as $filter)
                            <li>
                                <span class="font-mono text-xs">{{ $filter->canonical_field_key }}</span>
                                {{ $filter->operator }}
                                <span class="font-medium">{{ is_scalar($filter->value_json) ? (string) $filter->value_json : json_encode($filter->value_json) }}</span>
                            </li>
                        @empty
                            <li>No saved filters.</li>
                        @endforelse
                    </ul>
                </div>

                <div class="rounded-lg border border-slate-200 p-4 text-sm dark:border-slate-800">
                    <p class="font-medium">Link details</p>
                    <p class="mt-2 text-slate-600 dark:text-slate-300"><span class="font-medium">Token:</span> <span class="font-mono text-xs">{{ $shareLink->token }}</span></p>
                    <p class="text-slate-600 dark:text-slate-300"><span class="font-medium">Geometries:</span> {{ $search?->geometries?->count() ?? 0 }}</p>
                </div>
            </div>

            <div class="mt-5">
                <button id="runSharedSearch" type="button" class="rounded-md bg-cyan-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-cyan-500">
                    Run live results
                </button>
                <p id="sharedSearchStatus" class="mt-2 text-xs text-slate-500 dark:text-slate-400"></p>
            </div>

            <div id="sharedResults" class="mt-4 space-y-2"></div>
        </div>
    </main>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const runBtn = document.getElementById('runSharedSearch');
            const statusEl = document.getElementById('sharedSearchStatus');
            const resultsEl = document.getElementById('sharedResults');

            runBtn?.addEventListener('click', async () => {
                statusEl.textContent = 'Running search...';
                resultsEl.innerHTML = '';

                try {
                    const response = await fetch('/shared/{{ $shareLink->token }}/execute', {
                        method: 'POST',
                        headers: {
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                    });
                    const payload = await response.json();
                    if (!response.ok) {
                        statusEl.textContent = 'Search failed.';
                        return;
                    }

                    const items = payload?.data?.items || [];
                    statusEl.textContent = `${items.length} listing(s) returned.`;
                    items.slice(0, 20).forEach((item) => {
                        const row = document.createElement('div');
                        row.className = 'rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm dark:border-slate-800 dark:bg-slate-900/60';
                        row.innerHTML = `
                            <div class="font-semibold">${item.fullAddress || item.listingId || 'Listing'}</div>
                            <div class="text-slate-600 dark:text-slate-300">
                                ${(item.listPrice ? '$' + Number(item.listPrice).toLocaleString() : 'N/A')}
                                · ${item.bedroomsTotal ?? '-'} bd · ${item.bathroomsTotal ?? '-'} ba
                            </div>
                        `;
                        resultsEl.appendChild(row);
                    });
                } catch (error) {
                    statusEl.textContent = 'Search failed.';
                    console.error(error);
                }
            });
        });
    </script>
</body>
</html>
