(function () {
  const REFRESH_MS = 30000;
  const MONITORING_URL = "/dashboard/monitoring/data";
  let lastFetchAt = 0;
  let timer = null;
  let hasLiveData = false;

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

  function badgeClass(status) {
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

  function renderMonitoring(data) {
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

    const gis = data.gis || {};
    const gisTiles = [
      tile("Parcels", fmtNum(gis.parcels_total), "PostGIS mirror", null, gis.status, "GIS parcels"),
      tile("Cities", fmtNum(gis.cities_total), "Boundaries", null, gis.status, "GIS cities"),
      tile("Counties", fmtNum(gis.counties_total), "Boundaries", null, gis.status, "GIS counties"),
      tile("ZIPs", fmtNum(gis.zips_total), "Boundaries", null, gis.status, "GIS zips"),
    ];

    const cryptoAssets = (data.crypto && data.crypto.assets) || [];
    const cryptoTiles = cryptoAssets.length
      ? cryptoAssets.map((a) =>
          tile(
            a.asset_key.toUpperCase(),
            `$${Number(a.price_usd).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
            `${a.age_seconds}s ago`,
            null,
            a.stale ? "stale" : "healthy",
            `${a.asset_key} price`
          )
        )
      : [tile("Crypto", "—", "No snapshot", null, data.crypto?.status || "unknown", "Crypto prices")];

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

    const act = data.activation || {};
    const activationTiles = [
      tile("Domains", fmtNum(act.domain_count), `${fmtNum(act.verified_domain_count)} verified`, "/dashboard/setup", null, "Domains"),
      tile("API keys", fmtNum(act.token_count), "All users", "/dashboard/api-keys", null, "API keys"),
      tile("Traffic 30d", fmtNum(act.domains_with_traffic_30d), "Domains with audit calls", "/dashboard/setup", null, "Activation traffic"),
    ];

    return (
      renderGroup("Listings", listings) +
      renderGroup("GIS Data", gisTiles) +
      renderGroup("Crypto", cryptoTiles) +
      renderGroup("Cache", cacheTiles) +
      renderGroup("Queues", queueTiles) +
      renderGroup("Activation", activationTiles)
    );
  }

  function renderQueueDetail(data) {
    const lines = (data.queues?.top_job_types || [])
      .map((row) => {
        const queue = row.queue || "—";
        const jobType = row.job_type || "unknown";
        const count = row.count ?? 0;
        return `${queue}\t${jobType}\t${count}`;
      })
      .filter((line) => line && !line.startsWith("undefined"));
    const el = $("monitoring-queue-detail");
    const details = $("monitoring-queues");
    if (!el || !details) return;
    if (lines.length) {
      el.textContent = lines.join("\n");
      details.hidden = false;
    } else {
      details.hidden = true;
    }
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
    const content = $("monitoring-content");
    if (content) {
      content.innerHTML = renderMonitoring(data);
    }
    renderQueueDetail(data);
    updateTimestamp();
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
