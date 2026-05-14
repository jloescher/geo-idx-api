# Coolify — production and staging

This guide describes how to run **Quantyra IDX API** on [Coolify](https://coolify.io/) using the **[Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)** (not Nixpacks). Use **two Coolify projects** (or two clearly separated groups of resources): one for **production** and one for **staging**, each with its own PostgreSQL database and environment variables.

**Queues:** this app uses Laravel’s **`database`** queue driver (`jobs` table in PostgreSQL). There is **no Redis** requirement for queues. Deploy **three** API processes (web + worker + scheduler) plus the **idx-images** edge.

**Related:** [Deployment & operations](deployment-operations.md) (Docker, Compose, Dokploy), root [README.md](../README.md), [AGENTS.md](../AGENTS.md) (Docker tables and PHP memory notes).

---

## 1. Repository and build settings (every application)

| Coolify field | Value |
|---------------|--------|
| **Build pack** | Dockerfile |
| **Base directory** | `/` (repository root; use a subdirectory only in a monorepo) |
| **Branch** | Production: `main` / `production` (your release branch). Staging: `staging` / `develop` (your pre-release branch). |

Each row in the tables below is a **separate Coolify application** (or an equivalent service in a single Compose stack). Repeat the same **repository** and **base directory**; only the Dockerfile, **Docker build target**, port, and env differ.

---

## 2. Production — four applications

### 2.1 API web (Octane)

| Field | Value |
|-------|--------|
| **Name (recommended)** | `idx-api` — so other containers can reach `http://idx-api:8000` (required by [nginx.idx-images.conf](../nginx.idx-images.conf)). If your platform prefixes names, set the **internal / Docker service hostname** to `idx-api`, or change the nginx `upstream` to match. |
| **Dockerfile** | `Dockerfile.production` |
| **Docker Build Target** | `octane` |
| **Port** | **8000** (Coolify defaults to **3000**; change it.) |
| **Healthcheck** | HTTP `GET /up` on port 8000, or command `php artisan octane:status` (image supports both). |

### 2.2 API queue worker

| Field | Value |
|-------|--------|
| **Name** | e.g. `idx-api-worker` |
| **Dockerfile** | `Dockerfile.production` |
| **Docker Build Target** | `queue-worker` |
| **Port** | None exposed publicly (no inbound HTTP). |
| **Command** | Use image default (`queue:work` with `php -d memory_limit=512M`). |

### 2.3 API scheduler

| Field | Value |
|-------|--------|
| **Name** | e.g. `idx-api-scheduler` |
| **Dockerfile** | `Dockerfile.production` |
| **Docker Build Target** | `scheduler` |
| **Port** | None exposed publicly. |
| **Command** | Use image default (`schedule:work`). |

### 2.4 Image edge (Nginx)

| Field | Value |
|-------|--------|
| **Name** | e.g. `idx-images` |
| **Dockerfile** | `Dockerfile.idx-images` |
| **Docker Build Target** | *(leave default / final stage)* |
| **Port** | **8080** |
| **Healthcheck** | `GET /health` on port **8080**. |

**Networking:** `idx-images` must resolve **`idx-api`** to the **web** container on the **same Docker network** (same Coolify project / stack). If Coolify assigns another hostname, update `upstream idx_api_images` in `nginx.idx-images.conf` and rebuild, or align the web service name to `idx-api`.

---

## 3. Staging — same layout, different Dockerfile

Use **`Dockerfile.staging`** for all **three** API applications (targets **`octane`**, **`queue-worker`**, **`scheduler`**). Staging includes **Xdebug** (trigger mode); use **`Dockerfile.idx-images`** unchanged for the fourth service (same Nginx image as production).

| Application | Dockerfile | Build target | Port |
|-------------|------------|--------------|------|
| Web | `Dockerfile.staging` | `octane` | **8000** |
| Worker | `Dockerfile.staging` | `queue-worker` | — |
| Scheduler | `Dockerfile.staging` | `scheduler` | — |
| idx-images | `Dockerfile.idx-images` | *(default)* | **8080** |

Point domains (e.g. `staging-idx-api.*`, `staging-idx-images.*`) at these services in Coolify’s **Domains** / reverse proxy UI.

---

## 4. Environment variables (production vs staging)

Set variables in Coolify **per application** for the three API services (worker and scheduler need the **same** `DB_*`, `APP_KEY`, `QUEUE_CONNECTION`, and app URLs as the web process). Copy from root **`.env.example`** and trim to what you use.

### 4.1 Core Laravel

| Variable | Production | Staging |
|----------|------------|---------|
| `APP_ENV` | `production` | `staging` |
| `APP_DEBUG` | `false` | `true` (optional; aids debugging) |
| `APP_KEY` | **Required** (unique per environment). | **Required** |
| `APP_URL` | Platform app URL (e.g. marketing + dashboard host). | Staging platform URL |
| `LOG_LEVEL` | `info` or `warning` | `debug` or `info` |

### 4.2 Database and queue (PostgreSQL only)

| Variable | Notes |
|----------|--------|
| `DB_CONNECTION` | `pgsql` |
| `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD` | Point each environment to its **own** Postgres database. |
| `QUEUE_CONNECTION` | **`database`** (required for the `jobs` table). |
| `WORKER_QUEUES` | Default `default`. If you use a dedicated GIS queue, e.g. `default,gis` (see `GIS_QUEUE` in `config/gis.php`). |

Also set `SESSION_DRIVER`, `CACHE_STORE` as in `.env.example` (commonly `database`).

### 4.3 Quantyra public URLs

Align with your real hostnames (see `.env.example`):

- `API_URL` / `IDX_API_PUBLIC_URL` — API + widgets host  
- `IMAGE_URL` / `IDX_IMAGES_PUBLIC_URL` — image proxy host  
- `IDX_PLATFORM_URL` — app / marketing host  
- `IDX_PLATFORM_HOSTS`, `IDX_API_HOSTS` — comma-separated hosts for `Route::domain()` (staging lists **only** staging hosts if you want `php artisan route:cache` to work; see §6).

### 4.4 Integrations (set per environment)

- **Bridge MLS:** `BRIDGE_API_KEY`, `BRIDGE_HOST`, `BRIDGE_DATASET`, etc.  
- **Telescope / Pulse / Debugbar:** usually **off** or gated in production; Telescope may be on in staging (`TELESCOPE_ENABLED=true`).

### 4.5 Xdebug (staging only)

The staging image sets `xdebug.client_host=host.docker.internal`. For remote debugging from your laptop through Coolify’s VPS, you may need to set **`XDEBUG_CONFIG`** or Coolify-equivalent overrides (e.g. `client_host=<your-ip>`) so the debugger reaches your IDE.

---

## 5. Post-deployment commands

Run inside the **web** container (or any API container with the same code and env), after each release:

```bash
php artisan migrate --force
```

Refresh caches if you changed config or `.env`:

```bash
php artisan optimize
```

---

## 6. Route cache (`route:cache`)

The **production** Docker image runs `config:cache` and `view:cache` at build time but **not** `route:cache`, because `routes/web.php` registers the **same route names** on each entry in `IDX_PLATFORM_HOSTS` / `IDX_API_HOSTS`, which breaks route serialization.

- **After deploy:** run `php artisan route:cache` **only** if that environment uses a **single** platform host (or unique route names per domain).  
- Otherwise omit `route:cache` and rely on file-based route loading.

---

## 7. CPU and memory (Coolify resource limits)

Set limits in Coolify per service (**Resources** / **Limits** / **Advanced**, depending on version). Container memory should exceed PHP **`memory_limit`** plus headroom (~300 MB on the web container for opcache).

| Service | vCPU (start) | RAM (start) |
|---------|----------------|-------------|
| Web (`octane`) | 0.5–1.0 | 1024–1536 MB |
| Worker (`queue-worker`) | 0.25–0.5 per replica | 512–1024 MB |
| Scheduler (`scheduler`) | 0.1–0.25 | 256–384 MB |
| idx-images | 0.1–0.25 | 128–256 MB |

**VPS:** minimum **2 vCPU / 4 GB** is tight for API + worker + scheduler + Postgres on one host; **4 vCPU / 8 GB+** is more comfortable for production. Reserve **1–2 GB** for PostgreSQL if it runs on the same server. Staging often needs **more RAM** than production if Xdebug and Telescope are active.

PHP **`memory_limit`** is set in the image `CMD`: web **256M**, worker **512M**, scheduler **256M** (staging Dockerfile adds **+128M** per role).

---

## 8. Coolify UI checklist (quick)

1. Create **PostgreSQL** (or attach external) for the environment.  
2. Create **four** applications from the repo with the Dockerfile pack.  
3. Set **ports** 8000 (API web) and 8080 (idx-images).  
4. Paste **environment variables** (shared across web/worker/scheduler for API).  
5. Ensure **network / service name** so `idx-api` resolves from `idx-images`.  
6. Attach **domains** and TLS.  
7. **Deploy**, then run **post-deploy** commands (§5).  
8. Optionally scale **workers** by increasing replicas in Coolify (each replica runs `queue:work`; sum RAM when sizing).

---

## 9. Official Coolify references

- [Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile) — port **3000** default; set **8000** / **8080** explicitly.  
- [Environment variables](https://coolify.io/docs/builds/packs/dockerfile#environment-variables) — UI and build-time injection.  
- [Pre / post deployment commands](https://coolify.io/docs/builds/packs/dockerfile#pre-post-deployment-commands) — optional `migrate`, `optimize` automation.
