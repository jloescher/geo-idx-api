(function () {
  const REFRESH_MS = 30000;
  const MONITORING_URL = "/dashboard/monitoring/data";
  const TABS = ["overview", "ingest", "queues", "data", "infra", "integrations", "incidents"];
  const QUEUE_PENDING_STALE = 500;
  const REPLICA_PENDING_STALE = 500;
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

  /** last_enqueue_at is MAX(created_at) on pending jobs only — completed rows are deleted. */
  function formatSchedulerEnqueue(scheduler) {
    if (scheduler.last_enqueue_at) return scheduler.last_enqueue_at;
    if (scheduler.leader_active) return "none pending (queue drains on success)";
    return "none in jobs table";
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
    if (status === "catching_up") return "badge badge-stale";
    if (status === "stale") return "badge badge-stale";
    return "badge badge-unknown";
  }

  function statusLabel(status) {
    if (status === "catching_up") return "catching up";
    return status || "unknown";
  }

  /** In-dashboard tab drill-down (not the domain-token API). */
  function monitoringTabHref(tab) {
    return `/dashboard/monitoring?tab=${encodeURIComponent(tab)}`;
  }

  function tile(label, value, sub, href, status, aria) {
    const tag = href ? "a" : "div";
    const hrefAttr = href ? ` href="${href}"` : "";
    const statusChip = status
      ? `<span class="${badgeClass(status)}">${statusLabel(status)}</span>`
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

  function renderStatList(title, rows, headers) {
    if (!rows.length) {
      return `<div class="metric-group"><h3 class="metric-group-title">${title}</h3><div class="empty-state"><p>No entries yet</p></div></div>`;
    }
    const head = headers.map((h) => `<th>${h}</th>`).join("");
    const body = rows.map((row) => `<tr>${row.map((col) => `<td>${col}</td>`).join("")}</tr>`).join("");
    return `<div class="metric-group"><h3 class="metric-group-title">${title}</h3><div class="monitoring-table-wrap"><table class="monitoring-table"><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table></div></div>`;
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
      tile(
        "Queue depth",
        fmtNum((queues.total_pending || 0) + (queues.total_reserved || 0) + (queues.total_scheduled || 0)),
        `${fmtNum(queues.total_reserved || 0)} reserved · ${fmtNum(queues.total_stale_reserved || 0)} stale`,
        null,
        queues.status || "unknown",
        "In-flight queue jobs"
      ),
      tile(
        "Cache hit rate",
        fmtPct(cache.hit_rate_pct),
        `${cache.window_minutes || 15}m window`,
        null,
        cache.status === "healthy" ? "healthy" : cache.status === "no_data" ? "unknown" : cache.status || "unknown",
        "Cache hit rate"
      ),
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
        monitoringTabHref("ingest"),
        row.status,
        `Listings ${row.dataset_slug}`
      );
    });
    const pipelineRows = data.sync_pipeline?.by_status || [];
    const replicationActive = (data.listings || []).some((l) => l.replication_in_progress);
    const pipelineChip = (row) => {
      if (row.status === "failed") return "critical";
      if (row.status === "pending" && (row.count || 0) > REPLICA_PENDING_STALE) return "stale";
      return "healthy";
    };
    const pipelineTiles = pipelineRows.length
      ? pipelineRows.map((row) =>
          tile(
            `${row.dataset_slug} · ${row.status}`,
            fmtNum(row.count),
            "replica_pages",
            null,
            pipelineChip(row),
            `Replica pages ${row.dataset_slug} ${row.status}`
          )
        )
      : [
          tile(
            "Replica pages",
            "0",
            replicationActive ? "Replication in progress" : "No active replica pages (sync idle)",
            null,
            replicationActive ? "stale" : "healthy",
            "Replica pipeline idle"
          ),
        ];
    const lagRows = (data.listings || []).map((row) => [
      row.dataset_slug || "—",
      row.lag_seconds !== null && row.lag_seconds !== undefined ? `${fmtNum(row.lag_seconds)}s` : "—",
      row.freshness_mode || "—",
      row.status || "unknown",
    ]);
    return (
      renderGroup("Listings Replication", listings) +
      renderGroup("Replica Pipeline", pipelineTiles) +
      renderStatList("Sync Lag by Dataset", lagRows, ["Dataset", "Lag", "Mode", "Status"])
    );
  }

  function renderGISOpsTable(sources, isAdmin) {
    const rows = (sources || []).map((src) => {
      const api = src.api_status || "unknown";
      const probe = src.last_probe_at ? fmtSyncAge(src.last_probe_at) : "—";
      const actions = isAdmin
        ? `<button type="button" class="btn btn-secondary btn-sm" data-gis-action="probe" data-source-key="${src.source_key}">Probe</button>
           <button type="button" class="btn btn-secondary btn-sm" data-gis-action="sync" data-source-key="${src.source_key}">Sync</button>`
        : "—";
      return `<tr>
        <td>${src.source_key || "—"}</td>
        <td>${src.county_slug || "—"}</td>
        <td>${src.sync_mode || "—"}</td>
        <td><span class="${badgeClass(api === "reachable" ? "healthy" : api === "unreachable" ? "critical" : "unknown")}">${api}</span></td>
        <td>${probe}</td>
        <td>${fmtNum(src.parcel_count)}</td>
        <td>${src.active_sync_job ? "yes" : "no"}</td>
        <td>${actions}</td>
      </tr>`;
    });
    const toolbar = isAdmin
      ? `<div class="monitoring-toolbar"><button type="button" class="btn btn-secondary btn-sm" data-gis-action="probe-all">Probe all</button>
         <span id="gis-ops-status" class="monitoring-meta" role="status"></span></div>`
      : "";
    return `${toolbar}<table class="monitoring-table"><thead><tr>
      <th>Source</th><th>County</th><th>Mode</th><th>API</th><th>Last probe</th><th>Parcels</th><th>Sync active</th><th>Actions</th>
    </tr></thead><tbody>${rows.join("") || '<tr><td colspan="8">No GIS sources in snapshot</td></tr>'}</tbody></table>`;
  }

  function renderDataPanel(data) {
    const gis = data.gis || {};
    const isAdmin = document.getElementById("monitoring")?.dataset?.isAdmin === "true";
    const gisTiles = [
      tile("Parcels", fmtNum(gis.parcels_total), fmtSyncAge(gis.parcels_last_synced_at), null, gis.parcels_status || "unknown", "GIS parcels"),
      tile("Cities", fmtNum(gis.cities_total), fmtSyncAge(gis.cities_last_synced_at), null, gis.cities_status || "unknown", "GIS cities"),
      tile("Counties", fmtNum(gis.counties_total), fmtSyncAge(gis.counties_last_synced_at), null, gis.counties_status || "unknown", "GIS counties"),
      tile("ZIPs", fmtNum(gis.zips_total), fmtSyncAge(gis.zips_last_synced_at), null, gis.zips_status || "unknown", "GIS zips"),
    ];
    const byCounty = Object.entries(gis.by_county || {})
      .sort((a, b) => Number(b[1]) - Number(a[1]))
      .slice(0, 8)
      .map(([county, total]) => [county, fmtNum(total)]);
    return (
      renderGroup("GIS Data Health", gisTiles) +
      renderStatList("Top Counties by Parcel Count", byCounty, ["County", "Parcels"]) +
      `<div class="monitoring-group"><h3 class="monitoring-group-title">GIS Sources</h3>${renderGISOpsTable(gis.sources, isAdmin)}</div>`
    );
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
    const ageRows = cryptoAssets
      .slice()
      .sort((a, b) => Number(b.age_seconds || 0) - Number(a.age_seconds || 0))
      .map((asset) => [
        (asset.asset_key || "").toUpperCase(),
        `${fmtNum(asset.age_seconds)}s`,
        asset.stale ? "stale" : "healthy",
      ]);
    return renderGroup("Integrations", cryptoTiles) + renderStatList("Asset Freshness", ageRows, ["Asset", "Age", "Status"]);
  }

  function renderQueuePanel(data) {
    const queues = data.queues || {};
    const queueRows = queues.by_queue || [];
    const staleAfterMin = Math.round((queues.stale_reserved_after_seconds || 600) / 60);
    const rollupTiles = [
      tile(
        "Ready",
        fmtNum(queues.total_pending),
        "Available now (not reserved)",
        null,
        (queues.total_pending || 0) > QUEUE_PENDING_STALE ? "stale" : "healthy",
        "Jobs ready to reserve"
      ),
      tile(
        "Reserved",
        fmtNum(queues.total_reserved),
        `${fmtNum(queues.total_stale_reserved || 0)} stale (>${staleAfterMin}m)`,
        null,
        (queues.total_stale_reserved || 0) > 0 ? "stale" : (queues.total_reserved || 0) > 0 ? "healthy" : "healthy",
        "Jobs claimed by workers"
      ),
      tile(
        "Scheduled",
        fmtNum(queues.total_scheduled),
        "Delayed (available_at in future)",
        null,
        (queues.total_scheduled || 0) > 0 ? "healthy" : "healthy",
        "Delayed jobs"
      ),
      tile(
        "Failed (7d)",
        fmtNum(queues.total_failed_recent ?? queues.total_failed),
        `${fmtNum(queues.total_failed || 0)} all-time in failed_jobs`,
        null,
        (queues.total_failed_recent || 0) > 0 ? "stale" : "healthy",
        "Recent failed jobs"
      ),
    ];
    const cache = data.cache || {};
    const cacheTiles = [
      tile(
        "Hit rate",
        fmtPct(cache.hit_rate_pct),
        `${cache.window_minutes || 15}m window`,
        null,
        cache.status === "healthy" ? "healthy" : cache.status === "no_data" ? "unknown" : cache.status || "unknown",
        "Cache hit rate"
      ),
      tile("Audit requests", fmtNum(cache.total), `${fmtNum(cache.hits)} hits`, null, null, "Cache audit volume"),
    ];
    const queueTiles = queueRows.map((q) => {
      const name = q.queue || "unknown";
      const pending = q.pending ?? 0;
      const scheduled = q.scheduled ?? 0;
      const reserved = q.reserved ?? 0;
      const staleReserved = q.stale_reserved ?? 0;
      const failed = q.failed ?? 0;
      const failedRecent = q.failed_recent ?? 0;
      const chip =
        staleReserved > 0 || failedRecent > 0
          ? "stale"
          : pending > QUEUE_PENDING_STALE
            ? "stale"
            : reserved > 0 || pending > 0 || scheduled > 0
              ? "healthy"
              : "healthy";
      return tile(
        name,
        fmtNum(pending + reserved + scheduled),
        `${fmtNum(pending)} ready · ${fmtNum(reserved)} reserved (${fmtNum(staleReserved)} stale) · ${fmtNum(scheduled)} scheduled`,
        monitoringTabHref("queues"),
        chip,
        `Queue ${name}`
      );
    });
    const allQueuesHealthy =
      queues.status === "healthy" &&
      (queues.total_pending || 0) === 0 &&
      (queues.total_failed_recent || 0) === 0 &&
      (queues.total_stale_reserved || 0) === 0;
    if (!queueTiles.length) {
      queueTiles.push(
        tile(
          "Queues",
          "0",
          allQueuesHealthy ? "No jobs in queue tables — workers idle" : "No queue rows",
          monitoringTabHref("queues"),
          allQueuesHealthy ? "healthy" : "unknown",
          "Queue depth"
        )
      );
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
    const inFlightRows = (queues.in_flight || []).map((row) => [
      row.job_id ?? "—",
      row.queue || "—",
      row.job_type || "unknown",
      row.state || "—",
      fmtNum(row.age_seconds) + "s",
      fmtNum(row.attempts),
      row.stale ? "stale" : "ok",
    ]);
    const batchRows = (queues.active_batches || []).map((row) => [
      row.name || row.batch_id || "—",
      `${fmtNum(row.pending_jobs)} / ${fmtNum(row.total_jobs)}`,
      fmtNum(row.failed_jobs),
      fmtNum(row.age_seconds) + "s",
    ]);
    const readyTypeRows = (queues.top_job_types || []).slice(0, 10).map((row) => [
      row.queue || "—",
      row.job_type || "unknown",
      fmtNum(row.count),
    ]);
    const reservedTypeRows = (queues.reserved_job_types || []).slice(0, 10).map((row) => [
      row.queue || "—",
      row.job_type || "unknown",
      fmtNum(row.count),
    ]);
    const inFlightSection = inFlightRows.length
      ? renderStatList("In-flight jobs", inFlightRows, ["ID", "Queue", "Type", "State", "Age", "Attempts", "Stale"])
      : `<div class="metric-group"><h3 class="metric-group-title">In-flight jobs</h3><div class="empty-state"><p>No rows in jobs</p><p class="empty-hint">Successful jobs are deleted immediately. Reserved and ready rows appear here while work is outstanding.</p></div></div>`;
    const batchSection = batchRows.length
      ? renderStatList("Active batches", batchRows, ["Batch", "Pending / total", "Failed", "Age"])
      : `<div class="metric-group"><h3 class="metric-group-title">Active batches</h3><div class="empty-state"><p>No open batches</p><p class="empty-hint">Multi-chunk persist batches (e.g. spark-replica-persist:beaches) show pending/total while workers drain chunks.</p></div></div>`;
    return (
      renderGroup("Queue Roll-up", rollupTiles) +
      renderGroup("Queue Health", queueTiles) +
      inFlightSection +
      batchSection +
      renderGroup("Queue Performance", cacheTiles) +
      renderGroup("Failed Jobs", failedTiles) +
      renderStatList("Ready job types", readyTypeRows, ["Queue", "Type", "Ready"]) +
      renderStatList("Reserved job types", reservedTypeRows, ["Queue", "Type", "Reserved"])
    );
  }

  function renderInfraPanel(data) {
    const infra = data.infrastructure || {};
    const scheduler = infra.scheduler || {};
    const infraTiles = [
      tile(
        "Scheduler lock",
        scheduler.leader_active ? "Leader active" : "No leader",
        `lock ${scheduler.lock_id || "—"} · pid ${scheduler.holder_pid ?? "—"} · backends ${scheduler.scheduler_backends ?? 0} · enqueue ${formatSchedulerEnqueue(scheduler)}`,
        null,
        scheduler.leader_active ? "healthy" : "critical",
        "Scheduler leadership"
      ),
      tile("Infrastructure status", infra.status || "unknown", "Advisory lock probe", null, infra.status || "unknown", "Infrastructure health"),
    ];
    const deps = [
      ["Scheduler lock", scheduler.leader_active ? "healthy" : "critical", scheduler.lock_id || "—"],
      ["Queue subsystem", data.queues?.status || "unknown", fmtNum(data.queues?.total_pending || 0)],
      ["Sync pipeline", data.sync_pipeline?.status || "unknown", fmtNum((data.sync_pipeline?.by_status || []).reduce((sum, r) => sum + (r.count || 0), 0))],
      ["GIS freshness", data.gis?.status || "unknown", `${fmtNum(data.gis?.parcels_total || 0)} parcels`],
    ];
    return renderGroup("Infrastructure", infraTiles) + renderStatList("Dependency Health", deps, ["Dependency", "Status", "Signal"]);
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

  function renderFirstLoadError(err) {
    const panel = $(`panel-${activeTab}`) || $("panel-overview");
    if (!panel) return;
    panel.innerHTML = `<div class="empty-state"><p>Monitoring data is unavailable</p><p class="empty-hint">${err || "Retry after re-authenticating or refreshing the page."}</p></div>`;
    $("monitoring-content")?.removeAttribute("hidden");
    $("monitoring-skeleton")?.setAttribute("hidden", "hidden");
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
        renderFirstLoadError(err.message);
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

  function bindTabDrillLinks() {
    const root = $("monitoring");
    if (!root) return;
    root.addEventListener("click", (event) => {
      const link = event.target.closest("a.metric-tile[href*='tab=']");
      if (!link) return;
      try {
        const url = new URL(link.href, window.location.origin);
        if (url.pathname !== "/dashboard/monitoring") return;
        const tab = url.searchParams.get("tab");
        if (!tab || !TABS.includes(tab)) return;
        event.preventDefault();
        activateTab(tab, false);
      } catch {
        /* ignore malformed href */
      }
    });
  }

  function setupTabs() {
    const tabList = document.querySelector(".monitoring-tabs");
    if (!tabList) return;
    if (!$("panel-overview")) {
      console.warn("monitoring: tab panels missing — hard-refresh to load latest dashboard.js");
    }
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
    } catch (err) {
      console.warn("monitoring bootstrap parse failed", err);
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

  async function gisAdminPost(path, body) {
    const res = await fetch(`/api/v1/admin/gis${path}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body || {}),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || res.statusText);
    return data;
  }

  function bindGISOps() {
    const panel = $("panel-data");
    if (!panel || document.getElementById("monitoring")?.dataset?.isAdmin !== "true") return;
    panel.addEventListener("click", async (event) => {
      const btn = event.target.closest("[data-gis-action]");
      if (!btn) return;
      const status = $("gis-ops-status");
      const action = btn.getAttribute("data-gis-action");
      const sourceKey = btn.getAttribute("data-source-key");
      try {
        btn.disabled = true;
        if (status) status.textContent = "Working…";
        if (action === "probe-all") {
          await gisAdminPost("/probe", {});
        } else if (action === "probe" && sourceKey) {
          await gisAdminPost("/probe", { source_key: sourceKey });
        } else if (action === "sync" && sourceKey) {
          const force = window.confirm(`Start parcel sync for ${sourceKey}?`);
          if (!force) return;
          await gisAdminPost("/sync", { source_key: sourceKey, force: true });
        }
        if (status) status.textContent = "Done";
        fetchMonitoring({ silent: true });
      } catch (err) {
        if (status) status.textContent = err.message || "Failed";
      } finally {
        btn.disabled = false;
      }
    });
  }

  document.addEventListener("DOMContentLoaded", () => {
    bindHostnamePreview();
    if (!$("monitoring")) return;
    setupTabs();
    bindTabDrillLinks();
    bindGISOps();
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
