# Yaak collection smoke tests

Automated checks for every path in [`yaak-api-collection.json`](yaak-api-collection.json).

For **manual** exploration in the browser (authorize PAT, try autocomplete + search), see [swagger-ui-testing.md](swagger-ui-testing.md).

## Run (recommended)

```bash
# From repo root; optional .env for GOOSE_DBSTRING (SELECT fixtures only) and ADMIN_SEED_* login.
export YAAK_BEARER_TOKEN='…'   # production PAT — required for MLS/GIS/search/comps
export YAAK_DOMAIN_SLUG='your-verified-domain.com'
make test-api-smoke
```

View the structured failure report (for Cursor AI debugging):

```bash
make test-api-smoke-report
# or: cat tests/smoke/reports/latest.json
```

On failure, the test run also prints a **Cursor AI feedback block** to stdout — copy it into an agent chat.

## Legacy bash wrapper

```bash
./docs/scripts/test_yaak_endpoints.sh
```

This script delegates to `make test-api-smoke` when Go is available.

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `YAAK_BASE_URL` | `https://idx.quantyralabs.cc` | API host (not `APP_URL` unless `YAAK_USE_APP_URL=1`) |
| `YAAK_BEARER_TOKEN` | — | Domain PAT (`idx:access` or `idx:full`) |
| `YAAK_DOMAIN_SLUG` | first active `domains` row via `GOOSE_DBSTRING` | `X-Domain-Slug` header |
| `YAAK_DATASET` | `stellar` | `?dataset=` for MLS routes |
| `YAAK_BBOX` | `-82.8,27.9,-82.6,28.0` | GIS bbox smoke query |
| `GOOSE_DBSTRING` | — | Read-only `SELECT` for listing_key / photo_id fixtures |
| `ADMIN_SEED_EMAIL` / `ADMIN_SEED_PASSWORD` | — | Optional login to obtain PAT when `YAAK_BEARER_TOKEN` unset |
| `YAAK_SESSION_COOKIE` | — | Reserved for future admin session route tests |

**Production DB:** `.env` may point at Patroni production. The smoke runner only runs `SELECT` for listing fixtures. It does **not** call mutating admin routes (`POST /api/v1/admin/flood-enrich`, GIS upload/sync).

**Build tag:** Smoke tests use `//go:build smoke` and are **not** run by `make test`. Use `make test-api-smoke` explicitly.

## What is validated

Unlike the old bash-only script (status codes only), the Go smoke suite checks:

- HTTP status (with per-route acceptable sets, e.g. MLS `200|404|502`)
- JSON response shape via path assertions (`results`, `hasMore`, GeoJSON `FeatureCollection`, etc.)
- **`GET /api/v1/listings`** — Bridge web envelope: `bundle` (array) and `total` (number), **not** OData `value` (that shape is for `/api/v1/properties`)
- Image routes: `Content-Type: image/*` and body ≥ 5 KB on `200` (502 no longer accepted)
- Autocomplete: top-level JSON array

Test cases live in [`tests/smoke/cases/`](../tests/smoke/cases/). Edit JSON there to add routes or assertions.

## Known fixes (deploy required for search)

| Issue | Cause | Fix |
|-------|-------|-----|
| `listings_collection` asserts wrong key | Test expected OData `value`; Bridge web returns `bundle` | Fixed in smoke cases + OpenAPI `ListingsResponse` |
| `search_largo_active` → 502 | `ScanMirrorListingSearchRow` skipped `estimated_total_monthly_fees`, misaligning pgx column scan | Fixed in `internal/service/mls/listing_response.go` — **redeploy API** |
| Search 502 with `city` only (older builds) | Geography filter bound pattern array twice | Fixed in `a9cd351` — ensure production includes that commit |

After deploying the API, re-run `make test-api-smoke`. `search_largo_active` should return **200** with `{ results, hasMore, nextSkip }`.

## Next.js client examples

When tests pass with a valid PAT, resolved requests are exported to [`docs/client-examples/`](client-examples/) (method, URL, headers, body, and a `fetch()` snippet). See [`docs/client-examples/README.md`](client-examples/README.md).

## Interpreting results

- **401** without `YAAK_BEARER_TOKEN` — expected for `/api/v1/*` and `/images/*`.
- **502** on `POST /api/v1/search` with `city` — if undeployed, check API logs for pgx scan errors or `expected N arguments, got N+1` (geography bind bug). Redeploy after `listing_response.go` scan fix.
- **`ADMIN_SEED_*` login** — only works when those credentials exist in the target environment’s `users` table (often local seed, not production).
- **Admin routes** — skipped unless `YAAK_SESSION_COOKIE` is wired in a future slice.

## Failure report format

`tests/smoke/reports/latest.json` contains:

- `failures[].request.curl` — reproducible curl command
- `failures[].expected` vs `failures[].actual` — status and JSON path context
- `failures[].diagnosis` — human-readable summary
- `failures[].nextjs_hint` — guidance for client implementation
