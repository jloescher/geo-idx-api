# Admin dashboard

Invite-only Fiber dashboard with section navigation for domains (onboarding, API keys, DNS verify) and live ops monitoring.

## Navigation

| Route | Section | Description |
|-------|---------|-------------|
| `/dashboard` | — | Redirects to `/dashboard/monitoring` |
| `/dashboard/monitoring` | Monitoring | Live ops metrics, 30s auto-refresh |
| `/dashboard/domains` | Domains | Add domain bundles, per-hostname API keys, TXT verify, delete |
| `/dashboard/invite` | Invite user | Admin only |

Legacy URLs redirect to Domains (query string preserved):

- `GET /dashboard/setup` → `/dashboard/domains`
- `GET /dashboard/api-keys` → `/dashboard/domains`

## Domains flow (add domain + verify)

1. Open **Domains** → fill hostname + MLS dataset checkboxes → **Add domain & verify DNS**
2. `POST /dashboard/domains` provisions in one PostgreSQL transaction:
   - Production domain (`example.com`)
   - Staging domain (`staging.example.com`, `parent_domain_id` → production)
   - One **Production** API token (`domain_id` → production row)
   - One **Staging** API token (`domain_id` → staging row)
   - `allowed_mls_datasets` JSON from checkbox selections; first checked feed becomes default `mls_dataset`
3. Redirect to `/dashboard/domains#verify` with a one-time session flash (tokens + TXT)
4. Publish TXT records at your DNS host
5. Click **Verify TXT** per hostname (`POST /dashboard/domains/:id/verify-txt`)

### API keys on the Domains page

- Each hostname shows a masked key (`idx_••••…`). **Show** / **Copy** work only when plaintext is in the current page flash (after provision or **Regenerate API key**).
- `POST /dashboard/domains/:id/regenerate-token` replaces the key for that domain (old key stops working immediately).
- `POST /dashboard/domains/:id/delete` removes the production domain; CASCADE deletes staging and both tokens.

DNS TXT verification is required before API auth (`domains.verification_status`). Bearer tokens with `domain_id` set must be used with `X-Domain-Slug` matching that domain.

### MLS dataset checkboxes

Catalog codes come from `mls.Resolver.Catalog()` (e.g. `bridge_stellar`, `spark_beaches`). At least one dataset must be selected.

## Monitoring

| Endpoint | Auth |
|----------|------|
| `GET /dashboard/monitoring` | Dashboard session (HTML) |
| `GET /dashboard/monitoring/data` | Dashboard session (JSON snapshot) |
| `GET /api/v1/admin/monitoring` | Same session middleware (JSON) |
| `POST /api/v1/admin/flood-enrich` | Enqueue FEMA NFHL flood enrichment ([fema-flood-enrichment.md](fema-flood-enrichment.md)) |

Refresh: manual **Refresh** button + 30s interval (pauses when tab hidden). Sessions persist in PostgreSQL (`dashboard_sessions`); re-login after migration or cookie invalidation.

### Monitoring tabs

- **Overview**: system rollup, queue pressure, cache efficiency, activation counters.
- **Ingest & Sync**: listing freshness + lag by dataset and `replica_pages` pipeline state.
- **Queue & Jobs**: queue depth, stale reservations (>10m), failed job hotspots.
- **Data Quality**: GIS layer freshness (parcels, cities, counties, zips) and source status.
- **Infrastructure**: scheduler advisory-lock leadership probe and infra health.
- **Integrations**: crypto/dependency freshness.
- **Incidents**: active warning/critical incidents generated from snapshot health checks.

### Metrics glossary

| Section | Fields | Notes |
|---------|--------|-------|
| **Listings** | total, active/pending, lag, freshness mode | Per `dataset_slug`; drill-down → `/api/v1/bridge/stats` |
| **GIS** | parcels, cities, counties, zips, source states, layer freshness | Stale if parcel/zip sync &gt;35d or generation mismatch |
| **Crypto** | BTC/ETH/SOL USD + age | Stale if snapshot &gt;1h |
| **Cache** | 15m hit rate from `mls_proxy_audit_logs` | |
| **Queues** | pending/reserved/failed by queue + stale_reserved | `stale_reserved` means reserved &gt;10m |
| **Queue failures** | top failed job types + latest exception preview | Grouped from `failed_jobs` |
| **Sync pipeline** | `replica_pages` counts by dataset/status | Flags stale on failed rows or large pending backlog |
| **Infrastructure** | scheduler advisory lock probe (`SCHEDULER_LEADER_LOCK_ID`) | Critical when no leader holds the lock |
| **Activation** | domains, keys, verified, 30d audit traffic | Traffic proxies “first API call” setup step |

### GIS freshness {#gis-freshness}

Parcels and ZIP tiles show a subtitle from `parcels_last_synced_at` / `zips_last_synced_at` (e.g. “Synced 2d ago” or “Never synced”) and an amber **stale** badge when the timestamp is older than 35 days. Parcel source rows also mark stale when stored generation does not match `gis_source_states.generation`.

County parcel sources, FDOR/FDOT upstream issues, and MLS coverage: [GIS sources](gis-sources.md).

## UI state matrix

| State | Monitoring | Domains |
|-------|------------|---------|
| Loading | Skeleton tiles while preserving previous tab content on refresh | — |
| Empty | Per-tab empty card when no records match available metrics | Empty state + add-domain form |
| Error | Alert + Retry; stale cached snapshot remains visible | `.form-error`, `verify_error`, `error` query |
| Success | Timestamp flash + tab-local render | Verify panel after provision; `verified=1` / `deleted=1` banners |
| Stale | Amber status chips + critical strip for cross-tab incidents | Pending badge on domain rows |

## Deprecated dashboard token endpoints

Manual token creation is disabled; use per-domain **Regenerate** on Domains:

- `POST /dashboard/api-tokens` → redirect with error
- `POST /dashboard/api-tokens/staging` → redirect with error
- `POST /dashboard/api-tokens/:id/revoke` → still supported (redirects to Domains)

## Schema (domain-scoped tokens)

Migration `00002_domain_token_scoping.sql` adds:

- `domains.parent_domain_id` (staging → production, `ON DELETE CASCADE`)
- `personal_access_tokens.domain_id` (`UNIQUE`, `ON DELETE CASCADE`)
