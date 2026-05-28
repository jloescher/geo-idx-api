# Yaak collection smoke tests

Automated checks for every path in [`yaak-api-collection.json`](yaak-api-collection.json).

For **manual** exploration in the browser (authorize PAT, try autocomplete + search), see [swagger-ui-testing.md](swagger-ui-testing.md).

## Run

```bash
# From repo root; loads .env for GOOSE_DBSTRING (SELECT fixtures only) and optional ADMIN_SEED_*.
export YAAK_BEARER_TOKEN='…'   # recommended for production — dashboard domain API key
export YAAK_DOMAIN_SLUG='your-verified-domain.com'  # optional override
./docs/scripts/test_yaak_endpoints.sh
```

| Variable | Default | Purpose |
|----------|---------|---------|
| `YAAK_BASE_URL` | `https://idx.quantyralabs.cc` | API host (not `APP_URL` unless `YAAK_USE_APP_URL=1`) |
| `YAAK_BEARER_TOKEN` | — | Domain PAT (`idx:access` or `idx:full`) |
| `YAAK_DOMAIN_SLUG` | first active `domains` row | `X-Domain-Slug` header |
| `YAAK_DATASET` | `stellar` | `?dataset=` for MLS routes |

**Production DB:** `.env` may point at Patroni production. The script only runs `SELECT` for listing fixtures. It does **not** call `POST /api/v1/admin/flood-enrich`.

**Admin routes** in the Yaak file (`/api/v1/admin/*`) require a dashboard `session_id` cookie and are **skipped**.

## Interpreting results

- **401** without `YAAK_BEARER_TOKEN` — expected for `/api/v1/*` and `/images/*`.
- **502** on `POST /api/v1/search` with `city` set — fixed in commit `a9cd351` (geography SQL arg count); redeploy API required on production.
- **`ADMIN_SEED_*` login** — only works when those credentials exist in the target environment’s `users` table (often local seed, not production).
