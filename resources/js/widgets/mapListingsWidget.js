import {
  createWidgetRoot,
  debounce,
  ensureLeafletAssets,
  fetchJson,
  getRuntime,
  renderComplianceFooter,
} from "./runtime";

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function formatPrice(n) {
  if (n === null || n === undefined || Number.isNaN(Number(n))) {
    return "—";
  }
  try {
    return new Intl.NumberFormat(undefined, { style: "currency", currency: "USD", maximumFractionDigits: 0 }).format(
      Number(n),
    );
  } catch {
    return `$${Math.round(Number(n)).toLocaleString()}`;
  }
}

export async function renderMapListingsWidget(target) {
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
  const footerHost = root.querySelector("[data-role='q-footer']");
  renderComplianceFooter(footerHost);

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
    if (!window.L || !map || !markersLayer) {
      return;
    }
    markersLayer.eachLayer((layer) => {
      const lid = layer.listingId;
      if (!lid || typeof layer.setStyle !== "function") {
        return;
      }
      const on = lid === selectedId;
      layer.setStyle({
        weight: on ? 3 : 2,
        fillOpacity: on ? 0.85 : 0.35,
      });
    });
  };

  const renderList = (items) => {
    if (!items.length) {
      listEl.innerHTML = `<p style="padding:12px;font-size:13px;">No listings match. Try another city.</p>`;
      return;
    }
    listEl.innerHTML = items
      .map((r) => {
        const id = escapeHtml(r.listingId || "");
        const addr = escapeHtml(r.fullAddress || r.city || "Address on file");
        const price = formatPrice(r.listPrice);
        const beds = r.bedroomsTotal != null ? `${r.bedroomsTotal} bd` : "";
        const baths = r.bathroomsTotal != null ? `${r.bathroomsTotal} ba` : "";
        const sq = r.livingArea != null ? `${r.livingArea} sqft` : "";
        const meta = [beds, baths, sq].filter(Boolean).join(" · ");
        const thumb = r.primaryImage?.thumbnail || r.primaryImage?.url || "";
        const img = thumb
          ? `<img src="${escapeHtml(thumb)}" alt="" style="width:72px;height:56px;object-fit:cover;border-radius:8px;flex-shrink:0;" loading="lazy" />`
          : `<div style="width:72px;height:56px;border-radius:8px;background:#e2e8f0;flex-shrink:0;"></div>`;
        return `<button type="button" data-listing-id="${id}" style="display:flex;gap:10px;width:100%;text-align:left;padding:10px;margin-bottom:8px;border:1px solid #e2e8f0;border-radius:12px;background:var(--qgx-background);cursor:pointer;">
          ${img}
          <span style="min-width:0;">
            <span style="display:block;font-weight:700;color:var(--qgx-primary);">${price}</span>
            <span style="display:block;font-size:13px;margin-top:2px;">${addr}</span>
            <span style="display:block;font-size:11px;color:#64748b;margin-top:4px;">${escapeHtml(meta)}</span>
          </span>
        </button>`;
      })
      .join("");
    listEl.querySelectorAll("[data-listing-id]").forEach((btn) => {
      btn.addEventListener("click", () => {
        selectedId = btn.getAttribute("data-listing-id");
        applySelection();
      });
    });
    applySelection();
  };

  const renderMarkers = (items) => {
    if (!window.L || !map) {
      return;
    }
    markersLayer.clearLayers();
    const bounds = [];
    items.forEach((r) => {
      if (r.latitude == null || r.longitude == null) {
        return;
      }
      const lat = Number(r.latitude);
      const lng = Number(r.longitude);
      const m = window.L.circleMarker([lat, lng], {
        radius: 9,
        color: "var(--qgx-primary)",
        weight: 2,
        fillColor: "var(--qgx-primary)",
        fillOpacity: 0.35,
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
    if (bounds.length) {
      map.fitBounds(bounds, { padding: [24, 24], maxZoom: 14 });
    }
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
          "page.skip": 0,
        }),
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
    window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
    }).addTo(map);
    markersLayer = window.L.layerGroup().addTo(map);
  }

  const mq = window.matchMedia("(max-width: 768px)");
  const split = root.querySelector("[data-role='split']");
  const applyMq = () => {
    if (!split) {
      return;
    }
    split.style.gridTemplateColumns = mq.matches ? "1fr" : "1fr 1fr";
  };
  applyMq();
  mq.addEventListener("change", applyMq);
}
