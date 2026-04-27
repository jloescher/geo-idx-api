import {
  createWidgetRoot,
  debounce,
  fetchJson,
  getRuntime,
  renderComplianceFooter,
} from "./runtime";
import { renderMapListingsWidget } from "./mapListingsWidget";

const DISCLOSURE = "Some IDX listings have been excluded from this website.";

export async function renderSearchWidget(target) {
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
    const q = input.value.trim();
    if (q.length < 2) {
      results.innerHTML = "";
      return;
    }
    results.innerHTML = "<p>Searching...</p>";
    try {
      const runtime = getRuntime();
      const payload = await fetchJson(`/widget/config/${encodeURIComponent(runtime.token)}`);
      results.innerHTML = `<p>Search ready for location ${payload.location_id}. Listings are served through authenticated API proxy routes.</p>`;
    } catch (_error) {
      results.innerHTML = "<p>Unable to search right now.</p>";
    }
  }, 300);

  input.addEventListener("input", runSearch);
}

export async function renderMapWidget(target) {
  await renderMapListingsWidget(target);
}

export async function renderCommunityWidget(target) {
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

export async function renderPropertyWidget(target) {
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
          mls_listing_id: "sample-listing",
        }),
      });
      status.textContent = "Lead captured. OTP flow initiated.";
    } catch (_error) {
      status.textContent = "Lead capture failed.";
    }
  });
}

export async function renderFooterWidget(target) {
  const root = createWidgetRoot(target, "footer");
  renderComplianceFooter(root);
}
