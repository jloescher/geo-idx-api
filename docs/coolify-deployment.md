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

**What the base includes:** `dunglas/frankenphp:php8.5-alpine`, `install-php-extensions` (pgsql, gd, opcache, …), staging **Xdebug**, Composer binary, `/var/cache/geoidx` layout.

**What Coolify still builds:** `COPY` app source, `composer install`, `npm ci` / Vite, `filament:assets`, config/view cache — **not** `install-php-extensions` on every deploy.

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

**Build argument (required)** — Coolify **Advanced → Build Arguments** (or environment variable with [inject build args](https://coolify.io/docs/builds/packs/dockerfile#inject-build-args-to-dockerfile) enabled):

| Environment | `FRANKENPHP_BASE_IMAGE` |
|-------------|-------------------------|
| Production | `ghcr.io/<owner>/<repo>-frankenphp:production` |
| Staging | `ghcr.io/<owner>/<repo>-frankenphp:staging` |

Example for repo `jloescher/geo-idx-api`:

- `ghcr.io/jloescher/geo-idx-api-frankenphp:production`
- `ghcr.io/jloescher/geo-idx-api-frankenphp:staging`

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

| Target | Purpose |
|--------|---------|
| `octane` | Web (FrankenPHP / Octane on **8000**) |
| `queue-worker` | Queue worker (**pgrep** healthcheck) |
| `scheduler` | Scheduler (**pgrep** healthcheck) |
| `idx-api-worker` / `idx-api-scheduler` | Aliases of the above |

### 1.5 Command overrides (Option B only)

**Worker — staging:** `/bin/sh -lc 'exec php -d memory_limit=640M artisan queue:work --queue=${WORKER_QUEUES:-default} --sleep=1 --tries=3 --timeout=120'`

**Worker — production:** `/bin/sh -lc 'exec php -d memory_limit=512M artisan queue:work --queue=${WORKER_QUEUES:-default} --sleep=1 --tries=3 --timeout=120'`

**Scheduler — staging:** `php -d memory_limit=384M artisan schedule:work`  
**Scheduler — production:** `php -d memory_limit=256M artisan schedule:work`

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

## 5. Post-deployment

```bash
php artisan migrate --force
php artisan optimize
```

---

## 6. Route cache

Omit `route:cache` at image build when multiple `IDX_PLATFORM_HOSTS` share route names. Post-deploy `route:cache` only for single-host setups.

---

## 7. CPU and memory

See [AGENTS.md](../AGENTS.md). Container RAM > PHP `memory_limit` + ~300 MB on web.

---

## 8. Coolify checklist

1. Publish both FrankenPHP bases via GHA (§1.1).
2. GHCR read credentials on Coolify.
3. Three API apps: Dockerfile pack + `FRANKENPHP_BASE_IMAGE` + correct target (§1.3).
4. idx-images on **8080**, network alias **`idx-api`**.
5. Shared runtime env; deploy; run §5.

---

## 9. Troubleshooting

### `failed to resolve FRANKENPHP_BASE_IMAGE` / `pull access denied`

Set **`FRANKENPHP_BASE_IMAGE`** build arg and GHCR registry login on the Coolify server. Run GHA base workflow first (§1.1).

### `map has no entry for key "Health"`

Use build targets **`queue-worker`** / **`scheduler`** (Option A), or disable HTTP healthcheck when reusing the **`octane`** image (Option B).

### Deploy still runs `install-php-extensions` for a long time

Coolify is not using the GHCR base — check `FROM ${FRANKENPHP_BASE_IMAGE}` and the build arg.

### Octane: Caddy storage / autosave warnings

Non-fatal FrankenPHP/Caddy noise; **`/up`** on **8000** should still pass.

### `ViteManifestNotFoundException`

Vite stage failed during Coolify build — check Node stage logs.

### ARM on amd64 VPS

Use GHA **`linux/amd64`** bases and `docker buildx --platform linux/amd64` locally.

---

## 10. References

- [Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)
- [Build arguments](https://coolify.io/docs/builds/packs/dockerfile#build-arguments)
- [Pre / post deployment commands](https://coolify.io/docs/builds/packs/dockerfile#pre-post-deployment-commands)
