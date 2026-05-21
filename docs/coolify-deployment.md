# Coolify — production and staging (Go)

Run **Quantyra IDX API** on [Coolify](https://coolify.io/) using the **[Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)** with [`Dockerfile`](../Dockerfile) targets **`api`**, **`worker`**, and **`scheduler`**, plus [`Dockerfile.idx-images`](../Dockerfile.idx-images).

Use **separate Coolify projects** for staging and production, each with its own PostgreSQL database (or shared Patroni primary for multi-DC production — see §8).

**Queues:** PostgreSQL `jobs` table (no Redis). Deploy **web**, **worker(s)**, **scheduler**, and **idx-images**.

**Related:** [README.md](../README.md), [deployment-operations.md](deployment-operations.md), [go-cutover.md](go-cutover.md).

---

## 1. Applications per environment (single host)

| App | Dockerfile | Build target | Port / health |
|-----|------------|--------------|---------------|
| **idx-api-web** | `Dockerfile` | `api` | **8000** — `GET /healthz` |
| **idx-api-worker** | `Dockerfile` | `worker` | No HTTP — process health optional |
| **idx-api-scheduler** | `Dockerfile` | `scheduler` | No HTTP |
| **idx-images** | `Dockerfile.idx-images` | default | **8080** — `GET /health` |

**Build context:** repository root (`.`).

**Runtime env:** Same `DB_*`, `BRIDGE_*`, `SPARK_*`, `WORKER_QUEUES`, and public URLs on web, worker, and scheduler.

---

## 2. Worker configuration

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

Optional split during heavy replication:

| Replica | `WORKER_QUEUES` |
|---------|-----------------|
| Fetch | `default,bridge-sync-fetch,spark-sync-fetch` |
| Persist | `bridge-sync-persist,spark-sync-persist` |

**Post-cutover:** purge legacy Laravel jobs once:

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
```

---

## 3. Post-deploy (migrations and admin seed)

Run **once per schema change** against the Patroni primary (or staging DB), not from every Coolify app:

```bash
export GOOSE_DBSTRING="postgres://USER:PASS@HOST:5432/idx_api?sslmode=require"
goose -dir migrations postgres "$GOOSE_DBSTRING" up
# Or: make migrate
```

**Admin login** (one-time, from laptop or one-off container with `ADMIN_SEED_*` in env — not on runtime API/worker env):

```bash
export GOOSE_DBSTRING="..."   # same DSN
export ADMIN_SEED_EMAIL=...
export ADMIN_SEED_PASSWORD=...
export ADMIN_SEED_NAME="Quantyra Admin"
make seed-admin
```

Notify customers to **re-issue API keys** from `/dashboard` after Go cutover.

---

## 4. idx-images

[`Dockerfile.idx-images`](../Dockerfile.idx-images), port **8080**. Upstream **`idx-api:8000`** on the shared Docker network.

On each Coolify server, set the API container network alias to **`idx-api`** (required by [`nginx.idx-images.conf`](../nginx.idx-images.conf)).

---

## 5. Resources (starting points)

| Service | CPU | RAM |
|---------|-----|-----|
| Web (`api`) | 0.5–1.0 | 512–1024 MB |
| Worker | 0.25–0.5 each | 512–1024 MB |
| Scheduler | 0.1–0.25 | 256–384 MB |
| idx-images | 0.1–0.25 | 128–256 MB |

Reserve host memory for PostgreSQL if co-located. Patroni cluster nodes are **not** on these Coolify hosts in the multi-DC layout (Tailscale only).

---

## 6. Spark / Bridge outbound

Workers and web need HTTPS to Bridge and Spark hosts (`BRIDGE_HOST`, `SPARK_REPLICATION_HOST`, `SPARK_API_HOST`). See [spark/idx-api-integration.md](spark/idx-api-integration.md).

---

## 7. Scheduler cluster leadership (required for 2+ schedulers)

Cron overlap protection (`withoutOverlap`) is **in-process only**. Two scheduler containers without a cluster lock will **double-enqueue** replication kickoff, proxy cache purge, etc.

The Go scheduler uses a **PostgreSQL session advisory lock** on a dedicated connection:

| Variable | Default | Purpose |
|----------|---------|---------|
| `SCHEDULER_LEADER_LOCK_ID` | `913374211` | `pg_try_advisory_lock` key (int64) |
| `SCHEDULER_STANDBY_POLL_SECONDS` | `15` | Standby retry interval |

**Logs:** `scheduler leader acquired` (runs cron) vs `scheduler standby, waiting for leader lock`.

Deploy **two** scheduler apps (NYC + ATL); normally **one** holds the lock. The other stays standby for failover when the leader disconnects (lock released).

**Warning:** Do not run two schedulers on Patroni without this lock — even on a single host.

---

## 8. Multi-DC production (NYC + ATL)

Production spans **re-db** (NYC) and **re-node-02** (ATL) with **shared Patroni PostgreSQL** over **Tailscale**. Phase 1: **all apps use the Patroni primary** only (no read replicas for API yet).

### Topology

```
Clients → Cloudflare Geo LB
            ├─ Pool NYC → re-db  → idx-api-nyc, idx-images-nyc
            └─ Pool ATL → re-node-02 → idx-api-atl, idx-images-atl
                    │
                    └─ Tailscale → Patroni primary (writes + queue + cron)
```

| Role | NYC (re-db) | ATL (re-node-02) |
|------|-------------|------------------|
| API | 1× `api` :8000 | 1× `api` :8000 |
| Worker | 2× `worker` | 2× `worker` |
| Scheduler | 1× `scheduler` | 1× `scheduler` (advisory lock — one leader) |
| idx-images | 1× :8080 → local `idx-api` | 1× :8080 → local `idx-api` |

**Total: 10 Coolify applications** (create **one app per server**, not one app with replicas on one host).

### Coolify app matrix

| Coolify app (suggested) | Server | Dockerfile | Target | Port / health |
|-------------------------|--------|------------|--------|---------------|
| `idx-api-nyc` | re-db | `Dockerfile` | `api` | 8000 — `GET /healthz` |
| `idx-api-atl` | re-node-02 | `Dockerfile` | `api` | 8000 |
| `idx-worker-nyc-1` | re-db | `Dockerfile` | `worker` | — |
| `idx-worker-nyc-2` | re-db | `Dockerfile` | `worker` | — |
| `idx-worker-atl-1` | re-node-02 | `Dockerfile` | `worker` | — |
| `idx-worker-atl-2` | re-node-02 | `Dockerfile` | `worker` | — |
| `idx-scheduler-nyc` | re-db | `Dockerfile` | `scheduler` | — |
| `idx-scheduler-atl` | re-node-02 | `Dockerfile` | `scheduler` | — |
| `idx-images-nyc` | re-db | `Dockerfile.idx-images` | default | 8080 — `GET /health` |
| `idx-images-atl` | re-node-02 | `Dockerfile.idx-images` | default | 8080 |

Attach the **same shared environment** (Coolify project/team env) to all ten apps.

### Shared environment (Patroni primary via Tailscale)

```env
DB_HOST=<patroni-primary-hostname-on-tailscale>
DB_PORT=5432
DB_DATABASE=idx_api
DB_USERNAME=...
DB_PASSWORD=...
DB_SSLMODE=require

WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
SCHEDULER_LEADER_LOCK_ID=913374211
SCHEDULER_STANDBY_POLL_SECONDS=15

BRIDGE_API_KEY=...
SPARK_ACCESS_TOKEN=...
IDX_API_PUBLIC_URL=https://idx-api.quantyralabs.cc
IDX_IMAGES_PUBLIC_URL=https://idx-images.quantyralabs.cc
IDX_PLATFORM_URL=https://idx.quantyralabs.cc
```

**Image cache:** Each API has local disk at `IMAGE_CACHE_PATH` (default `/var/cache/geoidx/images`). Geo-routed APIs do **not** share cache; extra MLS origin fetches on miss are acceptable.

**Build:** Git deploy from this repo, or GHCR images from [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml).

### Tailscale + Patroni verification (both Coolify servers)

1. Install Tailscale on **re-db** and **re-node-02**; allow routes to the Patroni primary VIP/hostname.
2. From **each** server (SSH or Coolify one-off shell):

```bash
export DB_HOST=... DB_PORT=5432 DB_DATABASE=idx_api DB_USERNAME=... DB_PASSWORD=... DB_SSLMODE=require
./scripts/verify-patroni-connectivity.sh
```

3. After API deploy, optionally: `API_URL=https://<that-dc-api-host> ./scripts/verify-patroni-connectivity.sh` (checks `GET /readyz` / PostGIS).

ATL workers polling `jobs` over Tailscale is expected; watch `readyz` timeouts if latency is high.

### Deploy order

1. Tailscale + `psql` from both servers.
2. Merge/deploy images with scheduler advisory lock.
3. Create 10 Coolify apps + shared env.
4. `goose up` once on Patroni primary.
5. `make seed-admin` once (not on runtime API env).
6. Start **workers** (all 4) → **schedulers** (both; confirm one leader in logs) → **APIs** → **idx-images**.
7. Cloudflare geo LB (below).
8. Smoke: `/healthz`, `/readyz`, workers drain `jobs`, replication kickoff in logs, purge legacy `CallQueuedHandler` rows if needed.

### Cloudflare load balancing (geo)

Use **Cloudflare Load Balancer** or **Geo Steering** on public hostnames:

| Hostname | Pool NYC | Pool ATL | Health check |
|----------|----------|----------|--------------|
| `idx-api.quantyralabs.cc` | re-db → `idx-api-nyc` :8000 | re-node-02 → `idx-api-atl` :8000 | `GET /healthz` |
| `idx-images.quantyralabs.cc` | re-db → `idx-images-nyc` :8080 | re-node-02 → `idx-images-atl` :8080 | `GET /health` |

Terminate TLS at Cloudflare or Coolify/Traefik; health checks must reach the app port through the proxy.

**Why per-DC idx-images:** [`nginx.idx-images.conf`](../nginx.idx-images.conf) proxies to **local** `idx-api:8000`. Without ATL idx-images, image traffic would cross regions to NYC.

Coolify’s single-host Traefik LB is insufficient for two datacenters; use Cloudflare (recommended) or NYC Traefik with remote backends (adds cross-region hops).

### Multi-DC resources (12 vCPU / 48 GB per host)

| Container | CPU limit | RAM limit |
|-----------|-----------|-----------|
| API (each) | 0.5–1.0 | 512–1024 MB |
| Worker (each) | 0.25–0.5 | 512–1024 MB |
| Scheduler (each) | 0.1–0.25 | 256–384 MB |
| idx-images (each) | 0.1–0.25 | 128–256 MB |

Leave headroom for Coolify; Patroni runs elsewhere.

---

## 9. Phase 2 — Patroni read replicas (optional)

**Phase 1:** point **all** containers at the **primary** only.

Replicas can offload **API read** paths later (PostGIS search, comps mirror `SELECT`, cache Gets). They are **not** safe for:

| Consumer | Use primary? |
|----------|----------------|
| Workers (`FOR UPDATE SKIP LOCKED`, persist/upsert) | **Yes — primary only** |
| Schedulers (enqueue + advisory lock) | **Yes — primary only** |
| API writes (audit, domain/token, cache Put) | **Yes — primary only** |

**Future:** `DB_READ_HOST` / second pool in `internal/repository/db.go`; route explicit read paths only; PgBouncer or HAProxy `replica` pool on a separate Tailscale port. Monitor `pg_stat_replication` lag.

Geo benefit is largest when an API region is **near** a replica in that region.

---

## 10. Risks (multi-DC)

| Risk | Mitigation |
|------|------------|
| Dual schedulers without lock | Advisory lock (§7) |
| ATL → Patroni latency | Acceptable for queue poll; tune timeouts |
| Split image cache | Geo-route idx-images to local DC |
| Patroni failover during job | Workers retry; kickoff on next minute |
| Read replica lag | Phase 1: primary only |

---

## 11. Local smoke build

```bash
docker build -f Dockerfile --target api -t idx-api:local .
docker build -f Dockerfile --target worker -t idx-api-worker:local .
docker run --rm -p 8000:8000 --env-file .env idx-api:local
```

---

## Legacy note

Older docs referenced FrankenPHP/Octane (`Dockerfile.production`, `php artisan queue:work`). The **current** stack is **Go binaries** in [`Dockerfile`](../Dockerfile). Remove FrankenPHP base image variables from Coolify if migrating an existing project.
