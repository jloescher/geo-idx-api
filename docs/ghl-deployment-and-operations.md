# GHL Integration — Deployment & Operations

Covers **Docker**, **docker-compose**, **Dokploy**, migrations, queues, and scheduled tasks. **GHL-specific** runtime lives in **`idx-api`**; **container images** for the whole stack are defined at the **monorepo root** under **`docker/`** (see [../docker/README.md](../docker/README.md)).

---

## Docker images (monorepo root context)

All Dockerfiles use **`COPY idx-api/`** or **`COPY geo-web/`** from the repo root. Set **build context = repository root** (`.`) in Dokploy and in `docker compose build` — **not** `./idx-api` or `./geo-web` alone.

| Service | Dockerfile | Base | Exposed port |
|---------|------------|------|----------------|
| **idx-api** | `docker/Dockerfile.idx-api` | `phpswoole/swoole:php8.5-alpine` | `8000` |
| **geo-web** | `docker/Dockerfile.geo-web` | `phpswoole/swoole:php8.5-alpine` | `8000` |
| **idx-images** | `docker/Dockerfile.idx-images` | `nginx:1.27-alpine` | `8080` |

**idx-api image notes**

- **Logs:** `/var/log/geoidx/` includes `ghl_audit.log` (created at build for write access).
- **Bridge / MLS cache paths** are created in the same Dockerfile (see [idx-api-bridge-proxy.md](idx-api-bridge-proxy.md)).

**Root `.dockerignore`** excludes `vendor/`, `node_modules`, `docs/`, `mobile/`, etc., to keep build contexts small.

**Canonical Dokploy table:** [../docker/README.md](../docker/README.md).

Build locally (from repository root):

```bash
docker build -f docker/Dockerfile.idx-api -t quantyra/idx-api:latest .
docker build -f docker/Dockerfile.geo-web -t quantyra/geo-web:latest .
docker build -f docker/Dockerfile.idx-images -t quantyra/idx-images:latest .
docker compose build
```

---

## docker-compose (repository root)

Service **`idx-api`** includes environment variables for Quantyra public URLs and GHL (see root `.env.example`). Relevant additions:

- `IDX_PLATFORM_URL` — default `https://idx.quantyralabs.cc` (IDX App: GHL + independent dashboard)
- `IDX_API_PUBLIC_URL` — default `https://idx-api.quantyralabs.cc` (IDX API: endpoints + JS widgets)
- `IDX_IMAGES_PUBLIC_URL` — default `https://idx-images.quantyralabs.cc` (image rewrite proxy)
- `GHL_CLIENT_ID`, `GHL_CLIENT_SECRET`, `GHL_REDIRECT_URI`, `GHL_WEBHOOK_*`, `GHL_ADMIN_REFRESH_TOKEN`, `GHL_SCOPES`, `GHL_AUDIT_*`

Full list: [ghl-environment-variables.md](ghl-environment-variables.md).

---

## Dokploy (recommended layout)

| Setting | Value |
|---------|--------|
| Repository | `quantyra-geoidx` (or your fork) |
| Application | `idx-api` (repeat for `geo-web` and `idx-images` with their Dockerfiles) |
| Build context | **Repository root** (`.`) — not `idx-api/` alone |
| Dockerfile path | `docker/Dockerfile.idx-api` (or `docker/Dockerfile.geo-web` / `docker/Dockerfile.idx-images`) |
| Published port / reverse proxy | Per app: e.g. `idx-api.quantyralabs.cc` → **8000**; `idx-images` → **8080** (see Traefik labels in `docker-compose.yml`). |
| Health check | **idx-api** / **geo-web:** `php artisan octane:status`. **idx-images:** `GET /health` on port **8080**. |

**Post-deploy commands** (run once per release or via Dokploy “Execute command”):

```bash
php artisan migrate --force
php artisan db:seed --class=GhlConfigSeeder --force
php artisan config:cache
php artisan route:cache
php artisan view:cache
```

**Queue worker** (separate process or second Dokploy service using the **idx-api** image — same `docker/Dockerfile.idx-api`, override `CMD` / entrypoint to `queue:work`):

```bash
php artisan queue:work --sleep=3 --tries=3
```

GHL jobs use queue names from config (`GHL_QUEUE_SYNC`, `GHL_QUEUE_WEBHOOKS`, `GHL_QUEUE_MAINTENANCE`); default is `default` if unset.

**Scheduler** (host cron or Dokploy cron hitting the container):

```cron
* * * * * cd /var/www/html && php artisan schedule:run >> /dev/null 2>&1
```

Scheduled command: **`ghl:refresh-tokens`** (hourly, `withoutOverlapping`) defined in `idx-api/routes/console.php`.

---

## Migrations path

GHL migrations are in:

```
idx-api/database/migrations/ghl/
```

They are loaded via `App\Providers\AppServiceProvider::boot()`:

```php
$this->loadMigrationsFrom(database_path('migrations/ghl'));
```

Run:

```bash
cd idx-api
php artisan migrate
# or explicit path (optional):
php artisan migrate --path=database/migrations/ghl
```

---

## Observability

| Concern | Where |
|---------|--------|
| Webhook duplicates | `ghl_webhook_events.webhook_id` (unique when present). |
| Sync failures | `ghl_sync_logs.sync_status`, `error_message`. |
| Token health | `ghl_oauth_tokens.expires_at`, `status`. |
| Audit | Table `ghl_audit_logs` + file `GHL_AUDIT_LOG_PATH` if configured. |

---

## Troubleshooting

| Symptom | Check |
|---------|--------|
| 403 on OAuth callback | Session `state` mismatch; same-site cookies; HTTPS and `SESSION_DOMAIN`. |
| 401 on `/api/leadconnector/*` | Expired access token; refresh via GHL or `POST /oauth/leadconnector/refresh` with admin token. |
| Webhook 401 | `GHL_WEBHOOK_SECRET` / header name / algorithm vs GHL dashboard. Set `GHL_WEBHOOK_REQUIRE_SIGNATURE=false` only in non-production. |
| Widget 403 Origin | URL not listed in `ghl_registered_urls` primary/additional URLs. |
| Leads not in GHL | Queue worker running? `ghl_sync_logs` errors? Active token for `ghl_location_id`? |
| Docker build fails (`COPY failed`) | Build context must be the **repo root**, Dockerfile path **`docker/Dockerfile.*`** — see [../docker/README.md](../docker/README.md). |
| Wrong app in container | Do not set context to `idx-api/` only; Laravel files must land at `/var/www/html` via `COPY idx-api/ .` from root. |

---

## Related docs

- [../docker/README.md](../docker/README.md) — Dokploy three-service layout and Dockerfile paths.
- [ghl-marketplace-integration.md](ghl-marketplace-integration.md)
- [ghl-environment-variables.md](ghl-environment-variables.md)
- [ghl-api-routes-reference.md](ghl-api-routes-reference.md)
- [idx-api-bridge-proxy.md](idx-api-bridge-proxy.md) — MLS proxy, image edge, env for Bridge.
