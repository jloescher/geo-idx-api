# Coolify — production and staging

This guide describes how to run **Quantyra IDX API** on [Coolify](https://coolify.io/) using the **[Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)** (not Nixpacks). Use **two Coolify projects** (or two clearly separated groups of resources): one for **production** and one for **staging**, each with its own PostgreSQL database and environment variables.

**Queues:** this app uses Laravel’s **`database`** queue driver (`jobs` table in PostgreSQL). There is **no Redis** requirement for queues. Deploy **one** web (Octane), **one or more** queue workers, **exactly one** scheduler, plus the **idx-images** edge. See §2.5 for scaling workers.

**Related:** [Deployment & operations](deployment-operations.md) (Docker, Compose, Dokploy), root [README.md](../README.md), [AGENTS.md](../AGENTS.md) (Docker tables and PHP memory notes).

---

## 1. Repository and build settings (every application)

| Coolify field | Value |
|---------------|--------|
| **Build pack** | Dockerfile |
| **Base directory** | `/` (repository root; use a subdirectory only in a monorepo) |
| **Branch** | Production: `main` / `production` (your release branch). Staging: `staging` / `develop` (your pre-release branch). |

Each row in the tables below is a **separate Coolify application** (or an equivalent service in a single Compose stack). Repeat the same **repository** and **base directory**; only the Dockerfile, **Docker build target**, port, and env differ.

### 1.1 Docker build target vs application name

Coolify’s **Docker Build Target** is passed to `docker build --target=…` and must match a **`FROM … AS <stage>`** name in the Dockerfile (for example **`octane`**, **`queue-worker`**, **`scheduler`**). It is **not** automatically the same as the Coolify **application name** (for example `idx-api-worker`).

| Mistake | Typical error |
|---------|----------------|
| Build target **`idx-api-worker`** when only **`queue-worker`** exists | `target stage "idx-api-worker" could not be found` |

**Canonical targets:** **`octane`** (web), **`queue-worker`**, **`scheduler`**.

**Alias stages (forgiving):** [Dockerfile.staging](../Dockerfile.staging) and [Dockerfile.production](../Dockerfile.production) also expose **`idx-api-worker`** (same image as **`queue-worker`**) and **`idx-api-scheduler`** (same as **`scheduler`**). Prefer the canonical names in new Coolify apps so logs and docs stay clear.

### 1.2 Vite / `public/build` on image build

The API Dockerfiles **never** copy `public/build` from the git build context (it is listed in [`.dockerignore`](../.dockerignore)). Each build runs **`npm ci`** and **`npm run build`** inside a **Node** stage after **`composer install`**, then **`COPY --from=… ./public/build`** into the PHP image so `@vite()` and the marketing home have **`manifest.json`** at runtime. Do not expect to “bring your own” committed `public/build` from the repo on deploy.

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
| **Docker Build Target** | `queue-worker` (optional alias: **`idx-api-worker`** — same image) |
| **Port** | None exposed publicly (no inbound HTTP). |
| **Command** | Use image default (`queue:work` with `php -d memory_limit=512M`). |

### 2.3 API scheduler

| Field | Value |
|-------|--------|
| **Name** | e.g. `idx-api-scheduler` |
| **Dockerfile** | `Dockerfile.production` |
| **Docker Build Target** | `scheduler` (optional alias: **`idx-api-scheduler`** — same image) |
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

**Networking:** `idx-images` must resolve **`idx-api`** to the **web** container on the **same Docker network** (same Coolify project / stack). If Coolify assigns another internal hostname, either set a **network alias** / service name **`idx-api`** for the Octane app or change the upstream host in [nginx.idx-images.conf](../nginx.idx-images.conf) and rebuild the idx-images image.

[nginx.idx-images.conf](../nginx.idx-images.conf) uses Docker’s embedded DNS (**`resolver 127.0.0.11`**) and a **variable `proxy_pass`** so Nginx starts even if **`idx-api`** appears on the network slightly later than idx-images (rolling updates). You still need a **shared network** and a resolvable name for the API container; the resolver does not replace correct service discovery.

### 2.5 Multiple queue workers (scaling)

Laravel’s **`database`** driver is safe with **multiple concurrent `queue:work` processes**: each worker claims jobs using the database, so you can scale horizontally for throughput (GIS jobs, listing cache refresh, etc.).

**Preferred (Coolify replicas):** On the **queue worker** application (same Dockerfile and **`queue-worker`** / **`idx-api-worker`** target), open **Configuration** → **Advanced** (or **Resources** / scaling, depending on Coolify version) and set **Replicas** / **Instances** to **`2`** or higher. Coolify runs multiple containers from the same service; each runs the image default `artisan queue:work`. Keep **identical environment variables** on that app so every replica uses the same `DB_*`, `APP_KEY`, and `QUEUE_CONNECTION=database`.

**If replicas are unavailable** (older Coolify, or `container_name` blocking scaling): create **additional Coolify applications** cloned from the worker app—same repository, **`Dockerfile.production`** or **`Dockerfile.staging`**, build target **`queue-worker`**, same env and branch—so each app is one worker process. Name them distinctly (e.g. `idx-api-worker-2`) for logs only; DNS names do not need to match for workers.

**Scheduler:** Run **only one** replica of the **`scheduler`** service. Multiple `schedule:work` processes can run the same scheduled tasks in duplicate unless you design around it.

**Queues env:** `WORKER_QUEUES` (e.g. `default` or `default,gis`) applies to **each** worker replica—all replicas listen to those queues and compete for jobs, which is usually what you want.

**Sizing:** Multiply worker **RAM** and **vCPU** by replica count (see §7). Watch PostgreSQL **`max_connections`** if you add many workers and other services on the same database.

---

## 3. Staging — same layout, different Dockerfile

Use **`Dockerfile.staging`** for all **three** API applications (targets **`octane`**, **`queue-worker`**, **`scheduler`**). Staging includes **Xdebug** (trigger mode); use **`Dockerfile.idx-images`** unchanged for the fourth service (same Nginx image as production).

| Application | Dockerfile | Build target | Port |
|-------------|------------|--------------|------|
| Web | `Dockerfile.staging` | `octane` | **8000** |
| Worker | `Dockerfile.staging` | `queue-worker` | — |
| Scheduler | `Dockerfile.staging` | `scheduler` | — |
| idx-images | `Dockerfile.idx-images` | *(default)* | **8080** |

Scaling **multiple workers** (replicas or duplicate worker apps) matches production; see **§2.5**.

Point domains (e.g. `staging-idx-api.*`, `staging-idx-images.*`) at these services in Coolify’s **Domains** / reverse proxy UI.

---

## 4. Environment variables (production vs staging)

Set variables in Coolify **per application** for the **web**, **each worker**, and **scheduler** (all worker replicas and the scheduler need the **same** `DB_*`, `APP_KEY`, `QUEUE_CONNECTION`, and app URLs as the web process). Copy from root **`.env.example`** and trim to what you use.

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

Schema inventory, PostGIS requirements for the listings mirror, and the legacy `dropIfExists` migration are documented in **[database-migrations.md](database-migrations.md)**.

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
2. Create **four** applications from the repo with the Dockerfile pack (web, worker, scheduler, idx-images). If you add **extra worker apps** instead of replicas, create one application per worker process (§2.5).  
3. Set **ports** 8000 (API web) and 8080 (idx-images).  
4. Paste **environment variables** (shared across web/worker/scheduler for API).  
5. Ensure **network / service name** so `idx-api` resolves from `idx-images`.  
6. Attach **domains** and TLS.  
7. **Deploy**, then run **post-deploy** commands (§5).  
8. Optionally scale **workers**: increase **replicas** on the worker app or add duplicate worker apps — see **§2.5** (each process runs `queue:work`; sum RAM and DB connections when sizing).

---

## 9. Troubleshooting (deploy / rolling updates)

### `storage/logs/laravel.log` permission denied (container unhealthy)

The API image runs Octane as **`www-data`**. If **`storage/logs`** is missing (it is excluded from the Docker build context via `.dockerignore`) or root-owned files exist under **`storage/`** / **`bootstrap/cache/`** after build-time `php artisan` commands, Laravel cannot log and Octane may fail writing its process state — the **Dockerfile healthcheck** then never passes.

**Fix:** Use an image that includes the post-artisan `mkdir` + `chown`/`chmod` step (see `Dockerfile.staging` / `Dockerfile.production`). Rebuild and redeploy.

**If you mount a persistent volume on `storage/`** in Coolify, ensure the mount is writable by the **`www-data`** user the image uses (same uid/gid as in `Dockerfile.staging`), or `chown` the volume once after first deploy.

### Coolify: `APP_ENV=staging` build-time warning

Coolify may inject `APP_ENV` at **build** time. For Laravel, prefer **`APP_ENV` as runtime-only** in Coolify (or use `local` during build if you must pass it at build time), per Coolify’s own hint — see [environment variables](https://coolify.io/docs/builds/packs/dockerfile#environment-variables).

### Rolling updates stuck on healthcheck

Coolify expects healthchecks that can probe HTTP with **`curl` or `wget`** ([troubleshooting](https://coolify.io/docs/troubleshoot/applications/no-available-server)). The API Dockerfile uses **`curl -fsS http://127.0.0.1:8000/up`** so the proxy can verify the app without bootstrapping a full `php artisan` invocation.

### idx-images: `host not found in upstream "idx-api:8000"` or container `(unhealthy)`

**Symptoms:** Nginx logs **`[emerg] host not found in upstream`** and the container never passes Coolify’s rolling update healthcheck.

**Causes:** idx-images and the **Octane** API are on **different** Docker networks, or the API service has **no DNS name `idx-api`** on that network, or the idx-images app was mis-pointed at the **PHP** Dockerfile instead of **`Dockerfile.idx-images`**.

**Fix:** Put both services in the **same** Coolify project/stack, and ensure the web API is reachable as **`idx-api`** (display name and/or internal hostname / network alias, depending on Coolify version). Rebuild idx-images after changing [nginx.idx-images.conf](../nginx.idx-images.conf). The image uses **runtime DNS** for the upstream; see §2.4.

### Octane: `ViteManifestNotFoundException` in the container

The runtime image must contain **`public/build/manifest.json`** produced **during `docker build`**, not from git (see §1.2). If you see this error after deploy, the image was built without the Node/Vite stage succeeding, or an old image layer was reused incorrectly — rebuild the **web** application image from the current Dockerfile and redeploy.

## 10. Official Coolify references

- [Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile) — port **3000** default; set **8000** / **8080** explicitly.  
- [Environment variables](https://coolify.io/docs/builds/packs/dockerfile#environment-variables) — UI and build-time injection.  
- [Pre / post deployment commands](https://coolify.io/docs/builds/packs/dockerfile#pre-post-deployment-commands) — optional `migrate`, `optimize` automation.
