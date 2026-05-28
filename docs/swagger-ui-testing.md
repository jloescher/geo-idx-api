# Swagger UI — testing guide

Manual and smoke-test directions for the interactive API explorer served by the Go API at **`/swagger`**, backed by the embedded OpenAPI 3.1 document at **`/openapi.json`**.

## Prerequisites

| Requirement | Notes |
|-------------|--------|
| Running API | `make run-api` or deployed `idx-api-web` (default port **8000**) |
| Network (browser) | Swagger UI loads CSS/JS from **unpkg.com** (`swagger-ui-bundle.js` + `swagger-ui-standalone-preset.js`); offline or blocked CDN = blank page or “No layout defined for StandaloneLayout” |
| Verified domain | Active row in `domains` with TXT verification (or local seed domain) |
| API token (most routes) | Dashboard-issued PAT with `idx:access` or `idx:full`, or `POST /api/auth/token` for `idx:full` |
| PostGIS + migrations | GIS autocomplete needs Goose **00007** (`pg_trgm`) and populated `gis_cities` / `gis_counties` |

**Production database:** If your workspace `.env` points at production Patroni, prefer **read-only** checks (GET autocomplete, GET health). Do not run admin mutate routes or destructive SQL unless explicitly intended.

## OpenAPI source and embed

| Artifact | Location |
|----------|----------|
| Human-edited source | [`docs/yaak-api-collection.json`](yaak-api-collection.json) |
| Embedded copy (API binary) | `internal/openapi/spec/openapi.json` |
| Sync command | `make openapi-sync` (also runs before `make build`) |

After editing the Yaak/OpenAPI file:

```bash
make openapi-sync
make run-api   # or redeploy idx-api-web
```

Verify the live spec includes your change:

```bash
curl -sS http://localhost:8000/openapi.json | jq -r '.paths | keys[]' | grep autocomplete
```

Expected paths (among others):

- `/api/v1/gis/autocomplete/cities`
- `/api/v1/gis/autocomplete/counties`

## Open the explorer

| URL | Auth | Purpose |
|-----|------|---------|
| `http://localhost:8000/swagger` | None | Swagger UI (browser) |
| `http://localhost:8000/openapi.json` | None | Raw OpenAPI JSON |
| `http://localhost:8000/swagger/` | None | Redirects to `/swagger` |

Production example: `https://idx.quantyralabs.cc/swagger` (replace host with `APP_URL` / `IDX_API_PUBLIC_URL`).

**Verify assets loaded:** In browser DevTools → Network, confirm **200** for:

- `swagger-ui.css`
- `swagger-ui-bundle.js`
- `swagger-ui-standalone-preset.js` (required for `StandaloneLayout`)

**Server dropdown:** The spec lists **Development** and **Local** server URLs. Pick the host that matches where you are testing before **Execute**.

## Authorize protected routes

Most `/api/v1/*` operations use **`bearerAuth`** in the spec.

1. Click **Authorize** (lock icon).
2. Enter the PAT **without** the `Bearer ` prefix (Swagger adds it):  
   `abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890`
3. Click **Authorize**, then **Close**.

**Domain context** (required for token mode):

- Add header **`X-Domain-Slug`**: `your-verified-domain.com`, **or**
- Add query parameter **`domain`**: `your-verified-domain.com`

In Swagger UI, use each operation’s **Parameters** section for `domain`, or use **Try it out** → add a custom header if your browser extension allows; otherwise prefer `?domain=` on the request (documented on GIS autocomplete operations).

**Dataset selection** (MLS routes):

- Use query **`dataset`**: `stellar` or `beaches` on `/api/v1/listings`, search, RESO paths, etc.
- On **Bridge web** proxy routes (`/api/v1/listings`, agents, offices, openhouses), `dataset` is **IDX routing only** and must **not** be sent to upstream OData (the API strips it). If Bridge returns `Cannot find property 'dataset'`, remove `dataset` from the query string for web routes.

## Recommended test sequence

### 1. Infrastructure (no auth)

| Step | Operation in UI | Expected |
|------|-----------------|----------|
| 1a | `GET /healthz` | `200`, `{"status":"ok"}` |
| 1b | `GET /readyz` | `200`, `"ready": true`, PostGIS version string |
| 1c | `GET /openapi.json` | `200`, `Content-Type: application/json`, `"openapi": "3.1.0"` |

`GET /healthz` and `GET /readyz` are registered in code but may not appear in the Yaak/OpenAPI file; use curl for those if missing from the UI.

```bash
curl -sS http://localhost:8000/healthz
curl -sS http://localhost:8000/readyz
```

### 2. Issue a token (optional)

| Step | Operation | Body | Expected |
|------|-----------|------|----------|
| 2a | `POST /api/auth/token` | `{"email":"…","password":"…"}` from `ADMIN_SEED_*` (local seed) | `200`, `token`, `abilities` includes `idx:full` |

