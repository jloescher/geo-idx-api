# Deployment & operations

Covers **Docker**, **docker-compose**, **[Coolify](coolify-deployment.md)**, **Dokploy**, migrations, queues, and scheduled tasks. Container images are defined at the project root (see [README.md](../README.md)).

---

## Docker images (project root context)

All Dockerfiles build from this project root context. Set **build context = project root** (`.`) in Coolify, Dokploy, and in `docker compose build`.

| Service | Dockerfile | Base | Exposed port |
|---------|------------|------|----------------|
| **idx-api (web)** | [`Dockerfile.production`](../Dockerfile.production) (target **`octane`**) | FrankenPHP + PHP 8.5 Alpine + **intl** (Filament) | **8000** |
| **idx-api (worker)** | same file (target **`queue-worker`**) | (same image) | — |
| **idx-api (scheduler)** | same file (target **`scheduler`**) | (same image) | — |
| **idx-api (staging)** | [`Dockerfile.staging`](../Dockerfile.staging) | FrankenPHP + Xdebug | **8000** |
| **idx-images** | `Dockerfile.idx-images` | `nginx:1.27-alpine` | `8080` |

**idx-api image notes**

- **Logs:** `/var/log/geoidx/` (paths created in the Dockerfile where applicable).
- **Bridge / MLS cache paths** are created in the same Dockerfile (see [idx-api-bridge-proxy.md](idx-api-bridge-proxy.md)).

**Root [`.dockerignore`](../.dockerignore)** shrinks build context (`vendor/`, `node_modules/`, `docs/`, `.git`, etc.) for Coolify and local builds.

**Canonical deployment notes:** [README.md](../README.md), [AGENTS.md](../AGENTS.md), and **[coolify-deployment.md](coolify-deployment.md)** (Coolify production & staging).

Build locally (from project root):

```bash
docker build -f Dockerfile.production --target octane -t quantyra/idx-api:latest .
docker build -f Dockerfile.production --target queue-worker -t quantyra/idx-api-worker:latest .
docker build -f Dockerfile.production --target scheduler -t quantyra/idx-api-scheduler:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .
docker compose build
```

---

## docker-compose (project root)

Services use environment variables from root **`.env.example`** as a template. Common URL-related variables:

- `IDX_PLATFORM_URL` — platform app (marketing + dashboard) public URL  
- `IDX_API_PUBLIC_URL` / `API_URL` — API + widgets host  
- `IDX_IMAGES_PUBLIC_URL` / `IMAGE_URL` — image proxy host  
- `IDX_PLATFORM_HOSTS`, `IDX_API_HOSTS` — comma-separated hosts for `Route::domain()` routing  

---

## Coolify

**Primary guide:** **[Coolify deployment (production & staging)](coolify-deployment.md)** — four applications per environment, Dockerfile build pack, ports **8000** / **8080**, PostgreSQL **`QUEUE_CONNECTION=database`**, networking, post-deploy, route-cache caveat, CPU/RAM.

Quick reference:

| Concern | Production | Staging |
|---------|------------|---------|
| API Dockerfile | `Dockerfile.production` | `Dockerfile.staging` |
| Build targets | `octane`, `queue-worker`, `scheduler` | same |
| idx-images Dockerfile | `Dockerfile.idx-images` | same |
| API port | **8000** | **8000** |
| idx-images port | **8080** | **8080** |

---

## Dokploy (recommended layout)

| Setting | Value |
|---------|--------|
| Repository | Your GeoIDX / idx-api fork |
| Application | `idx-api` and `idx-images` |
| Build context | **Project root** (`.`) |
| Dockerfile path | `Dockerfile.production` (or `Dockerfile.staging`) / `Dockerfile.idx-images` |
| Published port / reverse proxy | API host → **8000**; idx-images → **8080** |
| Health check | **idx-api:** `php artisan octane:status` or `GET /up` on **8000**. **idx-images:** `GET /health` on **8080**. |

**Post-deploy commands** (run once per release or via platform “Execute command”):

```bash
php artisan migrate --force
php artisan config:cache
php artisan route:cache
php artisan view:cache
```

**`route:cache`:** skip if your app registers the same route names on multiple `Route::domain()` hosts (see `routes/web.php`); the Docker production image omits route caching at build for that reason.

**Queue worker** (separate service using the **same** API image, Docker target **`queue-worker`**):

| Image | Default `queue:work` command |
|-------|------------------------------|
| `Dockerfile.production` | `php -d memory_limit=512M artisan queue:work …` |
| `Dockerfile.staging` | `php -d memory_limit=768M artisan queue:work …` (FrankenPHP staging base also sets **`memory_limit=768M`** in `php.ini`) |

Use the image CMD unless your platform overrides it. Env **`WORKER_QUEUES`** defaults to `default` in the Dockerfile; set **`default,bridge-sync-fetch,bridge-sync-persist`** on staging/production workers for Bridge replica jobs (see [Coolify deployment](coolify-deployment.md)). Set e.g. `default,gis` if you use a dedicated GIS queue (`config/gis.php`).

**Scheduler** (separate service, target **`scheduler`**, or host cron):

```cron
* * * * * cd /var/www/html && php artisan schedule:run >> /dev/null 2>&1
```

Scheduled tasks are defined in `routes/console.php`.

---

## Migrations

Core migrations live under `database/migrations/`. For a **file-by-file inventory**, PostGIS requirements, and the legacy `dropIfExists` cleanup migration, see **[Database migrations](database-migrations.md)**.

```bash
php artisan migrate --force
```

On first deploy or after pulling migration changes, run migrate from the API container (or host) against the correct `DB_*` target. Prefer **`migrate:fresh` only on disposable** databases (never shared staging/production unless intentional).

---

## Troubleshooting

| Symptom | Check |
|---------|--------|
| Docker build fails (`COPY failed`) | Build context must be the **project root**; Dockerfile path **`Dockerfile.*`** — see [README.md](../README.md). |
| Wrong app in container | Do not set a nested context; Laravel files must land at `/var/www/html` via `COPY . .` from project root. |

---

## Related docs

- [README.md](../README.md) — project Docker build and setup instructions.  
- [coolify-deployment.md](coolify-deployment.md) — Coolify production & staging.  
- [idx-api-bridge-proxy.md](idx-api-bridge-proxy.md) — MLS proxy, image edge, Bridge env.
