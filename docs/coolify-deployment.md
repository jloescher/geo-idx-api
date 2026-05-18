# Coolify — production and staging

This guide describes how to run **Quantyra IDX API** on [Coolify](https://coolify.io/) using the **[Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)**. **FrankenPHP 8.5 + PHP extensions** are pre-built in **GHCR**; Coolify builds the **application layer** (Composer, Vite, Artisan caches) from [`Dockerfile.production`](../Dockerfile.production) or [`Dockerfile.staging`](../Dockerfile.staging) on each deploy.

Use **two Coolify projects** (staging and production), each with its own PostgreSQL database and environment variables.

**Queues:** **`database`** driver (`jobs` table). Deploy **web** (Octane), **worker(s)**, **scheduler**, and **idx-images**. See §2.5 for scaling workers.

**Related:** [README.md](../README.md), [AGENTS.md](../AGENTS.md).

---

## 1. Two-layer images — FrankenPHP base (GHA) + app (Coolify)

```mermaid
flowchart LR
  subgraph gha [GitHub Actions]
    baseProd["Dockerfile.frankenphp-base.production"]
    baseStg["Dockerfile.frankenphp-base.staging"]
    ghcrProd["ghcr.io/owner/repo-frankenphp:production"]
    ghcrStg["ghcr.io/owner/repo-frankenphp:staging"]
    baseProd --> ghcrProd
    baseStg --> ghcrStg
  end
  subgraph coolify [Coolify VPS]
    df["Dockerfile.production / .staging"]
    app["octane | queue-worker | scheduler"]
    df --> app
  end
  ghcrProd --> df
  ghcrStg --> df
```

| Layer | Built by | Dockerfile | GHCR tag |
|-------|----------|------------|----------|
| **FrankenPHP base (production)** | GHA on push to `main` | [`Dockerfile.frankenphp-base.production`](../Dockerfile.frankenphp-base.production) | `ghcr.io/<owner>/<repo>-frankenphp:production` |
| **FrankenPHP base (staging)** | GHA on push to `staging` | [`Dockerfile.frankenphp-base.staging`](../Dockerfile.frankenphp-base.staging) | `ghcr.io/<owner>/<repo>-frankenphp:staging` |
| **Application** | **Coolify** per deploy | [`Dockerfile.production`](../Dockerfile.production) or [`.staging`](../Dockerfile.staging) | *(local image on the server)* |

Workflow: [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml) — **`linux/amd64` only**.

**What the base includes:** `dunglas/frankenphp:php8.5-alpine`, `install-php-extensions` (pgsql, gd, opcache, …), staging **Xdebug**, Composer binary, `/var/cache/geoidx` layout. Staging base sets default **`memory_limit=768M`** in PHP (`Dockerfile.frankenphp-base.staging`); per-process `-d memory_limit=…` in app targets (web/scheduler) can override.

**What Coolify still builds:** `COPY` app source, `composer install`, config cache on all targets. **Octane only:** `npm ci` / Vite, `filament:assets`, and (production) `view:cache`. Worker and scheduler targets skip Node/Vite — **not** `install-php-extensions` on every deploy.

### 1.1 Bootstrap GHCR base images

Before the first Coolify app build:

1. Merge or push to **`main`** → publishes `ghcr.io/<owner>/<repo>-frankenphp:production`.
2. Push to **`staging`** → publishes `ghcr.io/<owner>/<repo>-frankenphp:staging`.
3. Or run the workflow manually: **Actions → Docker publish (FrankenPHP base) → Run workflow**.

Add **GHCR registry credentials** on the Coolify server (read access) so `docker build` can `FROM` the base image.

### 1.2 Coolify — Dockerfile build pack (API apps)

| Coolify field | Production | Staging |
|---------------|------------|---------|
| **Build pack** | Dockerfile | Dockerfile |
| **Dockerfile** | `Dockerfile.production` | `Dockerfile.staging` |
| **Docker Build Target** | `octane` (web), `queue-worker`, or `scheduler` per app — see §1.4 | Same |
| **Port (web only)** | **8000** | **8000** |

**Build argument `FRANKENPHP_BASE_IMAGE`** — baked into [`Dockerfile.staging`](../Dockerfile.staging) / [`.production`](../Dockerfile.production). **Important:** if Coolify has `FRANKENPHP_BASE_IMAGE` in environment variables with [inject build args](https://coolify.io/docs/builds/packs/dockerfile#inject-build-args-to-dockerfile) enabled, that value **overrides** the Dockerfile default (see your deploy log: `ARG FRANKENPHP_BASE_IMAGE=ghcr.io/jloescher/geoidx-frankenphp:staging`).

| Environment | Dockerfile default (GHA) | Also published as alias |
|-------------|--------------------------|-------------------------|
| Production | `ghcr.io/jloescher/geo-idx-api-frankenphp:production` | `ghcr.io/jloescher/geoidx-frankenphp:production` |
| Staging | `ghcr.io/jloescher/geo-idx-api-frankenphp:staging` | `ghcr.io/jloescher/geoidx-frankenphp:staging` |

If you set `FRANKENPHP_BASE_IMAGE` in Coolify, it must match a tag that **exists on GHCR** (run the GHA workflow first). To use the Dockerfile default instead, **remove** `FRANKENPHP_BASE_IMAGE` from Coolify env (or mark it build-time only if your Coolify version supports that).

**Before the first app deploy:** run GHA **Docker publish (FrankenPHP base)** on `staging` / `main`, then configure **GHCR read** on the Coolify server.

**Runtime env:** Same `DB_*`, `APP_KEY`, `QUEUE_CONNECTION`, and URLs on web, worker, and scheduler. **Turnstile** keys on **web** only.

Prefer **`APP_ENV` at runtime**, not only at image build time ([Coolify env docs](https://coolify.io/docs/builds/packs/dockerfile#environment-variables)).

### 1.3 Deploy layout — three API applications

**Option A — separate build targets (simplest healthchecks)**

| App | Dockerfile | Build target | Coolify healthcheck |
|-----|------------|--------------|---------------------|
| Web | `Dockerfile.*` | `octane` | HTTP `GET /up` on **8000** |
| Worker | same file | `queue-worker` | Image **pgrep** (or disable HTTP) |
| Scheduler | same file | `scheduler` | Image **pgrep** (or disable HTTP) |

Each deploy rebuilds the app layers from git; the FrankenPHP base is **pulled** from GHCR (fast when unchanged).

**Option B — one web build, shared image for worker/scheduler**

Build **only** target `octane` on the web app. On worker and scheduler Coolify apps, use the **same built image** with **command** overrides (§1.5) and **disable HTTP healthcheck** (octane image probes port 8000).

### 1.4 Canonical Docker build targets

| Target | Build stage | Purpose |
|--------|-------------|---------|
| `octane` | `builder-web` | Web (FrankenPHP / Octane on **8000**); includes Vite + Filament assets |
| `queue-worker` | `builder-cli` | Queue worker (**pgrep** healthcheck); no Vite/Node |
| `scheduler` | `builder-cli` | Scheduler (**pgrep** healthcheck); no Vite/Node |
| `idx-api-worker` / `idx-api-scheduler` | same as above | Aliases of the above |

### 1.5 Command overrides (Option B only)

**Worker — staging:** `/bin/sh -lc 'exec php -d memory_limit=768M artisan queue:work --queue=${WORKER_QUEUES:-default} --sleep=1 --tries=3 --timeout=120'` (matches `Dockerfile.staging` `queue-worker` CMD and staging FrankenPHP base default)

**Worker — production:** `/bin/sh -lc 'exec php -d memory_limit=512M artisan queue:work --queue=${WORKER_QUEUES:-default} --sleep=1 --tries=3 --timeout=120'`

Set **`WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist`** on the worker service so Bridge and Spark replica jobs run (scheduled kickoffs every 15 minutes via `bridge-listings-replica-sync` and `spark-listings-replica-sync`). During replication catch-up, run **two or more worker replicas** — optionally dedicate one replica to **`bridge-sync-persist,spark-sync-persist`** only and another to **`default,bridge-sync-fetch,spark-sync-fetch`** so parallel Postgres writes keep up while fetch jobs stay rate-limited.

**Spark hosts:** workers and the web container need outbound HTTPS to **`replication.sparkapi.com`** (sync) and **`sparkapi.com`** (live RESO proxy). See [spark/idx-api-integration.md](spark/idx-api-integration.md).

**Scheduler — staging:** `php -d memory_limit=384M artisan schedule:work --whisper`  
**Scheduler — production:** `php -d memory_limit=256M artisan schedule:work --whisper`

The scheduler image runs `schedule:work` (one `schedule:run` per minute). **`INFO  No scheduled commands are ready to run`** is normal on minutes when nothing is due (e.g. at `:07` only `*/10` and `*/15` tasks are waiting). With `--whisper`, that line is omitted; you will see output when a task actually runs (`Running [coingecko-price-refresh]`, etc.). **Scheduled jobs are pushed to the database queue** — the **worker** container must be running to execute them (scheduler logs ≠ worker logs).

### 1.6 idx-images (fourth app)

[`Dockerfile.idx-images`](../Dockerfile.idx-images) on Coolify, port **8080**. Upstream **`idx-api:8000`** on the shared Docker network.

### 1.7 Local / smoke builds

```bash
# 1) Build or pull base
docker buildx build --platform linux/amd64 -f Dockerfile.frankenphp-base.production \
  -t ghcr.io/<owner>/<repo>-frankenphp:production .

# 2) Build app
docker buildx build --platform linux/amd64 -f Dockerfile.production --target octane \
  --build-arg FRANKENPHP_BASE_IMAGE=ghcr.io/<owner>/<repo>-frankenphp:production .
```

---

## 2. Production — four applications

### 2.1 API web (Octane)

| Field | Value |
|-------|--------|
| **Name** | `idx-api` (for `idx-images` upstream) |
| **Dockerfile** | `Dockerfile.production` |
| **Build target** | `octane` |
| **Build arg** | `FRANKENPHP_BASE_IMAGE=ghcr.io/<owner>/<repo>-frankenphp:production` |
| **Port** | **8000** |

### 2.2 API queue worker

| Field | Value |
|-------|--------|
| **Dockerfile** | `Dockerfile.production` |
| **Build target** | `queue-worker` (Option A) or shared `octane` image (Option B) |
| **Healthcheck** | Use image pgrep, or disable HTTP if using Option B |

### 2.3 API scheduler

| Field | Value |
|-------|--------|
| **Dockerfile** | `Dockerfile.production` |
| **Build target** | `scheduler` (Option A) or shared `octane` image (Option B) |

### 2.4 Image edge (Nginx)

`Dockerfile.idx-images`, port **8080**, `GET /health`.

### 2.5 Multiple queue workers

Coolify **replicas** on the worker app, or duplicate worker applications. **One** scheduler only.

---

## 3. Staging

Same as production with **`Dockerfile.staging`**, build arg `FRANKENPHP_BASE_IMAGE=ghcr.io/<owner>/<repo>-frankenphp:staging`, and staging memory limits in the Dockerfile `CMD`.

---

## 4. Environment variables

Copy from **`.env.example`**. Per application for web, workers, and scheduler.

| Variable | Notes |
|----------|--------|
| `FRANKENPHP_BASE_IMAGE` | **Build-time** on Coolify (§1.2) |
| `DB_*`, `APP_KEY`, `QUEUE_CONNECTION` | Runtime, all API apps |
| `CLOUDFLARE_TURNSTILE_*` | Web app only |

---

## 5. Post-deployment (required)

Run migrations **once per environment** before workers can process jobs reliably. Staging/production use `CACHE_STORE=database` and `SESSION_DRIVER=database`, which require the `cache`, `cache_locks`, `sessions`, and `jobs` tables from Laravel’s default migrations.

**On the Octane container** (or any API container with the same `DB_*` env):

```bash
php artisan migrate --force
php artisan optimize
```

Verify:

```bash
php artisan migrate:status | head -20
php artisan crypto:refresh-prices   # optional smoke test (CoinGecko + cache write)
```

In Coolify you can use **Execute command** on the web app after deploy, or a **Post-deployment** hook that runs `php artisan migrate --force`.

---

## 6. Route cache

Omit `route:cache` at image build when multiple `IDX_PLATFORM_HOSTS` share route names. Post-deploy `route:cache` only for single-host setups.

---

## 7. CPU and memory

See [AGENTS.md](../AGENTS.md). **Container maximum memory must exceed PHP `memory_limit` plus headroom** (~250–400 MB on workers for job payloads and extensions; ~300 MB on web for Opcache).

| Service | PHP `memory_limit` (image default) | Coolify **Maximum Memory Limit** (starting point) |
|---------|-----------------------------------|-----------------------------------------------------|
| Worker — **staging** | **768M** (`Dockerfile.staging` + `Dockerfile.frankenphp-base.staging`) | **1024 MB** |
| Worker — production | **512M** | **896–1024 MB** |
| Scheduler — staging | **384M** | **384–512 MB** |
| Web (Octane) — staging | **384M** | **1024–1536 MB** |

Set **Number of CPUs** on workers to **0.5–1** during Bridge replication catch-up. Leaving all limits at **`0`** in Coolify means no cgroup cap (container can starve other services on a small VPS).

---

## 8. Coolify checklist

1. Publish both FrankenPHP bases via GHA (§1.1).
2. GHCR read credentials on Coolify.
3. Three API apps: Dockerfile pack + `FRANKENPHP_BASE_IMAGE` + correct target (§1.3).
4. idx-images on **8080**, network alias **`idx-api`**.
5. Shared runtime env; deploy; **run `php artisan migrate --force` (§5)** on first deploy and after new migrations.
6. Redeploy worker/scheduler after migrations if they crashed on missing tables.

---

## 9. Telescope / Pulse (staging diagnostics)

Set the **same** observability env on **web (octane)**, **worker**, and **scheduler**:

| Variable | Staging value |
|----------|----------------|
| `TELESCOPE_ENABLED` | `true` |
| `TELESCOPE_LOG_LEVEL` | `info` (required for `bridge.replication.*` logs in Telescope **Logs**) |
| `PULSE_ENABLED` | `true` (optional queue metrics at `/pulse`) |

**Post-deploy:** if you change `TELESCOPE_*` at runtime, run `php artisan config:cache` (or `config:clear`) on each container so values are not frozen from image build.

**Bridge replication in Telescope** (`/telescope` on the platform host, HTTP Basic Auth in production only):

- **Logs** — filter `bridge.replication` (`page_fetched`, `page_persisted`, `failed`, `kickoff`)
- **Events** — `BridgeReplicationPageFetched`, `BridgeReplicationPagePersisted`, `BridgeReplicationBatchFailed`
- **Jobs** / **Batches** — `BridgeSyncFetchPageJob`, `bridge-replica-persist:{dataset}`

Workers must use the **same `DB_*`** as web so `telescope_entries` is shared.

**Pulse `/pulse` shows "incomplete object" (Cache card):** Laravel 13 `config/cache.php` `serializable_classes` must allow `stdClass` and `Illuminate\Support\Collection` (Pulse caches aggregated card data in the app cache). This repo configures an allowlist; after deploy run `php artisan cache:clear` once to drop bad entries written while `serializable_classes` was `false`.

---

## 10. Troubleshooting

### Scheduler repeats “No scheduled commands are ready to run”

**Expected** if you are not using `--whisper` and the current minute is not a due time. Registered tasks (UTC): `coingecko-price-refresh` every **10** min (`:00`, `:10`, …); `mls:refresh-cache` and `bridge-listings-replica-sync` every **15** min (`:00`, `:15`, `:30`, `:45`); `bridge:purge-replica-pages` daily **04:15**; closed-listings purge **03:05**; GIS probe **Monday 06:30**. Bridge fetch jobs target **2 GETs/sec** (`BRIDGE_SYNC_MAX_REQUESTS_PER_SECOND=2`).

**Verify on the scheduler container:**

```bash
php artisan schedule:list
php artisan schedule:run -v    # at :00/:10/:15 you should see "Running [...]"
```

**Workers still idle?** The scheduler only **dispatches** queue jobs. Confirm the **worker** app is healthy, migrations ran (§5), and watch **worker** logs at a due minute — not the scheduler log stream.

### `relation "cache" does not exist` (workers / queue)

`CACHE_STORE=database` (default in `.env.example`) stores Laravel cache and queue restart signals in PostgreSQL. The **`cache`** and **`cache_locks`** tables come from `database/migrations/0001_01_01_000001_create_cache_table.php`.

**Fix:** run migrations on the staging/production database (§5):

```bash
php artisan migrate --force
```

Then restart workers (`queue:restart` happens automatically on next deploy, or redeploy the worker app). `RefreshCryptoPricingJob` also writes to this cache store.

**Symptoms in worker logs:** repeated `SQLSTATE[42P01]: relation "cache" does not exist` on `select * from "cache" where "key" in (...illuminate:queue:restart)` — the worker crashes between jobs when checking the queue restart signal. `BridgeSyncJob` may show **DONE** but you will not see `BridgeSyncFetchPageJob` / `BridgePersistReplicaChunkJob` succeed until migrations run (fetch jobs use `BridgeRateLimitGuard`, which writes to the same cache store).

### `Allowed memory size exhausted` on `BridgePersistReplica*`

Replication pages are up to **2000** listings with large `raw_data` JSON. Replication **always** includes **`Media`** in `$select` (independent of **`BRIDGE_SYNC_INCLUDE_MEDIA`**, which applies to incremental `Property` sync only). Bulk replication is scoped to **Active + Pending** via OData **`$filter`** on the first `/Property/replication` page; **`BRIDGE_SYNC_INCLUDE_MEDIA=true`** still increases incremental sync payload. Persist work runs as parallel **`BridgePersistReplicaChunkJob`** batches on **`bridge-sync-persist`** (`BRIDGE_SYNC_PERSIST_JOB_CHUNK`; use **25–50** with media). If OOM persists, lower the chunk env, add persist worker replicas, set Coolify worker **Maximum Memory Limit** ≥ **1024 MB**, or raise PHP memory in the worker start command (staging default **768M**).

Related: `SESSION_DRIVER=database` needs the **`sessions`** table (`0001_01_01_000000_create_users_table.php`); `QUEUE_CONNECTION=database` needs **`jobs`** (`0001_01_01_000002_create_jobs_table.php`).

### `base name (${FRANKENPHP_BASE_IMAGE}) should not be blank`

Coolify did not pass **`FRANKENPHP_BASE_IMAGE`** and the Dockerfile had no default. Use current `Dockerfile.staging` / `.production` (they include defaults), or set env **`FRANKENPHP_BASE_IMAGE`** in Coolify with inject build args enabled.

### `ghcr.io/jloescher/geoidx-frankenphp:staging: not found`

Coolify injected `FRANKENPHP_BASE_IMAGE=ghcr.io/jloescher/geoidx-frankenphp:staging` from your app env, which overrides the Dockerfile default. That tag only exists after GHA publishes the **alias** (see workflow). **Fix:** run **Docker publish (FrankenPHP base)** on `staging`, **or** delete `FRANKENPHP_BASE_IMAGE` from Coolify env, **or** set it to `ghcr.io/jloescher/geo-idx-api-frankenphp:staging`.

### `failed to resolve FRANKENPHP_BASE_IMAGE` / `pull access denied`

Run GHA **Docker publish (FrankenPHP base)** first. Add **GHCR read** credentials on the Coolify server.

### `map has no entry for key "Health"`

Use build targets **`queue-worker`** / **`scheduler`** (Option A), or disable HTTP healthcheck when reusing the **`octane`** image (Option B).

### Deploy still runs `install-php-extensions` for a long time

Coolify is not using the GHCR base — check `FROM ${FRANKENPHP_BASE_IMAGE}` and the build arg.

### Octane: Caddy storage / autosave warnings

Non-fatal FrankenPHP/Caddy noise; **`/up`** on **8000** should still pass.

**Octane deploy log: one failed `curl` then healthy** — normal while Octane binds port 8000 during the healthcheck start period; rolling update should still complete.

### `ViteManifestNotFoundException`

Vite stage failed during Coolify build — check Node stage logs.

### ARM on amd64 VPS

Use GHA **`linux/amd64`** bases and `docker buildx --platform linux/amd64` locally.

---

## 10. References

- [Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)
- [Build arguments](https://coolify.io/docs/builds/packs/dockerfile#build-arguments)
- [Pre / post deployment commands](https://coolify.io/docs/builds/packs/dockerfile#pre-post-deployment-commands)
