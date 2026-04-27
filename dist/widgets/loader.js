(function() {
	//#region resources/js/widgets/runtime.js
	var runtimeState = {
		apiBase: "",
		token: "",
		primaryColor: "#3b82f6",
		secondaryColor: "#1e40af",
		accentColor: "#10b981",
		textColor: "#0f172a",
		backgroundColor: "#ffffff",
		listingsPerPage: 20,
		widgetTheme: "light",
		validation: null,
		mapAssetsLoaded: false
	};
	function configureRuntime(input) {
		Object.assign(runtimeState, input);
	}
	function getRuntime() {
		return runtimeState;
	}
	function createWidgetRoot(target, kind) {
		const root = document.createElement("section");
		root.className = "qgx-widget";
		root.dataset.quantyraWidget = kind;
		root.style.setProperty("--qgx-primary", runtimeState.primaryColor);
		root.style.setProperty("--qgx-secondary", runtimeState.secondaryColor);
		root.style.setProperty("--qgx-accent", runtimeState.accentColor);
		root.style.setProperty("--qgx-text", runtimeState.textColor);
		root.style.setProperty("--qgx-background", runtimeState.backgroundColor);
		root.dataset.qgxTheme = runtimeState.widgetTheme || "light";
		target.appendChild(root);
		return root;
	}
	function renderComplianceFooter(container) {
		const branding = (runtimeState.validation || {}).branding || {};
		const nowIso = (/* @__PURE__ */ new Date()).toISOString();
		container.innerHTML = `
    <div class="qgx-footer" data-quantyragidx-footer="true" style="font-family: Inter, Arial, sans-serif; font-size: 12px; line-height: 1.5; color: var(--qgx-text); background: var(--qgx-background); border-top: 1px solid #cbd5e1; padding: 12px;">
      <p><strong>${branding.brokerage || "Brokerage information required"}</strong></p>
      <p>${branding.sourceAttribution || "Listings courtesy of Stellar MLS as distributed by MLS GRID"}</p>
      <p>Based on information submitted to the MLS GRID as of ${nowIso}.</p>
      <p>${branding.consumerDisclaimer || ""}</p>
      <p>Anti-scraping notice: Any use or search of data other than by a consumer looking to purchase real estate is prohibited.</p>
      <p>DMCA requests: <a href="mailto:${branding.dmcaEmail || "support@quantyralabs.cc"}">${branding.dmcaEmail || "support@quantyralabs.cc"}</a></p>
    </div>
  `;
	}
	function renderBlockedMessage(target, message) {
		target.innerHTML = `<div style="border:2px solid #ef4444;background:#fef2f2;color:#7f1d1d;padding:16px;font-family:Inter,Arial,sans-serif;">
    <strong>GeoIDX compliance block</strong><div style="margin-top:8px;">${message}</div>
  </div>`;
	}
	function debounce(fn, waitMs) {
		let handle = null;
		return (...args) => {
			window.clearTimeout(handle);
			handle = window.setTimeout(() => fn(...args), waitMs);
		};
	}
	async function fetchJson(path, options = {}) {
		const { headers: extraHeaders, ...rest } = options;
		const response = await fetch(`${runtimeState.apiBase}${path}`, {
			...rest,
			headers: {
				Accept: "application/json",
				...extraHeaders || {}
			}
		});
		if (!response.ok) {
			const msg = (await response.json().catch(() => ({}))).message || `Widget request failed with status ${response.status}`;
			throw new Error(msg);
		}
		return response.json();
	}
	async function ensureLeafletAssets() {
		if (runtimeState.mapAssetsLoaded) return;
		await new Promise((resolve, reject) => {
			const cssId = "qgx-leaflet-css";
			if (!document.getElementById(cssId)) {
				const css = document.createElement("link");
				css.id = cssId;
				css.rel = "stylesheet";
				css.href = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css";
				document.head.appendChild(css);
			}
			const jsId = "qgx-leaflet-js";
			if (document.getElementById(jsId)) {
				resolve();
				return;
			}
			const script = document.createElement("script");
			script.id = jsId;
			script.src = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js";
			script.async = true;
			script.onload = () => resolve();
			script.onerror = () => reject(/* @__PURE__ */ new Error("Failed to load Leaflet assets"));
			document.head.appendChild(script);
		});
		runtimeState.mapAssetsLoaded = true;
	}
	//#endregion
	//#region resources/js/widgets/mapListingsWidget.js
	function escapeHtml(s) {
		return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
	}
	function formatPrice(n) {
		if (n === null || n === void 0 || Number.isNaN(Number(n))) return "—";
		try {
			return new Intl.NumberFormat(void 0, {
				style: "currency",
				currency: "USD",
				maximumFractionDigits: 0
			}).format(Number(n));
		} catch {
			return `$${Math.round(Number(n)).toLocaleString()}`;
		}
	}
	async function renderMapListingsWidget(target) {
		const root = createWidgetRoot(target, "map");
		const perPage = Math.min(48, Math.max(6, Number(getRuntime().listingsPerPage) || 12));
		root.innerHTML = `
    <div class="qgx-maplist" style="font-family:Inter,system-ui,sans-serif;color:var(--qgx-text);background:var(--qgx-background);border:1px solid #e2e8f0;border-radius:12px;overflow:hidden;">
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:0;min-height:320px;" data-role="split">
        <div style="border-right:1px solid #e2e8f0;display:flex;flex-direction:column;min-width:0;">
          <div style="padding:10px 12px;border-bottom:1px solid #e2e8f0;background:color-mix(in srgb, var(--qgx-primary) 8%, transparent);">
            <label style="display:block;font-weight:600;font-size:13px;margin-bottom:6px;">Search area</label>
            <input data-role="q-input" type="search" placeholder="City (e.g. Tampa)" style="width:100%;padding:8px 10px;border:1px solid #cbd5e1;border-radius:8px;font-size:14px;" />
          </div>
          <div data-role="q-list" style="flex:1;overflow:auto;padding:8px;max-height:420px;"></div>
          <p data-role="q-status" style="font-size:11px;padding:6px 10px;color:#64748b;border-top:1px solid #e2e8f0;"></p>
        </div>
        <div style="min-height:260px;position:relative;" data-role="q-map"></div>
      </div>
      <div data-role="q-footer" style="border-top:1px solid #e2e8f0;"></div>
    </div>
  `;
		const listEl = root.querySelector("[data-role='q-list']");
		const mapEl = root.querySelector("[data-role='q-map']");
		const input = root.querySelector("[data-role='q-input']");
		const statusEl = root.querySelector("[data-role='q-status']");
		renderComplianceFooter(root.querySelector("[data-role='q-footer']"));
		let map = null;
		let markersLayer = null;
		let rows = [];
		let selectedId = null;
		const applySelection = () => {
			listEl.querySelectorAll("[data-listing-id]").forEach((el) => {
				const id = el.getAttribute("data-listing-id");
				el.style.outline = id === selectedId ? "2px solid var(--qgx-primary)" : "none";
				el.style.borderRadius = "10px";
			});
			if (!window.L || !map || !markersLayer) return;
			markersLayer.eachLayer((layer) => {
				const lid = layer.listingId;
				if (!lid || typeof layer.setStyle !== "function") return;
				const on = lid === selectedId;
				layer.setStyle({
					weight: on ? 3 : 2,
					fillOpacity: on ? .85 : .35
				});
			});
		};
		const renderList = (items) => {
			if (!items.length) {
				listEl.innerHTML = `<p style="padding:12px;font-size:13px;">No listings match. Try another city.</p>`;
				return;
			}
			listEl.innerHTML = items.map((r) => {
				const id = escapeHtml(r.listingId || "");
				const addr = escapeHtml(r.fullAddress || r.city || "Address on file");
				const price = formatPrice(r.listPrice);
				const meta = [
					r.bedroomsTotal != null ? `${r.bedroomsTotal} bd` : "",
					r.bathroomsTotal != null ? `${r.bathroomsTotal} ba` : "",
					r.livingArea != null ? `${r.livingArea} sqft` : ""
				].filter(Boolean).join(" · ");
				const thumb = r.primaryImage?.thumbnail || r.primaryImage?.url || "";
				return `<button type="button" data-listing-id="${id}" style="display:flex;gap:10px;width:100%;text-align:left;padding:10px;margin-bottom:8px;border:1px solid #e2e8f0;border-radius:12px;background:var(--qgx-background);cursor:pointer;">
          ${thumb ? `<img src="${escapeHtml(thumb)}" alt="" style="width:72px;height:56px;object-fit:cover;border-radius:8px;flex-shrink:0;" loading="lazy" />` : `<div style="width:72px;height:56px;border-radius:8px;background:#e2e8f0;flex-shrink:0;"></div>`}
          <span style="min-width:0;">
            <span style="display:block;font-weight:700;color:var(--qgx-primary);">${price}</span>
            <span style="display:block;font-size:13px;margin-top:2px;">${addr}</span>
            <span style="display:block;font-size:11px;color:#64748b;margin-top:4px;">${escapeHtml(meta)}</span>
          </span>
        </button>`;
			}).join("");
			listEl.querySelectorAll("[data-listing-id]").forEach((btn) => {
				btn.addEventListener("click", () => {
					selectedId = btn.getAttribute("data-listing-id");
					applySelection();
				});
			});
			applySelection();
		};
		const renderMarkers = (items) => {
			if (!window.L || !map) return;
			markersLayer.clearLayers();
			const bounds = [];
			items.forEach((r) => {
				if (r.latitude == null || r.longitude == null) return;
				const lat = Number(r.latitude);
				const lng = Number(r.longitude);
				const m = window.L.circleMarker([lat, lng], {
					radius: 9,
					color: "var(--qgx-primary)",
					weight: 2,
					fillColor: "var(--qgx-primary)",
					fillOpacity: .35
				});
				m.listingId = r.listingId;
				m.on("click", () => {
					selectedId = r.listingId;
					applySelection();
				});
				m.bindTooltip(formatPrice(r.listPrice), { direction: "top" });
				markersLayer.addLayer(m);
				bounds.push([lat, lng]);
			});
			if (bounds.length) map.fitBounds(bounds, {
				padding: [24, 24],
				maxZoom: 14
			});
		};
		const runSearch = debounce(async () => {
			const city = input.value.trim();
			if (city.length < 2) {
				listEl.innerHTML = "";
				statusEl.textContent = "";
				return;
			}
			statusEl.textContent = "Loading listings…";
			const runtime = getRuntime();
			const path = `/widget/api/listings-search?api_key=${encodeURIComponent(runtime.token)}`;
			try {
				const data = await fetchJson(path, {
					method: "POST",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						city,
						active_only: true,
						"page.limit": perPage,
						"page.skip": 0
					})
				});
				rows = Array.isArray(data.results) ? data.results : [];
				statusEl.textContent = `${rows.length} shown${data.has_more ? " (more available on full API access)" : ""}`;
				renderList(rows);
				renderMarkers(rows);
			} catch (e) {
				statusEl.textContent = e instanceof Error ? e.message : "Search failed.";
				listEl.innerHTML = "";
			}
		}, 350);
		input.addEventListener("input", runSearch);
		await ensureLeafletAssets();
		if (window.L) {
			map = window.L.map(mapEl).setView([27.9506, -82.4572], 10);
			window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", { attribution: "&copy; OpenStreetMap contributors" }).addTo(map);
			markersLayer = window.L.layerGroup().addTo(map);
		}
		const mq = window.matchMedia("(max-width: 768px)");
		const split = root.querySelector("[data-role='split']");
		const applyMq = () => {
			if (!split) return;
			split.style.gridTemplateColumns = mq.matches ? "1fr" : "1fr 1fr";
		};
		applyMq();
		mq.addEventListener("change", applyMq);
	}
	//#endregion
	//#region resources/js/widgets/widgets.js
	var DISCLOSURE = "Some IDX listings have been excluded from this website.";
	async function renderSearchWidget(target) {
		const root = createWidgetRoot(target, "search");
		root.innerHTML = `
    <div style="font-family:Inter,Arial,sans-serif;background:var(--qgx-background);color:var(--qgx-text);border:1px solid #e2e8f0;border-radius:12px;padding:12px;">
      <label style="display:block;font-weight:600;margin-bottom:8px;">Search listings</label>
      <input data-role="search-input" type="search" placeholder="City, zip, subdivision" style="width:100%;padding:10px;border:1px solid #cbd5e1;border-radius:8px;" />
      <p style="font-size:12px;margin-top:8px;">${DISCLOSURE}</p>
      <div data-role="results" style="margin-top:10px;"></div>
    </div>
  `;
		const input = root.querySelector("[data-role='search-input']");
		const results = root.querySelector("[data-role='results']");
		const runSearch = debounce(async () => {
			if (input.value.trim().length < 2) {
				results.innerHTML = "";
				return;
			}
			results.innerHTML = "<p>Searching...</p>";
			try {
				const runtime = getRuntime();
				results.innerHTML = `<p>Search ready for location ${(await fetchJson(`/widget/config/${encodeURIComponent(runtime.token)}`)).location_id}. Listings are served through authenticated API proxy routes.</p>`;
			} catch (_error) {
				results.innerHTML = "<p>Unable to search right now.</p>";
			}
		}, 300);
		input.addEventListener("input", runSearch);
	}
	async function renderMapWidget(target) {
		await renderMapListingsWidget(target);
	}
	async function renderCommunityWidget(target) {
		const root = createWidgetRoot(target, "community");
		root.innerHTML = `
    <div style="font-family:Inter,Arial,sans-serif;border:1px solid #e2e8f0;border-radius:12px;padding:12px;">
      <h3 style="margin:0 0 8px;">Community Listings</h3>
      <p style="margin:0 0 8px;">Use objective criteria only (location, price, property type).</p>
      <ul style="padding-left:18px;margin:0;">
        <li>Listings rendered through authenticated proxy only</li>
        <li>No seller contact fields are displayed</li>
        <li>Thumbnail cards stay within allowed field limits</li>
      </ul>
    </div>
  `;
	}
	async function renderPropertyWidget(target) {
		const root = createWidgetRoot(target, "property");
		root.innerHTML = `
    <div style="font-family:Inter,Arial,sans-serif;border:1px solid #e2e8f0;border-radius:12px;padding:12px;">
      <h3 style="margin:0 0 8px;">Property Details</h3>
      <p style="margin:0 0 10px;">Lead actions are OTP-gated before contact reveal.</p>
      <button data-role="lead-btn" style="background:var(--qgx-primary);color:#fff;border:0;border-radius:8px;padding:10px 12px;cursor:pointer;">Request details</button>
      <div data-role="lead-status" style="font-size:12px;margin-top:8px;"></div>
    </div>
  `;
		root.querySelector("[data-role='lead-btn']").addEventListener("click", async () => {
			const status = root.querySelector("[data-role='lead-status']");
			status.textContent = "Submitting lead...";
			try {
				await fetchJson(`/widget/api/leads?api_key=${encodeURIComponent(getRuntime().token)}`, {
					method: "POST",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						first_name: "GeoIDX",
						last_name: "Prospect",
						email: "prospect@example.com",
						phone: "555-555-5555",
						message: "OTP-gated property detail request",
						source: "property-widget",
						mls_listing_id: "sample-listing"
					})
				});
				status.textContent = "Lead captured. OTP flow initiated.";
			} catch (_error) {
				status.textContent = "Lead capture failed.";
			}
		});
	}
	async function renderFooterWidget(target) {
		renderComplianceFooter(createWidgetRoot(target, "footer"));
	}
	//#endregion
	//#region resources/js/widgets/loader.js
	var HANDLERS = {
		search: renderSearchWidget,
		map: renderMapWidget,
		community: renderCommunityWidget,
		property: renderPropertyWidget,
		footer: renderFooterWidget
	};
	async function validateDomain(apiBase, token, requireFooter) {
		const response = await fetch(`${apiBase}/api/widgets/validate`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				token,
				hostname: window.location.hostname,
				referrer: document.referrer || null,
				requireFooter
			})
		});
		const data = await response.json().catch(() => ({}));
		if (!response.ok || !data.ok) throw new Error(data.message || "Widget host validation failed.");
		return data;
	}
	function parseLoaderConfig(scriptEl) {
		const src = new URL(scriptEl.src, window.location.origin);
		const token = src.searchParams.get("token") || scriptEl.dataset.apiKey || "";
		const primaryColor = `#${(src.searchParams.get("primaryColor") || "3b82f6").replace(/^#/, "")}`;
		const secondaryColor = `#${(src.searchParams.get("secondaryColor") || "1e40af").replace(/^#/, "")}`;
		const accentColor = `#${(src.searchParams.get("accentColor") || "10b981").replace(/^#/, "")}`;
		const textColor = `#${(src.searchParams.get("textColor") || "0f172a").replace(/^#/, "")}`;
		const backgroundColor = `#${(src.searchParams.get("backgroundColor") || "ffffff").replace(/^#/, "")}`;
		return {
			token,
			apiBase: `${src.protocol}//${src.host}`,
			requireFooter: (scriptEl.dataset.footerRequired || "true") !== "false",
			primaryColor,
			secondaryColor,
			accentColor,
			textColor,
			backgroundColor,
			listingsPerPage: 20
		};
	}
	function normalizeHex(value, fallback) {
		if (!value || typeof value !== "string") return fallback;
		let v = value.trim();
		if (!v.startsWith("#")) v = `#${v}`;
		if (/^#[0-9a-fA-F]{3}$/.test(v)) v = `#${v[1]}${v[1]}${v[2]}${v[2]}${v[3]}${v[3]}`;
		return /^#[0-9a-fA-F]{6}$/.test(v) ? v : fallback;
	}
	function mergeServerWidgetConfig(loaderCfg, api) {
		const accent = api.accent_color || api.secondary_color || loaderCfg.accentColor;
		return {
			...loaderCfg,
			primaryColor: normalizeHex(api.primary_color, loaderCfg.primaryColor),
			secondaryColor: normalizeHex(api.secondary_color, loaderCfg.secondaryColor),
			accentColor: normalizeHex(accent, loaderCfg.accentColor),
			textColor: normalizeHex(api.text_color, loaderCfg.textColor),
			backgroundColor: normalizeHex(api.background_color, loaderCfg.backgroundColor),
			listingsPerPage: Number(api.listings_per_page) || loaderCfg.listingsPerPage || 20,
			widgetTheme: api.theme === "dark" ? "dark" : "light",
			locationId: api.location_id || loaderCfg.locationId
		};
	}
	function applyQueryColorOverrides(cfg, scriptEl) {
		const src = new URL(scriptEl.src, window.location.origin);
		const pick = (param, cur) => {
			const raw = src.searchParams.get(param);
			if (raw === null || raw === "") return cur;
			return normalizeHex(raw, cur);
		};
		return {
			...cfg,
			primaryColor: pick("primaryColor", cfg.primaryColor),
			secondaryColor: pick("secondaryColor", cfg.secondaryColor),
			accentColor: pick("accentColor", cfg.accentColor),
			textColor: pick("textColor", cfg.textColor),
			backgroundColor: pick("backgroundColor", cfg.backgroundColor)
		};
	}
	async function fetchWidgetRuntimeConfig(apiBase, token) {
		const res = await fetch(`${apiBase}/widget/config/${encodeURIComponent(token)}`, { credentials: "omit" });
		if (!res.ok) throw new Error("Unable to load widget theme configuration.");
		return res.json();
	}
	function hasFooterAnchor() {
		return !!document.querySelector("[data-quantyragidx-footer=\"true\"]");
	}
	function ensureTailwindCdn() {
		const id = "qgx-tailwind-cdn";
		if (document.getElementById(id)) return;
		const script = document.createElement("script");
		script.id = id;
		script.src = "https://cdn.tailwindcss.com";
		script.defer = true;
		document.head.appendChild(script);
	}
	async function initWidget(type, options = {}) {
		const runtimeTarget = options.target || document.querySelector(`[data-quantyra-widget="${type}"]`);
		if (!runtimeTarget) throw new Error(`No mount target found for widget type "${type}"`);
		const handler = HANDLERS[type];
		if (!handler) throw new Error(`Unsupported widget type "${type}"`);
		await handler(runtimeTarget, options);
	}
	async function boot() {
		const scriptEl = document.currentScript || document.querySelector("script[src*=\"/widget/loader.js\"]");
		if (!scriptEl) return;
		const config = parseLoaderConfig(scriptEl);
		if (!config.token) {
			window.QuantyraGeoIDX = {
				async initWidget() {
					throw new Error("Missing subscriber token.");
				},
				getValidation: () => null
			};
			renderBlockedMessage(document.body, "Missing subscriber token.");
			return;
		}
		/** Resolves after host validation + runtime config; {@link initWidget} awaits this so embedders (e.g. dashboard) never see undefined. */
		let resolveReady;
		let rejectReady;
		const runtimeReady = new Promise((resolve, reject) => {
			resolveReady = resolve;
			rejectReady = reject;
		});
		let validationForGetter = null;
		window.QuantyraGeoIDX = {
			async initWidget(type, options = {}) {
				await runtimeReady;
				return initWidget(type, options);
			},
			getValidation: () => validationForGetter
		};
		let validation;
		try {
			validation = await validateDomain(config.apiBase, config.token, config.requireFooter);
			validationForGetter = validation;
			let merged = mergeServerWidgetConfig(config, await fetchWidgetRuntimeConfig(config.apiBase, config.token));
			merged = applyQueryColorOverrides(merged, scriptEl);
			configureRuntime({
				...merged,
				validation
			});
			ensureTailwindCdn();
		} catch (error) {
			rejectReady(error);
			renderBlockedMessage(document.body, error.message);
			return;
		}
		resolveReady();
		const autoWidget = scriptEl.dataset.widget;
		if (autoWidget) {
			if (autoWidget !== "footer" && config.requireFooter && !hasFooterAnchor()) {
				renderBlockedMessage(document.body, "Footer widget is required for MLS compliance before loading other widgets.");
				return;
			}
			const host = document.createElement("div");
			host.dataset.quantyraWidget = autoWidget;
			scriptEl.parentNode.insertBefore(host, scriptEl.nextSibling);
			await initWidget(autoWidget, { target: host });
		}
	}
	boot();
	//#endregion
})();

//# sourceMappingURL=loader.js.map