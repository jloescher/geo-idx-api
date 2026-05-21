# Coolify Hosting Workflows

## Contents
- First-Time Deployment
- Post-Deploy Verification
- Multi-DC Setup
- Worker Queue Split
- Troubleshooting

## First-Time Deployment

Copy this checklist and track progress:

- [ ] Create Coolify project (staging or production)
- [ ] Add shared environment variables (`DB_*`, `BRIDGE_*`, `SPARK_*`)
- [ ] Create `idx-api-web` app — Dockerfile target `api`, port 8000
- [ ] Create `idx-api-worker` app — Dockerfile target `worker`, set `WORKER_QUEUES`
- [ ] Create `idx-api-scheduler` app — Dockerfile target `scheduler`, set `SCHEDULER_LEADER_LOCK_ID`
- [ ] Create `idx-images` app — `Dockerfile.idx-images`, port 8080
- [ ] Set API container network alias to `idx-api`
- [ ] Run goose migrations once: `GOOSE_DBSTRING="..." make migrate`
- [ ] Seed admin once: `ADMIN_SEED_EMAIL=... ADMIN_SEED_PASSWORD=... make seed-admin`
- [ ] Start workers → scheduler → API → idx-images (in that order)
- [ ] Verify `/healthz`, `/readyz`, and `/health` endpoints

## Post-Deploy Verification

1. Make changes (deploy or re-deploy)
2. Validate health endpoints:
   ```bash
   curl -sf https://idx-api.example.com/healthz  # liveness
   curl -sf https://idx-api.example.com/readyz   # Postgres + PostGIS
   curl -sf https://idx-images.example.com/health # Nginx edge
   ```
3. Check replication status:
   ```bash
   curl -sf https://idx-api.example.com/api/v1/bridge/stats
   ```
4. Verify workers processing:
   ```sql
   SELECT provider, dataset, status, COUNT(*) FROM replica_pages GROUP BY 1,2,3;
   ```
5. Validate scheduler leadership — one scheduler logs `scheduler leader acquired`; the other logs `scheduler standby, waiting for leader lock`
6. Smoke image proxy:
   ```bash
   curl -sf -o /dev/null -w "%{http_code}" https://idx-images.example.com/images/mls/test.jpg
   ```
7. If any validation fails, fix issues and repeat from step 2

## Multi-DC Setup

Prerequisites: Patroni cluster over Tailscale. See the **deploy-patroni** skill and **hosting-tailscale** skill.

Copy this checklist and track progress:

- [ ] Tailscale installed on both servers with routes to Patroni primary
- [ ] Verify connectivity from both servers: `./scripts/verify-patroni-connectivity.sh`
- [ ] Create 10 Coolify apps (see patterns.md Application Matrix)
- [ ] Attach shared project environment to all 10 apps
- [ ] Set `SCHEDULER_LEADER_LOCK_ID=913374211` on both schedulers
- [ ] Run `goose up` once on Patroni primary
- [ ] Start all 4 workers → both schedulers → both APIs → both idx-images
- [ ] Confirm one scheduler is leader in logs
- [ ] Configure Cloudflare geo LB (NYC pool → re-db, ATL pool → re-node-02)
- [ ] Smoke from both regions: `/healthz`, `/readyz`, worker log interleaving
- [ ] Purge legacy Laravel jobs: `DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';`

### Cloudflare Load Balancing

| Hostname | Pool NYC | Pool ATL | Health |
|----------|----------|----------|--------|
| `idx-api.domain.cc` | re-db → `:8000` | re-node-02 → `:8000` | `GET /healthz` |
| `idx-images.domain.cc` | re-db → `:8080` | re-node-02 → `:8080` | `GET /health` |

Coolify's single-host Traefik LB is insufficient for two datacenters. Use Cloudflare (recommended) or a standalone reverse proxy.

### Deploy Order

1. Tailscale + `psql` from both servers
2. Merge/deploy images with scheduler advisory lock enabled
3. Create 10 Coolify apps + shared env
4. `goose up` once on Patroni primary
5. `make seed-admin` once (not on runtime API env)
6. Start **workers** (all 4) → **schedulers** (both; confirm one leader in logs) → **APIs** → **idx-images**
7. Cloudflare geo LB
8. Smoke: `/healthz`, `/readyz`, workers drain `jobs`, replication kickoff in logs

## Worker Queue Split

When replication backlogs or queue depth is high, split workers by role:

```env
# default-worker (x1) — kickoff, purge, crypto, GIS
WORKER_QUEUES=default
# fetch-worker (x2) — MLS HTTP only
WORKER_QUEUES=bridge-sync-fetch,spark-sync-fetch
# persist-worker (x2-4) — PostgreSQL upsert
WORKER_QUEUES=bridge-sync-persist,spark-sync-persist
```

In Coolify, create separate applications for each worker type with different `WORKER_QUEUES` values. Scale persist workers independently from fetch workers.

### Tuning Env Variables

| Variable | Bridge Default | Spark Default |
|----------|---------------|---------------|
| `*_SYNC_REPLICATION_TOP` | 2000 | 1000 |
| `*_SYNC_PERSIST_JOB_CHUNK` | 50 | 50 |
| `*_SYNC_UPSERT_CHUNK` | see config | see config |

Adjust from queue depth and `pg_stat_statements` — don't tune blindly.

## Troubleshooting

### Worker Not Processing Jobs

1. Check `WORKER_QUEUES` matches queue names in the `jobs` table
2. Verify `DB_*` connectivity from the worker container
3. Check for stuck `processing` jobs: `SELECT * FROM jobs WHERE status = 'processing';`
4. Legacy Laravel jobs block the queue: `DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';`

### Scheduler Double-Enqueue

1. Confirm `SCHEDULER_LEADER_LOCK_ID` is set and identical on both schedulers
2. Check logs: only one should show `scheduler leader acquired`
3. If both are leaders, the advisory lock is not working — verify DB connectivity

### idx-images 502 Bad Gateway

1. API container network alias must be `idx-api` (not the Coolify-generated name)
2. Verify API is healthy: `GET /healthz` from the idx-images container
3. Check nginx config uses `resolver 127.0.0.11` with variable `proxy_pass`

### Replication Stuck

1. Check for orphaned `replica_pages` rows: `SELECT * FROM replica_pages WHERE status IN ('pending','processing');`
2. At most one pending/processing row per `provider`+`dataset` should exist
3. Verify `BRIDGE_API_KEY` and `SPARK_ACCESS_TOKEN` are valid
4. Check `GET /api/v1/bridge/stats` for `replication_in_progress` and `last_sync_finished_at`