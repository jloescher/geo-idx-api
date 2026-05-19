# Spark / Beaches MLS — Listing display compliance

Quantyra-facing guide for displaying **BeachesMLS** listing data sourced through the Spark Platform. This document summarizes Spark’s [Listing Data Display Rules](https://sparkplatform.com/docs/supporting_documentation/compliance) and maps obligations to idx-api, geo-web, and widget surfaces.

**Scope:** `spark_beaches` catalog feed (Postgres mirror partition `beaches`). Stellar MLS (Bridge) has separate compliance requirements.

---

## 1. Purpose and audience

| Audience | Use this doc to… |
|----------|------------------|
| **idx-api maintainers** | Route replication vs live hosts correctly; passthrough compliance fields in JSON |
| **geo-web / widget engineers** | Render required fields, disclaimers, and IDX logos on consumer UI |
| **Operators** | Respond to MLS audits; verify domain allowlists and feed entitlements |

---

## 2. Upstream references

| Resource | URL / path |
|----------|------------|
| Spark Listing Data Display Rules | https://sparkplatform.com/docs/supporting_documentation/compliance |
| Spark Listings API | https://sparkplatform.com/docs/api_services/listings |
| Spark API replication (host rules) | https://sparkplatform.com/docs/supporting_documentation/replication |
| Quantyra Spark integration | [idx-api-integration.md](idx-api-integration.md) |
| Platform overview | [platform-overview.md](platform-overview.md) |
| Beaches RESO metadata | [beaches_metadata.xml](beaches_metadata.xml) |
| Sample replication page | [beaches_50_listings.json](beaches_50_listings.json) |
| MLS Data License (on file) | [MLS Data License.txt](MLS%20Data%20License.txt) |
| Data Access Agreement (on file) | [Data Access Agreement.txt](Data%20Access%20Agreement.txt) |

---

## 3. API hosts (replication vs live)

Spark issues **replication-capable** API keys that **must not** call the production API host for bulk sync. idx-api enforces:

| Traffic | Host | Example RESO base |
|---------|------|-------------------|
| **Replication / sync** (workers only) | `replication.sparkapi.com` | `https://replication.sparkapi.com/Reso/OData` |
| **Live IDX** (proxy, closed search, photos) | `sparkapi.com` | `https://sparkapi.com/v1/Reso/OData` |

Replication keys that call `sparkapi.com` for bulk replication will fail per [Spark replication docs](https://sparkplatform.com/docs/supporting_documentation/replication).

**idx-api paths:**

| Path | Host |
|------|------|
| `SparkSyncJob` / `SparkSyncFetchPageJob` | Replication |
| `/api/v1/*` RESO proxy when `?dataset=spark_beaches` | Live (`sparkapi.com`) |
| Hybrid search live fallback / closed listings | Live |
| Image proxy (`/images/{listingKey}/{photoId}`) for Spark feed | Live (Property `$expand=Media`) |

`@odata.nextLink` values from replication responses are absolute URLs on the replication host — sync must follow them as-is (no rewrite to `sparkapi.com`).

---

## 4. Listing views (Summary vs Detail)

Spark defines two display contexts. The listing payload includes `DisplayCompliance.View` indicating which rule set applies.

### Summary view

Use for **brief** listing presentation (search results, map cards, grids).

- Spark Listings API: any listing returned from the **listings search** service.
- RESO OData: Property collection queries with `$top` greater than 1 (typical search).

### Detail view

Use for **full** listing reports (single listing page).

- Spark Listings API: search with `_limit=1`, or direct **individual listing** GET.
- RESO OData: single Property entity GET (`Property('ListingKey')`).

When in doubt, read `DisplayCompliance.View` on each listing (`Summary` or `Detail`) and apply the matching System Info field list (section 5).

---

## 5. MLS-wide rules (System Info)

The Spark **System Info** service returns `DisplayCompliance` keyed by MLS identifier. For each MLS (Beaches), it defines:

| Attribute | Meaning |
|-----------|---------|
| `View.Summary.DisplayCompliance` | Array of **StandardField** names that **must** appear on summary views |
| `View.Detail.DisplayCompliance` | Array of **StandardField** names that **must** appear on detail views |
| `DisclaimerText` | HTML disclaimer text to show with listings (summary and detail) |
| `DisclaimerTextOnly` | Plain-text disclaimer when HTML is not supported |

Example shape (from Spark docs; field names vary by MLS):

```json
{
  "DisplayCompliance": {
    "20000426143505724628000000": {
      "View": {
        "Summary": {
          "DisplayCompliance": ["ListOfficeName", "ListingUpdateTimestamp"]
        },
        "Detail": {
          "DisplayCompliance": [
            "ListOfficeName",
            "ListOfficePhone",
            "ListOfficeEmail",
            "ListingUpdateTimestamp"
          ]
        }
      },
      "DisclaimerText": "Information is deemed to be reliable, but is not guaranteed.",
      "DisclaimerTextOnly": "Information is deemed to be reliable, but is not guaranteed."
    }
  }
}
```

**Implementation note:** Required field names are **MLS-specific**. Do not hardcode Beaches field lists in idx-api. Consumer apps should load System Info from the **live** Spark host and cache with a TTL (recommended v2 enhancement). Until cached, treat all proxied listing JSON as potentially incomplete for compliance display.

---

## 6. Listing-specific rules (IDX role)

For listings retrieved under the **IDX** API key role, each listing includes its own `DisplayCompliance` object:

| Attribute | Description |
|-----------|-------------|
| `View` | `Summary` or `Detail` — which System Info `View.*` list applies |
| `IDXLogo` | Full-size IDX logo (`LogoUri`, `Type` = `Uri` or `Text`) |
| `IDXLogoSmall` | Small IDX logo (`LogoUri`, `Type` = `Uri` or `Text`) |

When `Type` is `Uri`, render `LogoUri` as an image. When `Type` is `Text`, render `LogoUri` as text.

Example (from Spark docs):

```json
{
  "DisplayCompliance": {
    "IDXLogoSmall": {
      "LogoUri": "http://somesite.com/logo/small.jpg",
      "Type": "Uri"
    },
    "View": "Detail",
    "IDXLogo": {
      "LogoUri": "Acme Realty",
      "Type": "Text"
    }
  }
}
```

idx-api **must not strip** `DisplayCompliance` (or disclaimer fields) from proxied JSON responses.

---

## 7. Agent and office attribution

Spark recommends displaying list/buyer agent and office data from the **Accounts** resource, not only denormalized fields on the listing record. Account changes do not update listing `ModificationTimestamp`.

Replication best practice ([Spark replication](https://sparkplatform.com/docs/supporting_documentation/replication)):

1. Replicate and refresh **Accounts** alongside listings.
2. Resolve `ListAgentId`, `ListOfficeId`, and related IDs to current account records at display time.

**Current idx-api scope:** mirror stores listing `raw_data` including agent/office fields from replication; full Accounts replication is **not** implemented in v1. Detail pages that require freshest agent/office attribution should prefer **live** Property (and future Accounts) calls on `sparkapi.com`.

---

## 8. Quantyra platform responsibilities

### idx-api

- Keep `SPARK_ACCESS_TOKEN` server-side only.
- Enforce domain and token MLS allowlists (`spark_beaches` / `beaches`).
- Log proxied requests in `mls_proxy_audit_logs` where enabled.
- Use **replication host only** in queue workers for sync jobs.
- Use **live host** for user-facing RESO proxy, hybrid live search, and image resolution.
- Passthrough upstream JSON including `DisplayCompliance` without filtering.

### Postgres mirror (`listings.raw_data`)

- Optimized for Active/Pending search and map workloads.
- Compliance metadata in mirror rows may be **stale** relative to live Spark.
- Closed / off-market and compliance-sensitive detail should use **live** RESO when freshness matters.

### geo-web / widgets

- Render System Info required fields for the active view (Summary vs Detail).
- Show `DisclaimerText` or `DisclaimerTextOnly` with listing UI.
- Render IDX logos per listing `DisplayCompliance` when role is IDX.
- Respect existing MLS footer / compliance blocks in the widget loader.

### idx-images

- Photo proxy does not replace listing attribution, disclaimers, or office display requirements on surrounding pages.

---

## 9. Engineering checklist

- [ ] **Search / grid UI** — Summary view: show all `View.Summary.DisplayCompliance` fields from System Info for the active view.
- [ ] **Listing detail UI** — Detail view: show all `View.Detail.DisplayCompliance` fields.
- [ ] **Disclaimers** — Display `DisclaimerText` or `DisclaimerTextOnly` on pages that show MLS data.
- [ ] **IDX logos** — Honor per-listing `IDXLogo` / `IDXLogoSmall` when present (IDX role).
- [ ] **Hosts** — Never call `replication.sparkapi.com` from user-facing request handlers.
- [ ] **JSON passthrough** — Do not remove `DisplayCompliance` in proxy transforms.
- [ ] **System Info cache** (v2) — Fetch/cache `DisplayCompliance` from live System Info per feed.
- [ ] **Accounts replication** (v2) — Replicate Accounts for agent/office display per Spark guidance.
- [ ] **Audits** — Respond to MLS compliance inquiries within 72 hours per [Data Access Agreement](Data%20Access%20Agreement.txt).

---

## 10. Operational risks

| Risk | Mitigation |
|------|------------|
| Live RESO on `sparkapi.com` returns 401/403 for Beaches IDX key | Confirm with Spark that RESO is enabled on live host; try `SPARK_LIVE_RESO_ROOT=Version/3/Reso/OData` |
| Replication key used for live proxy | Use separate keys or ensure key works on both hosts per Spark dashboard |
| Stale agent phone/email on mirror-only pages | Prefer live Property GET for detail; plan Accounts replication |
| Missing disclaimer on widget | Wire System Info + listing `DisplayCompliance` in geo-web templates |

---

## 11. Related Spark Listings API notes

The [Listings service](https://sparkplatform.com/docs/api_services/listings) documents:

- IDX / Public / VOW / Portal roles (read-only for listing display).
- Expansions for Photos, OpenHouses, Documents, etc.
- Explicit pointer to this compliance model for all listing display.

For RESO-shaped clients, idx-api continues to expose OData Property/Member/Office routes; compliance rules apply equally to how **consumers** render the data, regardless of wire format.
