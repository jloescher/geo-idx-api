# Deployment Patterns Reference

## Contents
- Multi-Target Docker Build
- Environment Configuration
- Worker Queue Splitting
- Scheduler Leader Election
- Image Edge Proxy
- Anti-Patterns

## Multi-Target Docker Build

The project uses a single `Dockerfile` with three targets. All three binaries compile in one `CGO_ENABLED=0` build stage:

```dockerfile
# Existing: Dockerfile build stage
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler
```

**DO:** Use `--target` to select which binary each Coolify app runs.

**DON'T:** Build separate Dockerfiles per service — the multi-target pattern avoids duplicate build stages and ensures all binaries share the same dependency versions.

### Coolify Build Pack Settings

- **Build pack:** Dockerfile
- **Build context:** `.` (repo root)
- **Dockerfile location:** `Dockerfile`
- **Target:** `api`, `worker`, or `scheduler`

For idx-images: separate `Dockerfile.idx-images`, no target, port 8080.

## Environment Configuration

All services share the same environment variables. Attach Coolify project-level or team-level shared env to all apps.

**Required for all services:**

```env
DB_HOST=<patroni-primary-hostname>
DB_PORT=5432
DB_DATABASE=idx_api
DB_USERNAME=...
DB_PASSWORD=...
DB_SSLMODE=require
```

**Worker-specific:**

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

**Scheduler-specific (multi-DC only):**

```env
SCHEDULER_LEADER_LOCK_ID=913374211
SCHEDULER_STANDBY_POLL_SECONDS=15
```

### WARNING: Admin Seed in Runtime Env

**The Problem:** Placing `ADMIN_SEED_*` variables in the API/worker/scheduler runtime environment.

**Why This Breaks:** The seed command (`make seed-admin`) is a one-time CLI operation, not a runtime concern. Exposing admin credentials in runtime env increases blast radius if env leaks.

**The Fix:** Run `make seed-admin` from a laptop or one-off container with `ADMIN_SEED_*` set — never on running API/worker/scheduler containers.

## Worker Queue Splitting

### Combined Worker (dev / low traffic)

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

### Split Workers (production catch-up)

| Deployment | `WORKER_QUEUES` | Scale |
|------------|-----------------|-------|
| default-worker | `default` | 1× |
| fetch-worker | `bridge-sync-fetch,spark-sync-fetch` | 2× |
| persist-worker | `bridge-sync-persist,spark-sync-persist` | 2–4× |

**Why split:** During replication catch-up, fetch workers do HTTP I/O (network-bound) while persist workers do PostgreSQL upserts (CPU/disk-bound). Splitting prevents fetch backlogs from blocking persist and vice versa.

Workers use `FOR UPDATE SKIP LOCKED` with fair queue rotation — Bridge backlog cannot starve Spark jobs even on combined workers.

## Scheduler Leader Election

Two schedulers (one per DC) with PostgreSQL advisory lock:

| Log Line | Meaning |
|----------|---------|
| `scheduler leader acquired` | Holds the lock, runs cron |
| `scheduler standby, waiting for leader lock` | Peer waiting for failover |

### WARNING: Dual Schedulers Without Lock

**The Problem:** Running two scheduler containers without `SCHEDULER_LEADER_LOCK_ID`.

**Why This Breaks:** Cron overlap protection (`withoutOverlap`) is in-process only. Two schedulers will **double-enqueue** every job — replication kickoff, cache purge, crypto refresh — causing duplicate MLS fetches and wasted worker cycles.

**The Fix:** Always set `SCHEDULER_LEADER_LOCK_ID=913374211` when deploying two schedulers. One holds the lock; the other stays standby.

**When You Might Be Tempted:** "We only have one DC, so we don't need the lock." Correct for single-host — but if you ever add a second scheduler for failover, the lock is mandatory from day one.

## Image Edge Proxy

`nginx.idx-images.conf` proxies `/images/*` to the local API via Docker DNS:

```nginx
# Existing: nginx.idx-images.conf
resolver 127.0.0.11 ipv6=off valid=10s;
set $idx_api_upstream "idx-api:8000";
proxy_pass http://$idx_api_upstream;
```

**Why variable `proxy_pass`:** Nginx resolves upstream at startup with static `proxy_pass`. If the API container isn't running yet (Coolify rolling update), Nginx fails to start. Variable interpolation (`$idx_api_upstream`) defers resolution to request time.

**Network requirement:** Set the API container's Docker network alias to `idx-api` on every Coolify server.

## Anti-Patterns

### WARNING: Running Migrations on Every Deploy

**The Problem:** Running `goose up` from every Coolify app on every deploy.

**Why This Breaks:** Multiple replicas racing `goose up` can hit lock contention. More importantly, migrations should be intentional and verified before applying to production.

**The Fix:** Run `goose up` once per schema change, against the Patroni primary, from a laptop or one-off job — not from the API/worker startup.

### WARNING: Shared Image Cache Across DCs

**The Problem:** Mounting the same image cache volume on APIs in different datacenters.

**Why This Breaks:** The image cache (`/var/cache/geoidx/images`) is local filesystem. Cross-DC volume mounts add latency and complexity for no benefit — a cache miss just re-fetches from the MLS origin.

**The Fix:** Each API instance has its own local cache. Extra origin fetches on miss are acceptable.