Use the returned token in **Authorize** for subsequent steps.

### 3. GIS autocomplete (new in OpenAPI)

| Step | Operation | Parameters | Expected |
|------|-----------|------------|----------|
| 3a | `GET /api/v1/gis/autocomplete/cities` | `q=tam`, `limit=5`, `domain=<slug>` | `200`, JSON **array** of `{ city, county, county_slug, label }` |
| 3b | `GET /api/v1/gis/autocomplete/counties` | `q=pin`, `limit=5`, `domain=<slug>` | `200`, array of `{ county, county_slug, label }` |
| 3c | Empty prefix | `q=zzznonexistent` | `200`, `[]` (not 404) |

**curl equivalents** (replace token and domain):

```bash
export TOKEN='your-pat'
export DOMAIN='your-verified-domain.com'

curl -sS -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8000/api/v1/gis/autocomplete/cities?q=tam&limit=5&domain=$DOMAIN" | jq .

curl -sS -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8000/api/v1/gis/autocomplete/counties?q=pin&limit=5&domain=$DOMAIN" | jq .
```

**502 / database errors on autocomplete:** Confirm migration **00007** applied (`CREATE EXTENSION pg_trgm`) and `gis_cities` / `gis_counties` are populated (GIS boundary sync). See [gis-api.md](gis-api.md) and [gis-sources.md](gis-sources.md).

### 4. Search using autocomplete geography

| Step | Operation | Body / query | Expected |
|------|-----------|--------------|----------|
| 4a | `POST /api/v1/search` | `?dataset=stellar`, body with `"city": "Largo"` (or a city from step 3a) | `200`, `{ "results", "hasMore", "nextSkip" }` |
| 4b | Same with invalid JSON types | string where number expected without flexible parse | `400`, `invalid search body: …` |

Mirror search results are **scalar-only** (no `Media` / navigation JSONB). Use `GET /api/v1/listings/{listingId}` for full property payloads.

**Production note:** City search had a **502** bug fixed in `a9cd351`; ensure that build is deployed before testing `city` on production.

### 5. GIS parcel proxy (sanity)

| Step | Operation | Parameters | Expected |
|------|-----------|------------|----------|
| 5a | `GET /api/v1/gis` | `bbox=-82.83,27.95,-82.79,27.98`, `limit=10`, auth + domain | `200`, GeoJSON `FeatureCollection` (may be empty if parcels not synced) |

### 6. MLS proxy spot-check

| Step | Operation | Notes | Expected |
|------|-----------|-------|----------|
| 6a | `GET /api/v1/listings` | `$top=1` only; **omit** `dataset` if Bridge web errors | `200`, OData-style collection |
| 6b | `GET /api/v1/bridge/stats` | Auth + domain | `200`, per-dataset mirror stats |

## Troubleshooting

| Symptom | Likely cause | Action |
|---------|--------------|--------|
| Blank `/swagger` page | CDN blocked | Allow `unpkg.com` or test `/openapi.json` with curl/Yaak |
| “No layout defined for StandaloneLayout” | `swagger-ui-standalone-preset.js` blocked or missing | Allow unpkg; hard-refresh; redeploy API with current `/swagger` HTML |
| Stale operations in UI | Old binary | `make openapi-sync && make build`, redeploy |
| `401` on `/api/v1/*` | Missing/invalid PAT or domain | Authorize; set `X-Domain-Slug` or `?domain=` |
| `403` | Dataset not on domain allowlist | Use allowed `?dataset=` or update `domains.allowed_mls_datasets` |
| `400` … `dataset` on web listings | Param forwarded before strip fix | Upgrade API; omit `dataset` on web routes |
| `502` on search + `city` | Geography SQL bug (pre-`a9cd351`) | Deploy current API |
| Autocomplete `502` | Missing `pg_trgm` or DB error | Run migration 00007; check API logs |
| CORS errors from browser app | Swagger host ≠ API host | Call API from same origin or configure CORS separately |

## Automated alternative

For scripted coverage of all Yaak paths (including autocomplete), use:

- [yaak-endpoint-smoke-test.md](yaak-endpoint-smoke-test.md)
- `./docs/scripts/test_yaak_endpoints.sh`

Swagger UI is best for **interactive** exploration, token setup, and sharing examples with integrators; the smoke script is better for **CI/regression** after deploy.

## Related docs

- [api.md](api.md) — route groups and auth model
- [routes-reference.md](routes-reference.md) — full method/path table
- [gis-api.md](gis-api.md) — GIS parcel proxy and autocomplete
- [idx-api-bridge-proxy.md](idx-api-bridge-proxy.md) — search, cache, `dataset` routing
- [yaak-api-collection.json](yaak-api-collection.json) — OpenAPI source (import into Yaak or Redocly)
