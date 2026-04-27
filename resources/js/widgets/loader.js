import { configureRuntime, renderBlockedMessage } from "./runtime";
import {
  renderCommunityWidget,
  renderFooterWidget,
  renderMapWidget,
  renderPropertyWidget,
  renderSearchWidget,
} from "./widgets";

const HANDLERS = {
  search: renderSearchWidget,
  map: renderMapWidget,
  community: renderCommunityWidget,
  property: renderPropertyWidget,
  footer: renderFooterWidget,
};

async function validateDomain(apiBase, token, requireFooter) {
  const response = await fetch(`${apiBase}/api/widgets/validate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      token,
      hostname: window.location.hostname,
      referrer: document.referrer || null,
      requireFooter,
    }),
  });

  const data = await response.json().catch(() => ({}));
  if (!response.ok || !data.ok) {
    throw new Error(data.message || "Widget host validation failed.");
  }

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
  const apiBase = `${src.protocol}//${src.host}`;
  const requireFooter = (scriptEl.dataset.footerRequired || "true") !== "false";

  return {
    token,
    apiBase,
    requireFooter,
    primaryColor,
    secondaryColor,
    accentColor,
    textColor,
    backgroundColor,
    listingsPerPage: 20,
  };
}

function normalizeHex(value, fallback) {
  if (!value || typeof value !== "string") {
    return fallback;
  }
  let v = value.trim();
  if (!v.startsWith("#")) {
    v = `#${v}`;
  }
  if (/^#[0-9a-fA-F]{3}$/.test(v)) {
    v = `#${v[1]}${v[1]}${v[2]}${v[2]}${v[3]}${v[3]}`;
  }

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
    locationId: api.location_id || loaderCfg.locationId,
  };
}

function applyQueryColorOverrides(cfg, scriptEl) {
  const src = new URL(scriptEl.src, window.location.origin);
  const pick = (param, cur) => {
    const raw = src.searchParams.get(param);
    if (raw === null || raw === "") {
      return cur;
    }

    return normalizeHex(raw, cur);
  };

  return {
    ...cfg,
    primaryColor: pick("primaryColor", cfg.primaryColor),
    secondaryColor: pick("secondaryColor", cfg.secondaryColor),
    accentColor: pick("accentColor", cfg.accentColor),
    textColor: pick("textColor", cfg.textColor),
    backgroundColor: pick("backgroundColor", cfg.backgroundColor),
  };
}

async function fetchWidgetRuntimeConfig(apiBase, token) {
  const res = await fetch(`${apiBase}/widget/config/${encodeURIComponent(token)}`, { credentials: "omit" });
  if (!res.ok) {
    throw new Error("Unable to load widget theme configuration.");
  }

  return res.json();
}

function hasFooterAnchor() {
  return !!document.querySelector('[data-quantyragidx-footer="true"]');
}

function ensureTailwindCdn() {
  const id = "qgx-tailwind-cdn";
  if (document.getElementById(id)) {
    return;
  }
  const script = document.createElement("script");
  script.id = id;
  script.src = "https://cdn.tailwindcss.com";
  script.defer = true;
  document.head.appendChild(script);
}

async function initWidget(type, options = {}) {
  const runtimeTarget = options.target || document.querySelector(`[data-quantyra-widget="${type}"]`);
  if (!runtimeTarget) {
    throw new Error(`No mount target found for widget type "${type}"`);
  }
  const handler = HANDLERS[type];
  if (!handler) {
    throw new Error(`Unsupported widget type "${type}"`);
  }
  await handler(runtimeTarget, options);
}

async function boot() {
  const scriptEl = document.currentScript || document.querySelector('script[src*="/widget/loader.js"]');
  if (!scriptEl) {
    return;
  }

  const config = parseLoaderConfig(scriptEl);
  if (!config.token) {
    window.QuantyraGeoIDX = {
      async initWidget() {
        throw new Error("Missing subscriber token.");
      },
      getValidation: () => null,
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
    getValidation: () => validationForGetter,
  };

  let validation;
  try {
    validation = await validateDomain(config.apiBase, config.token, config.requireFooter);
    validationForGetter = validation;
    const server = await fetchWidgetRuntimeConfig(config.apiBase, config.token);
    let merged = mergeServerWidgetConfig(config, server);
    merged = applyQueryColorOverrides(merged, scriptEl);
    configureRuntime({ ...merged, validation });
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

void boot();
