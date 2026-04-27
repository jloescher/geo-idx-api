const runtimeState = {
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
  mapAssetsLoaded: false,
};

export function configureRuntime(input) {
  Object.assign(runtimeState, input);
}

export function getRuntime() {
  return runtimeState;
}

export function createWidgetRoot(target, kind) {
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

export function renderComplianceFooter(container) {
  const validation = runtimeState.validation || {};
  const branding = validation.branding || {};
  const nowIso = new Date().toISOString();
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

export function renderBlockedMessage(target, message) {
  target.innerHTML = `<div style="border:2px solid #ef4444;background:#fef2f2;color:#7f1d1d;padding:16px;font-family:Inter,Arial,sans-serif;">
    <strong>GeoIDX compliance block</strong><div style="margin-top:8px;">${message}</div>
  </div>`;
}

export function debounce(fn, waitMs) {
  let handle = null;
  return (...args) => {
    window.clearTimeout(handle);
    handle = window.setTimeout(() => fn(...args), waitMs);
  };
}

export async function fetchJson(path, options = {}) {
  const { headers: extraHeaders, ...rest } = options;
  const response = await fetch(`${runtimeState.apiBase}${path}`, {
    ...rest,
    headers: {
      Accept: "application/json",
      ...(extraHeaders || {}),
    },
  });
  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}));
    const msg = errBody.message || `Widget request failed with status ${response.status}`;
    throw new Error(msg);
  }
  return response.json();
}

export async function ensureLeafletAssets() {
  if (runtimeState.mapAssetsLoaded) {
    return;
  }

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
    const existing = document.getElementById(jsId);
    if (existing) {
      resolve();
      return;
    }
    const script = document.createElement("script");
    script.id = jsId;
    script.src = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js";
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("Failed to load Leaflet assets"));
    document.head.appendChild(script);
  });

  runtimeState.mapAssetsLoaded = true;
}
