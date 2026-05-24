# Admin dashboard

Invite-only Fiber dashboard with section navigation for domain onboarding, API keys, and live ops monitoring.

## Navigation

| Route | Section | Description |
|-------|---------|-------------|
| `/dashboard` | — | Redirects to `/dashboard/monitoring` |
| `/dashboard/monitoring` | Monitoring | Live ops metrics, 30s auto-refresh |
| `/dashboard/setup` | Setup | Add domain, TXT verify, domain list (merged flow) |
| `/dashboard/api-keys` | API keys | Token list with last-used timestamps |
| `/dashboard/invite` | Invite user | Admin only |

**Setup** combines the former “Add domain” and DNS verification flows: submitting the add-domain form provisions the bundle and scrolls to the verify panel with tokens and TXT records.

## Setup flow (add domain + verify)

1. Open **Setup** → fill hostname + MLS dataset checkboxes → **Add domain & verify DNS**
2. `POST /dashboard/domains` provisions in one PostgreSQL transaction:
   - Production domain (`example.com`)
   - Staging domain (`staging.example.com`)
   - **Production** and **Staging** API tokens (shown once in the verify panel)
   - `allowed_mls_datasets` JSON from checkbox selections; first checked feed becomes default `mls_dataset`
3. Redirect to `/dashboard/setup#verify` with a one-time session flash (tokens + TXT)
4. Publish TXT records at your DNS host
5. Click **Verify TXT** per hostname (`POST /dashboard/domains/:id/verify-txt`)

DNS TXT verification is required before API auth (`domains.verification_status`).

### MLS dataset checkboxes

Catalog codes come from `mls.Resolver.Catalog()` (e.g. `bridge_stellar`, `spark_beaches`). At least one dataset must be selected.

## Monitoring

| Endpoint | Auth |
|----------|------|
| `GET /dashboard/monitoring` | Dashboard session (HTML) |
| `GET /dashboard/monitoring/data` | Dashboard session (JSON snapshot) |
| `GET /api/v1/admin/monitoring` | Same session middleware (JSON) |

Refresh: manual **Refresh** button + 30s interval (pauses when tab hidden). Sessions persist in PostgreSQL (`dashboard_sessions`); re-login after migration or cookie invalidation.

### Metrics glossary

| Section | Fields | Notes |
|---------|--------|-------|
| **Listings** | total, active/pending, lag, freshness mode | Per `dataset_slug`; drill-down → `/api/v1/bridge/stats` |
| **GIS** | parcels, cities, counties, zips, source states, layer freshness | Stale if parcel/zip sync &gt;35d or generation mismatch |
| **Crypto** | BTC/ETH/SOL USD + age | Stale if snapshot &gt;1h |
| **Cache** | 15m hit rate from `mls_proxy_audit_logs` | |
| **Queues** | pending/reserved/failed by queue | PostgreSQL `jobs` / `failed_jobs` (not Asynq) |
| **Activation** | domains, keys, verified, 30d audit traffic | Traffic proxies “first API call” setup step |

### GIS freshness {#gis-freshness}

Parcels and ZIP tiles show a subtitle from `parcels_last_synced_at` / `zips_last_synced_at` (e.g. “Synced 2d ago” or “Never synced”) and an amber **stale** badge when the timestamp is older than 35 days. Parcel source rows also mark stale when stored generation does not match `gis_source_states.generation`.

## UI state matrix

| State | Monitoring | Setup (domains + verify) | API keys |
|-------|------------|--------------------------|----------|
| Loading | Skeleton tiles | — | — |
| Empty | Zeros + “No data yet” | Empty state + add-domain form | Empty state + link to Setup |
| Error | Alert + Retry | `.form-error` / `verify_error` query | — |
| Success | Timestamp flash | Verify panel after provision; `verified=1` banner | — |
| Stale | Amber badge | Pending badge on domain rows | — |

## Legacy endpoints

Unchanged for backward compatibility:

- `POST /dashboard/api-tokens` — manual token create
- `POST /dashboard/api-tokens/staging` — no longer blocks duplicate “Staging” names
- `POST /dashboard/domains/:id/verify-txt` — verifies DNS; mints Production token only if none exists
