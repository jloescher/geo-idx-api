(function () {
  const REFRESH_MS = 30000;
  const MONITORING_URL = "/dashboard/monitoring/data";
  const TABS = ["overview", "ingest", "queues", "data", "infra", "integrations", "incidents"];
  let lastFetchAt = 0;
  let timer = null;
  let hasLiveData = false;
  let activeTab = "overview";

  function $(id) {
    return document.getElementById(id);
  }

  function fmtNum(n) {
    if (n === null || n === undefined) return "—";
    return Number(n).toLocaleString();
  }

  function fmtPct(n) {
    if (n === null || n === undefined) return "—";
    return `${Number(n).toFixed(1)}%`;
  }

  function fmtCryptoUsd(n) {
    if (n === null || n === undefined) return "—";
    return Number(n).toLocaleString(undefined, {
      minimumFractionDigits: 3,
      maximumFractionDigits: 3,
    });
  }

  function badgeClass(status) {
    if (status === "critical") return "badge badge-critical";
    if (status === "healthy") return "badge badge-healthy";
    if (status === "stale") return "badge badge-stale";
    return "badge badge-unknown";
  }

  function tile(label, value, sub, href, status, aria) {
    const tag = href ? "a" : "div";
    const hrefAttr = href ? ` href="${href}"` : "";
    const statusChip = status
      ? `<span class="${badgeClass(status)}">${status}</span>`
      : "";
    return `<${tag} class="metric-tile"${hrefAttr} aria-label="${aria || label}">
<span class="metric-label">${label}</span>
<span class="metric-value">${value}</span>
${sub ? `<span class="metric-sub">${sub}</span>` : ""}
${statusChip}
</${tag}>`;
  }

  function renderGroup(title, tiles) {
    if (!tiles.length) {
      tiles = [
        tile(title, "0", "No data yet", null, "unknown", title),
      ];
    }
    return `<div class="metric-group"><h3 class="metric-group-title">${title}</h3><div class="monitoring-grid">${tiles.join("")}</div></div>`;
  }

  function fmtSyncAge(iso) {
    if (!iso) return "Never synced";
    const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86400000);
    if (days <= 0) return "Synced today";
    if (days === 1) return "Synced 1d ago";
    return `Synced ${days}d ago`;
  }

  function overviewTiles(data) {
    const listings = data.listings || [];
    const staleListings = listings.filter((l) => l.status === "stale").length;
    const incidents = data.incidents || [];
    const cache = data.cache || {};
    const queues = data.queues || {};
    return [
      tile("Datasets", fmtNum(listings.length), `${fmtNum(staleListings)} stale`, null, staleListings > 0 ? "stale" : "healthy", "Listing datasets"),
      tile("Open incidents", fmtNum(incidents.length), "Across monitoring tabs", null, incidents.some((i) => i.severity === "critical") ? "critical" : incidents.length > 0 ? "stale" : "healthy", "Open incidents"),
      tile("Queue pending", fmtNum(queues.total_pending), `${fmtNum(queues.total_stale_reserved)} stale reserved`, null, queues.status || "unknown", "Queue pending jobs"),
      tile("Cache hit rate", fmtPct(cache.hit_rate_pct), `${cache.window_minutes || 15}m window`, null, cache.total > 0 ? "healthy" : "unknown", "Cache hit rate"),
    ];
  }

  function renderOverviewPanel(data) {
    return (
      renderGroup("Overview", overviewTiles(data)) +
      renderGroup("Activation", [
        tile("Domains", fmtNum(data.activation?.domain_count), `${fmtNum(data.activation?.verified_domain_count)} verified`, "/dashboard/domains", null, "Domains"),
        tile("API keys", fmtNum(data.activation?.token_count), "Per-domain keys", "/dashboard/domains", null, "API keys"),
        tile("Traffic 30d", fmtNum(data.activation?.domains_with_traffic_30d), "Domains with audit calls", "/dashboard/domains", null, "Activation traffic"),
      ])
    );
  }

  function renderIngestPanel(data) {
    const listings = (data.listings || []).map((row) => {
      const sub = `${fmtNum(row.active_pending)} active/pending · ${row.freshness_mode || "—"}`;
      return tile(
        row.dataset_slug || "dataset",
        fmtNum(row.total),
        sub,
        "/api/v1/bridge/stats",
        row.status,
        `Listings ${row.dataset_slug}`
      );
    });
    const pipelineRows = data.sync_pipeline?.by_status || [];
    const pipelineTiles = pipelineRows.map((row) =>
      tile(
        `${row.dataset_slug} · ${row.status}`,
        fmtNum(row.count),
        "replica_pages",
        null,
        row.status === "failed" ? "critical" : row.status === "pending" ? "stale" : "healthy",
        `Replica pages ${row.dataset_slug} ${row.status}`
      )
    );
    return renderGroup("Listings Replication", listings) + renderGroup("Replica Pipeline", pipelineTiles);
  }

  function renderDataPanel(data) {
    const gis = data.gis || {};
    const gisTiles = [
      tile("Parcels", fmtNum(gis.parcels_total), fmtSyncAge(gis.parcels_last_synced_at), null, gis.parcels_status || "unknown", "GIS parcels"),
      tile("Cities", fmtNum(gis.cities_total), fmtSyncAge(gis.cities_last_synced_at), null, gis.cities_status || "unknown", "GIS cities"),
      tile("Counties", fmtNum(gis.counties_total), fmtSyncAge(gis.counties_last_synced_at), null, gis.counties_status || "unknown", "GIS counties"),
      tile("ZIPs", fmtNum(gis.zips_total), fmtSyncAge(gis.zips_last_synced_at), null, gis.zips_status || "unknown", "GIS zips"),
    ];
    return renderGroup("GIS Data Health", gisTiles);
  }

  function renderIntegrationsPanel(data) {
    const cryptoAssets = (data.crypto && data.crypto.assets) || [];
    const cryptoTiles = cryptoAssets.length
      ? cryptoAssets.map((a) =>
          tile(
            a.asset_key.toUpperCase(),
            `$${fmtCryptoUsd(a.price_usd)}`,
            `${a.age_seconds}s ago`,
            null,
            a.stale ? "stale" : "healthy",
            `${a.asset_key} price`
          )
        )
      : [tile("Crypto", "—", "No snapshot", null, data.crypto?.status || "unknown", "Crypto prices")];
    return renderGroup("Integrations", cryptoTiles);
  }

  function renderQueuePanel(data) {
    const cache = data.cache || {};
    const cacheTiles = [
      tile(
        "Hit rate",
        fmtPct(cache.hit_rate_pct),
        `${cache.window_minutes || 15}m window`,
        null,
        cache.total > 0 ? "healthy" : "unknown",
        "Cache hit rate"
      ),
      tile("Audit requests", fmtNum(cache.total), `${fmtNum(cache.hits)} hits`, null, null, "Cache audit volume"),
    ];
    const queues = data.queues || {};
    const queueRows = queues.by_queue || [];
    const queueTiles = queueRows.slice(0, 4).map((q) => {
      const name = q.queue || "unknown";
      const pending = q.pending ?? 0;
      const reserved = q.reserved ?? 0;
      const failed = q.failed ?? 0;
      return tile(
        name,
        fmtNum(pending),
        `${fmtNum(reserved)} reserved · ${fmtNum(failed)} failed`,
        "#monitoring-queues",
        pending > 50 ? "stale" : "healthy",
        `Queue ${name}`
      );
    });
    if (!queueTiles.length) {
      queueTiles.push(tile("Queues", "0", "PostgreSQL jobs", "#monitoring-queues", "unknown", "Queue depth"));
    }
    const failedTiles = (queues.failed_top || []).slice(0, 4).map((row) =>
      tile(
        `${row.queue} · ${row.job_type}`,
        fmtNum(row.count),
        row.last_exception ? row.last_exception : "No exception detail",
        null,
        "stale",
        `Failed jobs ${row.queue} ${row.job_type}`
      )
    );
    return (
      renderGroup("Queue Health", queueTiles) +
      renderGroup("Queue Performance", cacheTiles) +
      renderGroup("Failed Jobs", failedTiles)
    );
  }

  function renderInfraPanel(data) {
    const infra = data.infrastructure || {};
    const scheduler = infra.scheduler || {};
    const infraTiles = [
      tile(
        "Scheduler lock",
        scheduler.leader_active ? "Leader active" : "No leader",
        `lock_id ${scheduler.lock_id || "—"}`,
        null,
        scheduler.leader_active ? "healthy" : "critical",
        "Scheduler leadership"
      ),
      tile("Infrastructure status", infra.status || "unknown", "Advisory lock probe", null, infra.status || "unknown", "Infrastructure health"),
    ];
    return renderGroup("Infrastructure", infraTiles);
  }

  function renderIncidentsPanel(data) {
    const incidents = data.incidents || [];
    if (!incidents.length) {
      return `<div class="empty-state"><p>No active incidents</p><p class="empty-hint">Healthy systems have no open critical or warning incidents.</p></div>`;
    }
    return `<div class="incident-list">${incidents
      .map(
        (item) => `<article class="incident-row incident-${item.severity || "warning"}">
<header><span class="${badgeClass(item.severity)}">${item.severity || "warning"}</span> <strong>${item.title || "Incident"}</strong></header>
<p>${item.detail || ""}</p>
<p class="incident-source">Source: ${item.source || "unknown"}</p>
</article>`
      )
      .join("")}</div>`;
  }

  function renderCriticalStrip(data) {
    const strip = $("monitoring-critical-strip");
    if (!strip) return;
    const incidents = (data.incidents || []).filter((item) => item.severity === "critical");
    if (!incidents.length) {
      strip.hidden = true;
      strip.innerHTML = "";
      return;
    }
    strip.hidden = false;
    strip.innerHTML = `<strong>Critical incidents</strong> ${incidents
      .map((item) => `<span class="critical-pill">${item.title}</span>`)
      .join("")}`;
  }

  function setLoading(loading, options) {
    const keepContent = options && options.keepContent;
    const panel = $("monitoring-panel");
    const skeleton = $("monitoring-skeleton");
    const content = $("monitoring-content");
    const refreshBtn = $("monitoring-refresh");
    if (panel) panel.setAttribute("aria-busy", loading ? "true" : "false");
    if (skeleton && !keepContent) skeleton.hidden = !loading;
    if (content && !keepContent) content.hidden = loading;
    if (refreshBtn) refreshBtn.setAttribute("aria-busy", loading ? "true" : "false");
  }

  function showError(msg) {
    const box = $("monitoring-error");
    const text = $("monitoring-error-text");
    if (box && text) {
      text.textContent = msg || "Failed to load monitoring data.";
      box.hidden = false;
    }
  }

  function markRefreshStale() {
    const el = $("monitoring-updated");
    if (!el) return;
    el.textContent = "Showing cached snapshot · refresh failed";
    el.classList.add("monitoring-stale");
  }

  function clearRefreshStale() {
    $("monitoring-updated")?.classList.remove("monitoring-stale");
  }

  async function readMonitoringResponse(res) {
    const contentType = res.headers.get("content-type") || "";
    if (res.status === 401) {
      throw new Error("Session expired — sign in again to refresh live metrics.");
    }
    if (!res.ok) {
      throw new Error(`Monitoring request failed (HTTP ${res.status}).`);
    }
    if (!contentType.includes("application/json")) {
      throw new Error("Monitoring returned an unexpected response. Hard-refresh the page or sign in again.");
    }
    return res.json();
  }

  async function fetchMonitoring(options) {
    const silent = options && options.silent;
    const keepContent = silent && hasLiveData;
    setLoading(true, { keepContent });
    hideError();
    try {
      const res = await fetch(MONITORING_URL, {
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      const data = await readMonitoringResponse(res);
      lastFetchAt = Date.now();
      applyMonitoringData(data);
      hasLiveData = true;
      clearRefreshStale();
      hideError();
    } catch (err) {
      if (hasLiveData || silent) {
        hideError();
        markRefreshStale();
      } else {
        showError(err.message || "Failed to load monitoring data.");
      }
    } finally {
      setLoading(false);
    }
  }

  function hideError() {
    const box = $("monitoring-error");
    if (box) box.hidden = true;
  }

  function updateTimestamp() {
    const el = $("monitoring-updated");
    if (!el || !lastFetchAt) return;
    const secs = Math.max(0, Math.floor((Date.now() - lastFetchAt) / 1000));
    el.textContent = `Updated ${secs}s ago`;
    el.classList.add("flash-success");
    window.setTimeout(() => el.classList.remove("flash-success"), 800);
  }

  function applyMonitoringData(data) {
    const panels = {
      overview: $("panel-overview"),
      ingest: $("panel-ingest"),
      queues: $("panel-queues"),
      data: $("panel-data"),
      infra: $("panel-infra"),
      integrations: $("panel-integrations"),
      incidents: $("panel-incidents"),
    };
    if (panels.overview) panels.overview.innerHTML = renderOverviewPanel(data);
    if (panels.ingest) panels.ingest.innerHTML = renderIngestPanel(data);
    if (panels.queues) panels.queues.innerHTML = renderQueuePanel(data);
    if (panels.data) panels.data.innerHTML = renderDataPanel(data);
    if (panels.infra) panels.infra.innerHTML = renderInfraPanel(data);
    if (panels.integrations) panels.integrations.innerHTML = renderIntegrationsPanel(data);
    if (panels.incidents) panels.incidents.innerHTML = renderIncidentsPanel(data);
    renderCriticalStrip(data);
    updateTimestamp();
  }

  function activateTab(tabName, focusTab) {
    if (!TABS.includes(tabName)) tabName = "overview";
    activeTab = tabName;
    TABS.forEach((name) => {
      const tab = $(`tab-${name}`);
      const panel = $(`panel-${name}`);
      const selected = name === tabName;
      if (tab) {
        tab.setAttribute("aria-selected", selected ? "true" : "false");
        tab.setAttribute("tabindex", selected ? "0" : "-1");
      }
      if (panel) {
        panel.hidden = !selected;
        panel.classList.toggle("active", selected);
      }
    });
    const url = new URL(window.location.href);
    url.searchParams.set("tab", tabName);
    window.history.replaceState(null, "", url.toString());
    if (focusTab) {
      $(`tab-${tabName}`)?.focus();
    }
  }

  function setupTabs() {
    const tabList = document.querySelector(".monitoring-tabs");
    if (!tabList) return;
    const requested = new URL(window.location.href).searchParams.get("tab");
    activateTab(requested || "overview", false);
    tabList.addEventListener("click", (event) => {
      const btn = event.target.closest("[data-tab]");
      if (!btn) return;
      activateTab(btn.getAttribute("data-tab"), true);
    });
    tabList.addEventListener("keydown", (event) => {
      const currentIndex = TABS.indexOf(activeTab);
      if (currentIndex < 0) return;
      let nextIndex = currentIndex;
      if (event.key === "ArrowRight") nextIndex = (currentIndex + 1) % TABS.length;
      if (event.key === "ArrowLeft") nextIndex = (currentIndex - 1 + TABS.length) % TABS.length;
      if (event.key === "Home") nextIndex = 0;
      if (event.key === "End") nextIndex = TABS.length - 1;
      if (nextIndex !== currentIndex) {
        event.preventDefault();
        activateTab(TABS[nextIndex], true);
      }
    });
  }

  function loadBootstrap() {
    const el = $("monitoring-bootstrap");
    if (!el || !el.textContent || el.textContent === "null") {
      return false;
    }
    try {
      const data = JSON.parse(el.textContent);
      lastFetchAt = Date.now();
      applyMonitoringData(data);
      hasLiveData = true;
      hideError();
      setLoading(false);
      return true;
    } catch {
      return false;
    }
  }

  function scheduleRefresh() {
    if (timer) window.clearInterval(timer);
    timer = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        fetchMonitoring({ silent: true });
      }
    }, REFRESH_MS);
  }

  function bindHostnamePreview() {
    const input = $("domain-hostname");
    const preview = $("staging-preview");
    if (!input || !preview) return;
    const update = () => {
      const host = (input.value || "your-domain.com").trim().toLowerCase() || "your-domain.com";
      preview.innerHTML = `Staging hostname will be <code>staging.${host}</code>`;
    };
    input.addEventListener("input", update);
    update();
  }

  document.addEventListener("DOMContentLoaded", () => {
    bindHostnamePreview();
    if (!$("monitoring")) return;
    setupTabs();
    $("monitoring-refresh")?.addEventListener("click", () => fetchMonitoring());
    $("monitoring-retry")?.addEventListener("click", () => fetchMonitoring());
    const booted = loadBootstrap();
    if (!booted) {
      fetchMonitoring();
    } else {
      hasLiveData = true;
      const content = $("monitoring-content");
      const skeleton = $("monitoring-skeleton");
      if (content) content.hidden = false;
      if (skeleton) skeleton.hidden = true;
      updateTimestamp();
    }
    scheduleRefresh();
    window.setInterval(updateTimestamp, 5000);
  });
})();
