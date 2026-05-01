<x-filament-panels::page>
    <div class="idx-agent-shell space-y-6">
        <div class="idx-panel">
            <h3 class="idx-panel-title">Map + filters</h3>
            <p class="idx-panel-subtitle">
                Draw include/exclude rectangles, polygons, or circles; geocoding goes through the API with a proper Nominatim User-Agent; optional debounced search-as-you-pan.
            </p>

            <div class="mt-4 grid gap-6 lg:grid-cols-3">
                <div>
                    <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">Canonical fields</p>
                    <ul class="space-y-2 text-sm">
                        @foreach ($coreFields as $field)
                            <li class="rounded-md border border-slate-200/80 px-2 py-1 dark:border-slate-700">
                                <span class="font-mono text-xs text-cyan-700 dark:text-cyan-300">{{ $field['key'] }}</span>
                                — {{ $field['label'] }}
                            </li>
                        @endforeach
                    </ul>

                    <div class="mt-4 space-y-3">
                        <p class="text-sm font-medium text-slate-700 dark:text-slate-200">Quick filters</p>
                        <label class="block text-xs text-slate-600 dark:text-slate-300">
                            Min price
                            <input id="idxFilterMinPrice" type="number" min="0" class="idx-input" placeholder="250000" />
                        </label>
                        <label class="block text-xs text-slate-600 dark:text-slate-300">
                            Min beds
                            <input id="idxFilterMinBeds" type="number" min="0" class="idx-input" placeholder="3" />
                        </label>
                        <label class="block text-xs text-slate-600 dark:text-slate-300">
                            City
                            <input id="idxFilterCity" type="text" class="idx-input" placeholder="Tampa" />
                        </label>
                    </div>

                    <div class="mt-4 space-y-2">
                        <p class="text-sm font-medium text-slate-700 dark:text-slate-200">Filter categories</p>
                        <div id="idxCategoryAccordion" class="space-y-1">
                            <p class="text-xs text-slate-500">Loading categories...</p>
                        </div>
                    </div>

                    <div class="mt-4 space-y-2">
                        <p class="text-sm font-medium text-slate-700 dark:text-slate-200">Additional filters</p>
                        <div class="grid gap-2">
                            <select id="idxAdditionalFieldKey" class="idx-input">
                                <option value="">Select field...</option>
                            </select>
                            <select id="idxAdditionalFieldOperator" class="idx-input">
                                <option value="eq">Equals</option>
                                <option value="contains">Contains</option>
                                <option value="gte">Greater than or equal</option>
                                <option value="lte">Less than or equal</option>
                            </select>
                            <input id="idxAdditionalFieldValue" type="text" class="idx-input" placeholder="Filter value" list="idxAdditionalFieldSuggestions" />
                            <datalist id="idxAdditionalFieldSuggestions"></datalist>
                            <button id="idxAddAdditionalFilterBtn" type="button" class="idx-btn-secondary">Add filter</button>
                        </div>
                        <div id="idxAdditionalFilterRows" class="space-y-1 text-xs text-slate-600 dark:text-slate-300">
                            <p>No additional filters selected.</p>
                        </div>
                    </div>
                </div>

                <div class="space-y-3">
                    <div class="flex flex-wrap items-center gap-2">
                        <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">Draw mode</span>
                        <button id="idxModeInclude" type="button" class="rounded-md bg-emerald-600 px-2 py-1 text-xs font-medium text-white">Include</button>
                        <button id="idxModeExclude" type="button" class="rounded-md bg-slate-600 px-2 py-1 text-xs font-medium text-white">Exclude</button>
                        <button id="idxClearShapes" type="button" class="idx-btn-secondary">Clear shapes</button>
                    </div>
                    <div class="grid gap-2">
                        <label class="block text-xs text-slate-600 dark:text-slate-300">
                            Saved searches
                            <select id="idxSavedSearchSelect" class="idx-input">
                                <option value="">Select a saved search...</option>
                            </select>
                        </label>
                        <div class="flex flex-wrap gap-2">
                            <input id="idxSaveSearchName" type="text" class="idx-input flex-1" placeholder="Search name (e.g. South Tampa Waterfront)" />
                            <button id="idxSaveSearchBtn" type="button" class="idx-btn-primary">
                                Save
                            </button>
                            <button id="idxLoadSearchBtn" type="button" class="idx-btn-secondary">
                                Load
                            </button>
                        </div>
                        <div class="flex flex-wrap gap-2">
                            <button id="idxCreateAlertBtn" type="button" class="rounded-md bg-amber-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-amber-500">
                                Create alert from selected
                            </button>
                        </div>
                    </div>
                    <div wire:ignore class="idx-agent-map" id="idxAgentSearchMap"></div>
                    <div class="mt-2 flex flex-wrap items-center gap-3">
                        <label class="inline-flex cursor-pointer items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                            <input id="idxMapAutoFit" type="checkbox" class="rounded border-slate-300 dark:border-slate-600" checked />
                            <span>Auto-fit map to results</span>
                        </label>
                        <button id="idxMapMyLocation" type="button" class="idx-btn-secondary text-xs">My location</button>
                        <button id="idxMapZoomShapes" type="button" class="idx-btn-secondary text-xs">Zoom to drawn shapes</button>
                        <button id="idxMapFullscreen" type="button" class="idx-btn-secondary text-xs">Fullscreen map</button>
                        <label class="inline-flex cursor-pointer items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                            <input id="idxSearchOnPan" type="checkbox" class="rounded border-slate-300 dark:border-slate-600" />
                            <span>Search as you pan (debounced)</span>
                        </label>
                    </div>
                    <div class="mt-1 flex items-center gap-2">
                        <input id="idxGeocodeInput" type="text" class="idx-input flex-1" placeholder="Search location (address, city, zip)" />
                        <button id="idxGeocodeBtn" type="button" class="idx-btn-secondary">Go</button>
                        <button id="idxSearchThisArea" type="button" class="idx-btn-secondary inline-flex hidden">
                            Search this area
                        </button>
                        <button id="idxClearMarkers" type="button" class="idx-btn-secondary inline-flex hidden">
                            Clear map pins
                        </button>
                    </div>
                    <p class="mt-1 text-[10px] leading-snug text-slate-400 dark:text-slate-500">
                        Geocoding ©
                        <a href="https://www.openstreetmap.org/copyright" class="text-cyan-700 underline dark:text-cyan-400" target="_blank" rel="noopener noreferrer">OpenStreetMap</a>
                        contributors, via
                        <a href="https://nominatim.org/" class="text-cyan-700 underline dark:text-cyan-400" target="_blank" rel="noopener noreferrer">Nominatim</a>.
                    </p>
                    <button id="idxRunSearch" type="button" class="idx-btn-primary inline-flex">
                        Run search
                    </button>
                    <button id="idxShareSearchBtn" type="button" class="idx-btn-secondary inline-flex">
                        Share search
                    </button>
                    <div id="idxShareUrlBox" class="hidden mt-1">
                        <input id="idxShareUrlInput" type="text" readonly class="idx-input w-full text-xs" />
                        <button id="idxCopyShareUrl" type="button" class="idx-btn-secondary mt-1 text-xs">Copy URL</button>
                    </div>
                    <p id="idxSearchStatus" class="text-xs text-slate-500 dark:text-slate-400"></p>
                </div>

                <div>
                    <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">Results</p>
                    <div id="idxSearchSummary" class="mb-2 text-xs text-slate-500 dark:text-slate-400">No search executed yet.</div>
                    <div id="idxSearchResults" class="space-y-2"></div>
                </div>
            </div>
        </div>
    </div>

    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" integrity="sha256-p4NxAoJBhIIN+hmNHrzRCf9tD/miZyoHS5obTRR9BMY=" crossorigin="" />
    <link rel="stylesheet" href="https://unpkg.com/leaflet-draw@1.0.4/dist/leaflet.draw.css" />
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js" integrity="sha256-20nQCchB9co0qIjJZRGuk2/Z9VM+kNiyxNV1lvTlZBo=" crossorigin=""></script>
    <script src="https://unpkg.com/leaflet-draw@1.0.4/dist/leaflet.draw.js"></script>
    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const el = document.getElementById('idxAgentSearchMap');
            if (!el || typeof L === 'undefined' || typeof L.Control.Draw === 'undefined') {
                return;
            }

            const map = L.map(el).setView([27.95, -82.45], 9);
            let suppressPanSearch = 0;
            function bumpSuppressPanSearch() {
                suppressPanSearch += 1;
                window.setTimeout(() => {
                    suppressPanSearch -= 1;
                }, 150);
            }
            const streetLayer = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                maxZoom: 19,
                attribution: '&copy; OpenStreetMap',
            });
            const satelliteLayer = L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}', {
                maxZoom: 19,
                attribution: '&copy; Esri',
            });
            streetLayer.addTo(map);

            const baseLayers = { Street: streetLayer, Satellite: satelliteLayer };
            L.control.layers(baseLayers, null, { position: 'topright' }).addTo(map);

            const drawnItems = new L.FeatureGroup().addTo(map);
            const markersLayer = new L.FeatureGroup().addTo(map);
            let drawMode = 'include';

            const mapAutoFitEl = document.getElementById('idxMapAutoFit');
            const mapMyLocationBtn = document.getElementById('idxMapMyLocation');
            const mapZoomShapesBtn = document.getElementById('idxMapZoomShapes');
            const mapFullscreenBtn = document.getElementById('idxMapFullscreen');

            const searchThisAreaBtn = document.getElementById('idxSearchThisArea');
            const clearMarkersBtn = document.getElementById('idxClearMarkers');
            const geocodeInput = document.getElementById('idxGeocodeInput');
            const geocodeBtn = document.getElementById('idxGeocodeBtn');
            let lastSearchFilters = null;

            clearMarkersBtn?.addEventListener('click', () => {
                markersLayer.clearLayers();
                clearMarkersBtn.classList.add('hidden');
            });

            const drawControl = new L.Control.Draw({
                draw: {
                    rectangle: { shapeOptions: { color: '#059669' } },
                    marker: false,
                    polyline: false,
                    circlemarker: false,
                    polygon: { shapeOptions: { color: '#059669' } },
                    circle: { shapeOptions: { color: '#059669' } },
                },
                edit: {
                    featureGroup: drawnItems,
                    remove: true,
                },
            });
            map.addControl(drawControl);

            const recordMapShapeTelemetry = (layerKind, mode) => {
                fetch('/agent/dashboard/events', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        Accept: 'application/json',
                        'X-CSRF-TOKEN': '{{ csrf_token() }}',
                    },
                    body: JSON.stringify({
                        event_type: 'map_shape_drawn',
                        title: `${mode === 'exclude' ? 'Exclude' : 'Include'} ${layerKind} on search map`,
                        metadata: { draw_mode: mode, geometry: layerKind },
                    }),
                }).catch(() => {});
            };

            map.on(L.Draw.Event.CREATED, (event) => {
                const layer = event.layer;
                layer.__idxMode = drawMode;
                if (drawMode === 'exclude') {
                    if (layer.setStyle) {
                        layer.setStyle({ color: '#dc2626', fillColor: '#fecaca', fillOpacity: 0.12 });
                    }
                } else if (layer.setStyle) {
                    layer.setStyle({ color: '#059669', fillColor: '#6ee7b7', fillOpacity: 0.12 });
                }
                drawnItems.addLayer(layer);
                let kind = 'shape';
                if (layer instanceof L.Polygon) {
                    kind = 'polygon';
                } else if (layer instanceof L.Circle) {
                    kind = 'circle';
                } else if (layer instanceof L.Rectangle) {
                    kind = 'rectangle';
                }
                recordMapShapeTelemetry(kind, drawMode);
            });

            const includeBtn = document.getElementById('idxModeInclude');
            const excludeBtn = document.getElementById('idxModeExclude');
            const clearBtn = document.getElementById('idxClearShapes');
            const runBtn = document.getElementById('idxRunSearch');
            const statusEl = document.getElementById('idxSearchStatus');
            const summaryEl = document.getElementById('idxSearchSummary');
            const resultsEl = document.getElementById('idxSearchResults');
            const savedSelectEl = document.getElementById('idxSavedSearchSelect');
            const saveNameEl = document.getElementById('idxSaveSearchName');
            const saveBtnEl = document.getElementById('idxSaveSearchBtn');
            const loadBtnEl = document.getElementById('idxLoadSearchBtn');
            const createAlertBtnEl = document.getElementById('idxCreateAlertBtn');
            const shareSearchBtnEl = document.getElementById('idxShareSearchBtn');
            const shareUrlBoxEl = document.getElementById('idxShareUrlBox');
            const shareUrlInputEl = document.getElementById('idxShareUrlInput');
            const copyShareUrlBtnEl = document.getElementById('idxCopyShareUrl');
            const minPriceEl = document.getElementById('idxFilterMinPrice');
            const minBedsEl = document.getElementById('idxFilterMinBeds');
            const cityEl = document.getElementById('idxFilterCity');
            const additionalFieldKeyEl = document.getElementById('idxAdditionalFieldKey');
            const additionalFieldOperatorEl = document.getElementById('idxAdditionalFieldOperator');
            const additionalFieldValueEl = document.getElementById('idxAdditionalFieldValue');
            const additionalFieldSuggestionsEl = document.getElementById('idxAdditionalFieldSuggestions');
            const addAdditionalFilterBtnEl = document.getElementById('idxAddAdditionalFilterBtn');
            const additionalFilterRowsEl = document.getElementById('idxAdditionalFilterRows');
            let savedSearches = [];
            let additionalFilters = [];
            let fieldCatalogCache = [];

            const setMode = (mode) => {
                drawMode = mode;
                includeBtn.classList.toggle('bg-emerald-600', mode === 'include');
                includeBtn.classList.toggle('bg-slate-600', mode !== 'include');
                excludeBtn.classList.toggle('bg-rose-600', mode === 'exclude');
                excludeBtn.classList.toggle('bg-slate-600', mode !== 'exclude');
            };
            setMode('include');

            includeBtn.addEventListener('click', () => setMode('include'));
            excludeBtn.addEventListener('click', () => setMode('exclude'));
            clearBtn.addEventListener('click', () => {
                drawnItems.clearLayers();
                statusEl.textContent = 'All shapes cleared.';
            });

            const boundsFromDrawnShapes = () => {
                const b = L.latLngBounds([]);
                let has = false;
                drawnItems.eachLayer((layer) => {
                    if (layer.getBounds && typeof layer.getBounds === 'function') {
                        const lb = layer.getBounds();
                        if (lb.isValid()) {
                            b.extend(lb);
                            has = true;
                        }
                    }
                });
                return has && b.isValid() ? b : null;
            };

            mapZoomShapesBtn?.addEventListener('click', () => {
                const b = boundsFromDrawnShapes();
                if (b) {
                    bumpSuppressPanSearch();
                    map.fitBounds(b, { padding: [28, 28], maxZoom: 15 });
                    statusEl.textContent = 'Map zoomed to drawn shapes.';
                } else {
                    statusEl.textContent = 'Draw a shape on the map first.';
                }
            });

            mapMyLocationBtn?.addEventListener('click', () => {
                if (!navigator.geolocation) {
                    statusEl.textContent = 'Location is not available in this browser.';
                    return;
                }
                statusEl.textContent = 'Locating…';
                navigator.geolocation.getCurrentPosition(
                    (pos) => {
                        const lat = pos.coords.latitude;
                        const lng = pos.coords.longitude;
                        bumpSuppressPanSearch();
                        map.setView([lat, lng], 14);
                        statusEl.textContent = 'Map centered on your location.';
                    },
                    () => {
                        statusEl.textContent = 'Could not read your location (permission or unavailable).';
                    },
                    { enableHighAccuracy: true, timeout: 12000 },
                );
            });

            mapFullscreenBtn?.addEventListener('click', () => {
                const target = el;
                if (!document.fullscreenElement) {
                    target.requestFullscreen?.().catch(() => {
                        statusEl.textContent = 'Fullscreen could not be entered.';
                    });
                } else {
                    document.exitFullscreen?.();
                }
            });

            document.addEventListener('fullscreenchange', () => {
                setTimeout(() => map.invalidateSize(), 200);
            });

            const serializeLayers = () => {
                const out = [];
                const pushLayer = (layer, mode) => {
                    if (layer instanceof L.Circle) {
                        const center = layer.getLatLng();
                        out.push({
                            geometry_type: 'circle',
                            mode,
                            geojson: {
                                center: { lat: center.lat, lng: center.lng },
                                radius_m: layer.getRadius(),
                            },
                        });
                        return;
                    }
                    if (typeof L.Rectangle !== 'undefined' && layer instanceof L.Rectangle) {
                        const b = layer.getBounds();
                        const sw = b.getSouthWest();
                        const ne = b.getNorthEast();
                        const coords = [
                            [sw.lng, sw.lat],
                            [ne.lng, sw.lat],
                            [ne.lng, ne.lat],
                            [sw.lng, ne.lat],
                            [sw.lng, sw.lat],
                        ];
                        out.push({
                            geometry_type: 'polygon',
                            mode,
                            geojson: { coordinates: [coords] },
                        });
                        return;
                    }
                    if (layer instanceof L.Polygon) {
                        const latlngs = layer.getLatLngs();
                        const ring = Array.isArray(latlngs[0]) ? latlngs[0] : latlngs;
                        const coords = ring.map((point) => [point.lng, point.lat]);
                        if (coords.length > 0) {
                            const first = coords[0];
                            const last = coords[coords.length - 1];
                            if (first[0] !== last[0] || first[1] !== last[1]) {
                                coords.push([first[0], first[1]]);
                            }
                        }
                        out.push({
                            geometry_type: 'polygon',
                            mode,
                            geojson: { coordinates: [coords] },
                        });
                    }
                };
                drawnItems.eachLayer((layer) => {
                    const mode = layer.__idxMode === 'exclude' ? 'exclude' : 'include';
                    pushLayer(layer, mode);
                });
                return out;
            };

            const clearAndRenderGeometries = (geometries) => {
                drawnItems.clearLayers();
                (geometries || []).forEach((geometry) => {
                    if (!geometry || !geometry.geometry_type || !geometry.mode) {
                        return;
                    }
                    let layer = null;
                    if (geometry.geometry_type === 'circle') {
                        const center = geometry.geojson?.center;
                        const radius = geometry.geojson?.radius_m;
                        if (center && Number.isFinite(center.lat) && Number.isFinite(center.lng) && Number.isFinite(radius)) {
                            layer = L.circle([center.lat, center.lng], { radius });
                        }
                    } else if (geometry.geometry_type === 'polygon') {
                        const ring = geometry.geojson?.coordinates?.[0];
                        if (Array.isArray(ring) && ring.length >= 3) {
                            const latlngs = ring.map((coord) => [coord[1], coord[0]]);
                            layer = L.polygon(latlngs);
                        }
                    }
                    if (!layer) {
                        return;
                    }
                    layer.__idxMode = geometry.mode;
                    if (geometry.mode === 'exclude') {
                        if (layer.setStyle) {
                            layer.setStyle({ color: '#dc2626', fillColor: '#fecaca', fillOpacity: 0.12 });
                        }
                    } else if (layer.setStyle) {
                        layer.setStyle({ color: '#059669', fillColor: '#6ee7b7', fillOpacity: 0.12 });
                    }
                    drawnItems.addLayer(layer);
                });
            };

            const currentFilters = () => {
                const filters = [];
                if (minPriceEl.value) {
                    filters.push({ field: 'property.list_price', operator: 'gte', value: Number(minPriceEl.value) });
                }
                if (minBedsEl.value) {
                    filters.push({ field: 'property.bedrooms_total', operator: 'gte', value: Number(minBedsEl.value) });
                }
                if (cityEl.value.trim() !== '') {
                    filters.push({ field: 'location.city', operator: 'eq', value: cityEl.value.trim() });
                }

                additionalFilters.forEach((filter) => {
                    if (!filter || !filter.field) {
                        return;
                    }
                    const inputType = inferInputType(filter.field);
                    let coercedValue = filter.value ?? null;
                    if (coercedValue !== null && coercedValue !== '') {
                        if (inputType === 'number') {
                            const num = Number(coercedValue);
                            if (!isNaN(num)) {
                                coercedValue = num;
                            }
                        }
                    }
                    filters.push({
                        field: filter.field,
                        operator: filter.operator || 'eq',
                        value: coercedValue,
                    });
                });
                return filters;
            };

            const renderAdditionalFilters = () => {
                if (!Array.isArray(additionalFilters) || additionalFilters.length === 0) {
                    additionalFilterRowsEl.innerHTML = '<p>No additional filters selected.</p>';
                    return;
                }
                additionalFilterRowsEl.innerHTML = '';
                additionalFilters.forEach((filter, index) => {
                    const row = document.createElement('div');
                    row.className = 'flex items-center justify-between rounded-md border border-slate-200/80 px-2 py-1 dark:border-slate-700';
                    row.innerHTML = `
                        <span>${filter.field} ${filter.operator} ${String(filter.value ?? '')}</span>
                        <button type="button" data-filter-index="${index}" class="rounded border border-rose-300 px-1 py-0 text-[11px] text-rose-700 dark:border-rose-800 dark:text-rose-300">Remove</button>
                    `;
                    additionalFilterRowsEl.appendChild(row);
                });
            };

            const mapFiltersToPersisted = (filters) => (filters || []).map((filter) => ({
                canonical_field_key: filter.field,
                operator: filter.operator,
                value_json: filter.value ?? null,
            }));

            const mapPersistedToExecuteFilters = (filters) => (filters || []).map((filter) => ({
                field: filter.canonical_field_key,
                operator: filter.operator,
                value: filter.value_json ?? null,
            }));

            const refreshSavedSearches = async () => {
                const response = await fetch('/agent/searches', {
                    headers: { 'Accept': 'application/json' },
                });
                if (!response.ok) {
                    throw new Error('Failed loading saved searches');
                }
                const payload = await response.json();
                savedSearches = payload?.data || [];
                savedSelectEl.innerHTML = '<option value="">Select a saved search...</option>';
                savedSearches.forEach((search) => {
                    const option = document.createElement('option');
                    option.value = String(search.id);
                    option.textContent = search.name || `Search #${search.id}`;
                    savedSelectEl.appendChild(option);
                });
            };

            const loadFieldCatalog = async () => {
                const response = await fetch('/agent/searches/lookups/options', {
                    headers: { Accept: 'application/json' },
                });
                if (!response.ok) {
                    throw new Error('Failed loading field catalog');
                }
                const payload = await response.json();
                const scopedRows = Array.isArray(payload?.data) ? payload.data : [];
                fieldCatalogCache = scopedRows;
                const fields = [];
                scopedRows.forEach((scopeRow) => {
                    const values = Array.isArray(scopeRow?.values) ? scopeRow.values : [];
                    values.forEach((entry) => {
                        const key = String(entry?.LookupName || '');
                        if (key === '') {
                            return;
                        }
                        fields.push(key);
                    });
                });
                const uniqueFields = [...new Set(fields)].sort((a, b) => a.localeCompare(b)).slice(0, 300);
                additionalFieldKeyEl.innerHTML = '<option value="">Select field...</option>';
                uniqueFields.forEach((fieldKey) => {
                    const option = document.createElement('option');
                    option.value = fieldKey;
                    option.textContent = fieldKey;
                    additionalFieldKeyEl.appendChild(option);
                });

                loadCategoryAccordion();
            };

            const categoryAccordionEl = document.getElementById('idxCategoryAccordion');
            let categoryFieldsCache = {};

            const loadCategoryAccordion = async () => {
                try {
                    const response = await fetch('/agent/searches/fields', { headers: { Accept: 'application/json' } });
                    if (!response.ok) return;
                    const payload = await response.json();
                    const fields = payload?.data?.fields || [];
                    if (fields.length === 0) {
                        categoryAccordionEl.innerHTML = '<p class="text-xs text-slate-500">No categorized fields available.</p>';
                        return;
                    }

                    const groups = {};
                    fields.forEach((field) => {
                        const cat = field.category || 'additional_fields';
                        if (!groups[cat]) groups[cat] = [];
                        groups[cat].push(field);
                    });

                    categoryFieldsCache = groups;
                    categoryAccordionEl.innerHTML = '';

                    const categoryLabels = {
                        general: 'General',
                        locations: 'Locations',
                        school_boundaries: 'School Boundaries',
                        excluded_boundaries: 'Excluded Boundaries',
                        schools: 'Schools (MLS)',
                        features: 'Features',
                        amenities: 'Amenities',
                        dates: 'Dates',
                        open_house_photos: 'Open House / Photos',
                        additional_fields: 'Additional Fields',
                    };

                    const categoryOrder = ['general', 'locations', 'features', 'amenities', 'dates', 'schools', 'school_boundaries', 'excluded_boundaries', 'open_house_photos', 'additional_fields'];

                    categoryOrder.forEach((cat) => {
                        const catFields = groups[cat];
                        if (!catFields || catFields.length === 0) return;

                        const label = categoryLabels[cat] || cat;
                        const section = document.createElement('div');
                        section.className = 'rounded-lg border border-slate-200 dark:border-slate-700';

                        const header = document.createElement('button');
                        header.type = 'button';
                        header.className = 'flex w-full items-center justify-between px-3 py-2 text-xs font-semibold text-slate-700 dark:text-slate-200';
                        header.innerHTML = `<span>${label} (${catFields.length})</span><span class="idx-category-chevron transition-transform">▸</span>`;

                        const body = document.createElement('div');
                        body.className = 'hidden border-t border-slate-200 px-3 py-2 dark:border-slate-700';
                        body.innerHTML = '';

                        const fieldList = document.createElement('div');
                        fieldList.className = 'grid gap-1 md:grid-cols-2';
                        catFields.slice(0, 20).forEach((field) => {
                            const chip = document.createElement('button');
                            chip.type = 'button';
                            chip.className = 'truncate rounded border border-slate-200 px-2 py-1 text-left text-[11px] text-slate-600 hover:border-cyan-300 hover:text-cyan-700 dark:border-slate-700 dark:text-slate-300 dark:hover:border-cyan-800 dark:hover:text-cyan-300';
                            chip.textContent = field.display_label || field.source_field_key;
                            chip.title = `${field.source_field_key} (${field.field_type})`;
                            chip.addEventListener('click', () => {
                                additionalFieldKeyEl.value = field.source_field_key;
                                additionalFieldKeyEl.dispatchEvent(new Event('change'));
                            });
                            fieldList.appendChild(chip);
                        });
                        body.appendChild(fieldList);

                        header.addEventListener('click', () => {
                            const isHidden = body.classList.contains('hidden');
                            body.classList.toggle('hidden');
                            const chevron = header.querySelector('.idx-category-chevron');
                            if (chevron) chevron.style.transform = isHidden ? 'rotate(90deg)' : '';
                        });

                        section.appendChild(header);
                        section.appendChild(body);
                        categoryAccordionEl.appendChild(section);
                    });
                } catch {}
            };

            const inferInputType = (fieldName) => {
                const lower = fieldName.toLowerCase();
                const numericHints = ['price', 'sqft', 'area', 'size', 'bedroom', 'bathroom', 'count', 'number', 'year', 'lot', 'days', 'latitude', 'longitude', 'depth', 'frontage', 'length', 'width', 'garage'];
                const dateHints = ['date', 'timestamp', 'on_market'];
                for (const hint of dateHints) {
                    if (lower.includes(hint)) return 'date';
                }
                for (const hint of numericHints) {
                    if (lower.includes(hint)) return 'number';
                }
                return 'text';
            };

            additionalFieldKeyEl?.addEventListener('change', async () => {
                const selectedField = additionalFieldKeyEl.value;
                additionalFieldSuggestionsEl.innerHTML = '';
                additionalFieldValueEl.type = inferInputType(selectedField || '');
                additionalFieldValueEl.placeholder = additionalFieldValueEl.type === 'date' ? 'YYYY-MM-DD' : (additionalFieldValueEl.type === 'number' ? '0' : 'Filter value');
                if (!selectedField) {
                    return;
                }

                const localEntries = [];
                fieldCatalogCache.forEach((scopeRow) => {
                    const values = Array.isArray(scopeRow?.values) ? scopeRow.values : [];
                    values.forEach((entry) => {
                        if (String(entry?.LookupName || '') === selectedField && Array.isArray(entry?.LookupValue)) {
                            entry.LookupValue.forEach((lv) => {
                                const label = String(lv?.LookupValue || lv?.Value || '');
                                if (label !== '') {
                                    localEntries.push(label);
                                }
                            });
                        }
                    });
                });

                if (localEntries.length > 0) {
                    const unique = [...new Set(localEntries)].sort((a, b) => a.localeCompare(b)).slice(0, 200);
                    unique.forEach((val) => {
                        const opt = document.createElement('option');
                        opt.value = val;
                        additionalFieldSuggestionsEl.appendChild(opt);
                    });
                    return;
                }

                try {
                    const resp = await fetch(`/agent/searches/lookups/options?field=${encodeURIComponent(selectedField)}`, {
                        headers: { Accept: 'application/json' },
                    });
                    if (!resp.ok) return;
                    const data = await resp.json();
                    const items = Array.isArray(data?.data) ? data.data : [];
                    const enumVals = [];
                    items.forEach((item) => {
                        const v = String(item?.LookupValue || item?.value || '');
                        if (v !== '') enumVals.push(v);
                    });
                    const unique = [...new Set(enumVals)].sort((a, b) => a.localeCompare(b)).slice(0, 200);
                    unique.forEach((val) => {
                        const opt = document.createElement('option');
                        opt.value = val;
                        additionalFieldSuggestionsEl.appendChild(opt);
                    });
                } catch (_) { /* non-critical */ }
            });

            const renderResults = (payload) => {
                const items = payload?.data?.items || [];
                statusEl.textContent = 'Search complete.';
                summaryEl.textContent = `${items.length} listing(s) returned from ${payload?.data?.meta?.sources ?? 0} source(s).`;

                markersLayer.clearLayers();
                const bounds = [];

                items.slice(0, 100).forEach((item) => {
                    const row = document.createElement('div');
                    row.className = 'rounded-md border border-slate-200/80 bg-slate-50 px-2 py-2 text-xs dark:border-slate-700 dark:bg-slate-900/60';
                    row.innerHTML = `
                        <div class="font-semibold text-slate-800 dark:text-slate-100">${item.fullAddress || item.listingId || 'Listing'}</div>
                        <div class="text-slate-600 dark:text-slate-300">
                            ${item.listPrice ? ('$' + Number(item.listPrice).toLocaleString()) : 'N/A'} ·
                            ${item.bedroomsTotal ?? '-'} bd · ${item.bathroomsTotal ?? '-'} ba
                        </div>
                        <div class="text-slate-500 dark:text-slate-400">${item.source_dataset || ''} / ${item.source_mls || ''}</div>
                    `;
                    resultsEl.appendChild(row);

                    const lat = Number(item.latitude ?? NaN);
                    const lng = Number(item.longitude ?? NaN);
                    if (Number.isFinite(lat) && Number.isFinite(lng)) {
                        const marker = L.marker([lat, lng]);
                        const popupContent = `
                            <div style="font-size:12px;max-width:200px;">
                                <strong>${item.fullAddress || item.listingId || 'Listing'}</strong><br/>
                                ${item.listPrice ? ('$' + Number(item.listPrice).toLocaleString()) : 'N/A'} ·
                                ${item.bedroomsTotal ?? '-'} bd · ${item.bathroomsTotal ?? '-'} ba
                            </div>
                        `;
                        marker.bindPopup(popupContent);
                        markersLayer.addLayer(marker);
                        bounds.push([lat, lng]);
                    }
                });

                if (bounds.length > 0 && (!mapAutoFitEl || mapAutoFitEl.checked)) {
                    bumpSuppressPanSearch();
                    map.fitBounds(bounds, { padding: [30, 30], maxZoom: 14 });
                }
                if (clearMarkersBtn) {
                    clearMarkersBtn.classList.toggle('hidden', markersLayer.getLayers().length === 0);
                }
                searchThisAreaBtn?.classList.add('hidden');
                lastSearchFilters = currentFilters();
            };

            const searchOnPanEl = document.getElementById('idxSearchOnPan');
            let panSearchTimer = null;
            const PAN_DEBOUNCE_MS = 600;

            const runViewportExecute = async (options) => {
                const hideManualButton = Boolean(options?.hideManualButton);
                const bounds = map.getBounds();
                const ne = bounds.getNorthEast();
                const sw = bounds.getSouthWest();
                const baseFilters =
                    lastSearchFilters && lastSearchFilters.length > 0 ? lastSearchFilters : currentFilters();
                const geoms = serializeLayers();
                if (baseFilters.length === 0 && geoms.length === 0) {
                    return;
                }
                if (!hideManualButton) {
                    statusEl.textContent = 'Searching visible area...';
                } else {
                    statusEl.textContent = 'Searching map view…';
                }
                resultsEl.innerHTML = '';
                summaryEl.textContent = '';
                if (!hideManualButton) {
                    searchThisAreaBtn?.classList.add('hidden');
                }
                const viewportFilters = baseFilters.filter(
                    (f) => !['location.viewport', 'property.latitude', 'property.longitude'].includes(f.field),
                );
                viewportFilters.push({
                    field: 'location.viewport',
                    operator: 'bbox',
                    value: `${sw.lat},${sw.lng},${ne.lat},${ne.lng}`,
                });
                try {
                    const response = await fetch('/agent/searches/execute', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            Accept: 'application/json',
                        },
                        body: JSON.stringify({
                            filters: viewportFilters,
                            geometries: geoms,
                            telemetry: { trigger: hideManualButton ? 'viewport_pan' : 'viewport' },
                        }),
                    });
                    const payload = await response.json();
                    if (!response.ok) {
                        statusEl.textContent = hideManualButton ? 'Pan search failed.' : 'Viewport search failed.';
                        return;
                    }
                    renderResults(payload);
                } catch (_) {
                    statusEl.textContent = hideManualButton ? 'Pan search failed.' : 'Viewport search failed.';
                }
            };

            const schedulePanSearch = () => {
                if (!searchOnPanEl?.checked || suppressPanSearch > 0) {
                    return;
                }
                const baseFilters =
                    lastSearchFilters && lastSearchFilters.length > 0 ? lastSearchFilters : currentFilters();
                const geoms = serializeLayers();
                if (baseFilters.length === 0 && geoms.length === 0) {
                    return;
                }
                window.clearTimeout(panSearchTimer);
                panSearchTimer = window.setTimeout(() => {
                    panSearchTimer = null;
                    if (!searchOnPanEl?.checked || suppressPanSearch > 0) {
                        return;
                    }
                    runViewportExecute({ hideManualButton: true }).catch(() => {
                        statusEl.textContent = 'Pan search failed.';
                    });
                }, PAN_DEBOUNCE_MS);
            };

            map.on('moveend', () => {
                if (searchThisAreaBtn && lastSearchFilters && lastSearchFilters.length > 0) {
                    searchThisAreaBtn.classList.remove('hidden');
                }
                schedulePanSearch();
            });
            map.on('zoomend', schedulePanSearch);

            searchThisAreaBtn?.addEventListener('click', () => {
                runViewportExecute({ hideManualButton: false }).catch(() => {
                    statusEl.textContent = 'Viewport search failed.';
                });
            });

            searchOnPanEl?.addEventListener('change', () => {
                schedulePanSearch();
            });

            geocodeBtn?.addEventListener('click', async () => {
                const query = (geocodeInput?.value || '').trim();
                if (!query) {
                    statusEl.textContent = 'Enter a location to search.';
                    return;
                }
                statusEl.textContent = 'Geocoding...';
                try {
                    const resp = await fetch(
                        `/agent/searches/geocode?${new URLSearchParams({ q: query, limit: '1' })}`,
                        { headers: { Accept: 'application/json' } },
                    );
                    const payload = await resp.json();
                    if (!resp.ok) {
                        statusEl.textContent = payload?.message || 'Geocode failed.';
                        return;
                    }
                    const results = payload?.data;
                    if (Array.isArray(results) && results.length > 0) {
                        const lat = parseFloat(results[0].lat);
                        const lon = parseFloat(results[0].lon);
                        if (!Number.isNaN(lat) && !Number.isNaN(lon)) {
                            bumpSuppressPanSearch();
                            map.setView([lat, lon], 13);
                            statusEl.textContent = 'Map moved to geocoded location.';
                            return;
                        }
                    }
                    statusEl.textContent = 'No results for that query.';
                } catch (_) {
                    statusEl.textContent = 'Geocode failed.';
                }
            });

            geocodeInput?.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    geocodeBtn?.click();
                }
            });

            saveBtnEl.addEventListener('click', async () => {
                const name = saveNameEl.value.trim();
                if (name === '') {
                    statusEl.textContent = 'Provide a search name first.';
                    return;
                }
                statusEl.textContent = 'Saving search...';
                try {
                    const response = await fetch('/agent/searches', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({
                            name,
                            source: 'manual',
                            search_state_json: {
                                min_price: minPriceEl.value || null,
                                min_beds: minBedsEl.value || null,
                                city: cityEl.value.trim() || null,
                                additional_filters: additionalFilters,
                            },
                            filters: mapFiltersToPersisted(currentFilters()),
                            geometries: serializeLayers(),
                        }),
                    });
                    if (!response.ok) {
                        const payload = await response.json();
                        statusEl.textContent = 'Save failed.';
                        summaryEl.textContent = JSON.stringify(payload.errors || payload, null, 2);
                        return;
                    }
                    await refreshSavedSearches();
                    statusEl.textContent = 'Search saved.';
                } catch (error) {
                    statusEl.textContent = 'Save failed.';
                    summaryEl.textContent = String(error);
                }
            });

            loadBtnEl.addEventListener('click', async () => {
                const id = Number(savedSelectEl.value || 0);
                if (!id) {
                    statusEl.textContent = 'Select a saved search to load.';
                    return;
                }
                const selected = savedSearches.find((row) => Number(row.id) === id);
                if (!selected) {
                    statusEl.textContent = 'Saved search not found.';
                    return;
                }
                const state = selected.search_state_json || {};
                minPriceEl.value = state.min_price ?? '';
                minBedsEl.value = state.min_beds ?? '';
                cityEl.value = state.city ?? '';
                additionalFilters = Array.isArray(state.additional_filters)
                    ? state.additional_filters.filter((item) => item && item.field)
                    : [];
                renderAdditionalFilters();
                clearAndRenderGeometries(selected.geometries || []);
                statusEl.textContent = 'Saved search loaded.';
            });

            addAdditionalFilterBtnEl.addEventListener('click', () => {
                const field = String(additionalFieldKeyEl?.value || '').trim();
                const operator = String(additionalFieldOperatorEl?.value || 'eq').trim();
                const rawValue = String(additionalFieldValueEl?.value || '').trim();
                if (field === '' || rawValue === '') {
                    statusEl.textContent = 'Select additional field and value first.';
                    return;
                }
                const inputType = inferInputType(field);
                let finalValue = rawValue;
                if (inputType === 'number') {
                    const num = Number(rawValue);
                    if (!isNaN(num)) {
                        finalValue = num;
                    }
                }
                additionalFilters.push({ field, operator, value: finalValue });
                additionalFieldValueEl.value = '';
                renderAdditionalFilters();
                statusEl.textContent = 'Additional filter added.';
            });

            additionalFilterRowsEl.addEventListener('click', (event) => {
                const button = event.target.closest('button[data-filter-index]');
                if (!(button instanceof HTMLElement)) {
                    return;
                }
                const index = Number(button.dataset.filterIndex || -1);
                if (index < 0 || index >= additionalFilters.length) {
                    return;
                }
                additionalFilters.splice(index, 1);
                renderAdditionalFilters();
                statusEl.textContent = 'Additional filter removed.';
            });

            createAlertBtnEl.addEventListener('click', async () => {
                const id = Number(savedSelectEl.value || 0);
                if (!id) {
                    statusEl.textContent = 'Select a saved search before creating an alert.';
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
                            agent_search_id: id,
                            name: `Alert for search #${id}`,
                            alert_type: 'listing',
                            status: 'active',
                            schedule_json: { cadence: 'daily' },
                        }),
                    });
                    if (!response.ok) {
                        const payload = await response.json();
                        statusEl.textContent = 'Alert creation failed.';
                        summaryEl.textContent = JSON.stringify(payload.errors || payload, null, 2);
                        return;
                    }
                    statusEl.textContent = 'Alert created.';
                } catch (error) {
                    statusEl.textContent = 'Alert creation failed.';
                    summaryEl.textContent = String(error);
                }
            });

            runBtn.addEventListener('click', async () => {
                const selectedId = Number(savedSelectEl.value || 0);
                const selected = selectedId ? savedSearches.find((row) => Number(row.id) === selectedId) : null;
                const filters = selected
                    ? mapPersistedToExecuteFilters(selected.filters || [])
                    : currentFilters();
                const geometries = selected ? (selected.geometries || []) : serializeLayers();

                statusEl.textContent = 'Running search...';
                resultsEl.innerHTML = '';
                summaryEl.textContent = '';

                try {
                    const response = await fetch('/agent/searches/execute', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({
                            filters,
                            geometries,
                            telemetry: { trigger: selectedId ? 'saved_loaded' : 'manual' },
                        }),
                    });
                    const payload = await response.json();
                    if (!response.ok) {
                        statusEl.textContent = 'Search failed.';
                        summaryEl.textContent = JSON.stringify(payload.errors || payload, null, 2);
                        return;
                    }

                    renderResults(payload);
                } catch (error) {
                    statusEl.textContent = 'Search failed.';
                    summaryEl.textContent = String(error);
                }
            });

            shareSearchBtnEl?.addEventListener('click', async () => {
                const filters = currentFilters();
                const geometries = serializeLayers();
                if (filters.length === 0 && geometries.length === 0) {
                    statusEl.textContent = 'Set filters or draw shapes first.';
                    return;
                }

                try {
                    const response = await fetch('/agent/searches/serialize', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                            'Accept': 'application/json',
                        },
                        body: JSON.stringify({ filters, geometries }),
                    });
                    const payload = await response.json();
                    if (!response.ok) {
                        statusEl.textContent = 'Failed to generate share URL.';
                        return;
                    }
                    shareUrlInputEl.value = payload?.data?.url || '';
                    shareUrlBoxEl.classList.remove('hidden');
                    statusEl.textContent = 'Share URL generated.';
                } catch (_) {
                    statusEl.textContent = 'Failed to generate share URL.';
                }
            });

            copyShareUrlBtnEl?.addEventListener('click', () => {
                if (shareUrlInputEl?.value) {
                    navigator.clipboard.writeText(shareUrlInputEl.value).then(() => {
                        statusEl.textContent = 'URL copied to clipboard.';
                    }).catch(() => {
                        shareUrlInputEl.select();
                        statusEl.textContent = 'Select and copy the URL manually.';
                    });
                }
            });

            refreshSavedSearches().catch(() => {
                statusEl.textContent = 'Unable to load saved searches.';
            });
            loadFieldCatalog().catch(() => {
                statusEl.textContent = 'Unable to load additional filter catalog.';
            });
            renderAdditionalFilters();

            setTimeout(() => map.invalidateSize(), 250);
        });
    </script>
</x-filament-panels::page>
