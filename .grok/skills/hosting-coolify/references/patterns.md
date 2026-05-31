# Coolify Hosting Patterns

## Contents
- Application Matrix
- Resource Allocation
- Dockerfile Target Patterns
- Networking and DNS
- Environment Configuration
- Anti-Patterns

## Application Matrix

### Single-Host (Staging)

Four Coolify applications in one project:

| App | Dockerfile | Target | Port | Health |
|-----|-----------|--------|------|--------|
| idx-api-web | `Dockerfile` | `api` | 8000 | `GET /healthz` |
| idx-api-worker | `Dockerfile` | `worker` | — | process health optional |
| idx-api-scheduler | `Dockerfile` | `scheduler` | — | — |
| idx-images | `Dockerfile.idx-images` | default | 8080 | `GET /health` |

Build context for all: repository root (`.`).

### Multi-DC (Production — 10 Apps)

One app per server, per role. See the **deploy-patroni** skill for database topology.

| App | Server | Target | Port |
|-----|--------|--------|------|
| idx-api-nyc | re-db | `api` | 8000 |
| idx-api-atl | re-node-02 | `api` | 8000 |
| idx-worker-nyc-1 | re-db | `worker` | — |
| idx-worker-nyc-2 | re-db | `worker` | — |
| idx-worker-atl-1 | re-node-02 | `worker` | — |
| idx-worker-atl-2 | re-node-02 | `worker` | — |
| idx-scheduler-nyc | re-db | `scheduler` | — |
| idx-scheduler-atl | re-node-02 | `scheduler` | — |
| idx-images-nyc | re-db | `Dockerfile.idx-images` | 8080 |
| idx-images-atl | re-node-02 | `Dockerfile.idx-images` | 8080 |

## Resource Allocation

### Single-Host Starting Points

| Service | CPU | RAM |
|---------|-----|-----|
| Web (`api`) | 0.5–1.0 | 512–1024 MB |
| Worker | 0.25–0.5 | 512–1024 MB |
| Scheduler | 0.1–0.25 | 256–384 MB |
| idx-images | 0.1–0.25 | 128–256 MB |

Reserve host memory for PostgreSQL if co-located. Patroni cluster nodes run on separate hosts in multi-DC (Tailscale only).

### Worker Scaling Guidance

Scale workers by queue depth, not CPU. Monitor `jobs` table:
- Replication backlogs → add persist workers (`bridge-sync-persist`, `spark-sync-persist`)
- Slow fetches → add fetch workers (`bridge-sync-fetch`, `spark-sync-fetch`)
- General queue lag → add default worker

## Dockerfile Target Patterns

The `Dockerfile` compiles all three binaries in a single `build` stage, then copies the relevant binary into each target:

```dockerfile
FROM golang:1.25-alpine AS build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler
```

Each runtime target is a minimal Alpine image. Only `api` has a `HEALTHCHECK`:

```dockerfile
FROM alpine:3.21 AS api
COPY --from=build /out/api /usr/local/bin/api
USER nobody
EXPOSE 8000
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8000/healthz || exit 1
```

Worker and scheduler have no HTTP port — no `HEALTHCHECK` directive.

## Networking and DNS

### Container Alias (Required)

`nginx.idx-images.conf` proxies to `idx-api:8000` via Docker DNS. On each Coolify server, set the API container's network alias to **`idx-api`**:

```nginx
resolver 127.0.0.11 ipv6=off valid=10s;
set $idx_api_upstream "idx-api:8000";
proxy_pass http://$idx_api_upstream;
```

### WARNING: Hardcoded Upstream IP

**The Problem:**

```nginx
# BAD — nginx caches DNS at startup
proxy_pass http://172.18.0.5:8000;
```

**Why This Breaks:** During rolling updates, the API container gets a new IP. Nginx continues proxying to the old IP until nginx itself restarts, causing 502s.

**The Fix:**

```nginx
# GOOD — resolves at request time via Docker DNS
resolver 127.0.0.11 ipv6=off valid=10s;
set $idx_api_upstream "idx-api:8000";
proxy_pass http://$idx_api_upstream;
```

## Environment Configuration

### Shared Environment (All Apps)

All four (or ten) apps must share `DB_*`, `BRIDGE_*`, `SPARK_*`, and public URL variables. Use Coolify's **project-level** or **team-level** shared environment.

| Variable | Required By | Purpose |
|----------|-------------|---------|
| `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD` | All | PostgreSQL connection |
| `WORKER_QUEUES` | Worker only | Queue names to poll |
| `SCHEDULER_LEADER_LOCK_ID` | Scheduler only | Advisory lock key |
| `BRIDGE_API_KEY` | API + Worker | Bridge MLS auth |
| `SPARK_ACCESS_TOKEN` | API + Worker | Spark MLS auth |
| `IDX_API_PUBLIC_URL` | API | Public-facing URL |
| `IDX_IMAGES_PUBLIC_URL` | idx-images | Public image URL |

### Admin Seed (One-Time, NOT in Runtime Env)

```bash
export GOOSE_DBSTRING="postgres://..."
export ADMIN_SEED_EMAIL=...
export ADMIN_SEED_PASSWORD=...
make seed-admin
```

NEVER put `ADMIN_SEED_*` in Coolify runtime environment — run from a laptop or one-off container.

## Anti-Patterns

### WARNING: Two Schedulers Without Advisory Lock

**The Problem:** Running two scheduler containers without `SCHEDULER_LEADER_LOCK_ID` causes double-enqueue of every cron job.

**Why This Breaks:** Cron overlap protection (`withoutOverlap`) is in-process only. Two processes have no awareness of each other. Each scheduler enqueues its own `mls.replication_kickoff`, `mls.proxy_cache_purge`, etc., doubling work and API calls.

**The Fix:** Set `SCHEDULER_LEADER_LOCK_ID=913374211` on both schedulers. One acquires the lock and runs crons; the other logs `scheduler standby, waiting for leader lock`.

### WARNING: Per-App Environment Drift

**The Problem:** Copying environment variables into each Coolify app individually leads to stale values when one app is updated and others are not.

**The Fix:** Use Coolify's shared environment at the project or team level. All apps reference the same source of truth.