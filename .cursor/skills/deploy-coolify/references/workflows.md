# Deployment Workflows Reference

## Contents
- Fresh Single-Host Deployment
- Multi-DC Deployment (NYC + ATL)
- Post-Deploy Verification
- Laravel-to-Go Cutover
- Scaling Workers During Catch-Up
- Troubleshooting

## Fresh Single-Host Deployment

Copy this checklist and track progress:

- [ ] Clone repo, copy `.env.example` to `.env`, fill all `DB_*`, `BRIDGE_*`, `SPARK_*` values
- [ ] Create 4 Coolify apps: idx-api-web (`api`), idx-api-worker (`worker`), idx-api-scheduler (`scheduler`), idx-images (`Dockerfile.idx-images`)
- [ ] Set build context to repo root (`.`) for all apps
- [ ] Attach shared environment to all 4 apps
- [ ] Set API container network alias to `idx-api` (required by idx-images Nginx)
- [ ] Run migrations once: `export GOOSE_DBSTRING="postgres://..." && make migrate`
- [ ] Seed admin once: `export ADMIN_SEED_EMAIL=... ADMIN_SEED_PASSWORD=... && make seed-admin`
- [ ] Deploy all 4 apps (Coolify rebuild triggers)
- [ ] Verify: `GET /healthz`, `GET /readyz`, scheduler logs show `scheduler leader acquired`, workers drain `jobs`

## Multi-DC Deployment (NYC + ATL)

### Prerequisites

1. Patroni cluster operational with primary accessible over Tailscale
2. Tailscale installed on both Coolify servers with routes to Patroni primary
3. Cloudflare account for geo load balancing

### Deploy Order

Copy this checklist and track progress:

- [ ] **Step 1:** Verify Tailscale + psql from both servers: `./scripts/verify-patroni-connectivity.sh`
- [ ] **Step 2:** Create 10 Coolify apps (see app matrix in `docs/coolify-deployment.md` §8)
- [ ] **Step 3:** Attach shared environment to all 10 apps (same Patroni primary DSN)
- [ ] **Step 4:** Set `SCHEDULER_LEADER_LOCK_ID=913374211` on both scheduler apps
- [ ] **Step 5:** Run `goose up` once on Patroni primary
- [ ] **Step 6:** Run `make seed-admin` once (not on runtime containers)
- [ ] **Step 7:** Start workers (all 4) → schedulers (both) → APIs (both) → idx-images (both)
- [ ] **Step 8:** Verify one scheduler shows `scheduler leader acquired`, other shows `scheduler standby`
- [ ] **Step 9:** Configure Cloudflare geo LB for `idx-api` and `idx-images` hostnames
- [ ] **Step 10:** Smoke test from both regions: `/healthz`, `/readyz`, workers drain `jobs`

### App Matrix (10 Apps)

| App | Server | Dockerfile | Target | Port |
|-----|--------|------------|--------|------|
| idx-api-nyc | re-db | `Dockerfile` | `api` | 8000 |
| idx-api-atl | re-node-02 | `Dockerfile` | `api` | 8000 |
| idx-worker-nyc-1 | re-db | `Dockerfile` | `worker` | — |
| idx-worker-nyc-2 | re-db | `Dockerfile` | `worker` | — |
| idx-worker-atl-1 | re-node-02 | `Dockerfile` | `worker` | — |
| idx-worker-atl-2 | re-node-02 | `Dockerfile` | `worker` | — |
| idx-scheduler-nyc | re-db | `Dockerfile` | `scheduler` | — |
| idx-scheduler-atl | re-node-02 | `Dockerfile` | `scheduler` | — |
| idx-images-nyc | re-db | `Dockerfile.idx-images` | — | 8080 |
| idx-images-atl | re-node-02 | `Dockerfile.idx-images` | — | 8080 |

**One app per server per role** — do not use Coolify replica scaling on a single host.

## Post-Deploy Verification

After any deploy, validate the feedback loop:

1. **Health:** `curl -f https://<api-host>/healthz` and `curl -f https://<api-host>/readyz`
2. **Scheduler:** Logs show `scheduler leader acquired` (not standby on both)
3. **Workers:** `SELECT count(*) FROM jobs;` — count should decrease as workers process
4. **Replication:** `GET /api/v1/bridge/stats` — check `replication_in_progress` / `last_sync_finished_at`
5. **Images:** `curl -f https://<images-host>/health` returns `OK`
6. **Re-verify:** Wait one minute, check again — replication kickoff should appear in logs

## Laravel-to-Go Cutover

Copy this checklist and track progress:

- [ ] Deploy Go API, worker, scheduler against the same PostgreSQL database
- [ ] Run `goose up` (idempotent on existing Laravel-era schema)
- [ ] Point traffic to Go API (port 8000)
- [ ] Scale workers (fetch + persist split recommended)
- [ ] Start scheduler container
- [ ] Purge legacy Laravel jobs: `DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';`
- [ ] Notify customers to re-issue API keys from `/dashboard` (SHA-256 storage, not legacy `id|secret`)
- [ ] Verify `/healthz`, `/readyz`, `POST /api/v1/search`, `/images/*`
- [ ] Monitor replication lag via `GET /api/v1/bridge/stats`

See the **go-cutover** documentation at `docs/go-cutover.md` for full details.

## Scaling Workers During Catch-Up

During initial replication or re-seed, MLS data volume is high. Split workers for throughput:

1. Deploy fetch workers (2×): `WORKER_QUEUES=bridge-sync-fetch,spark-sync-fetch`
2. Deploy persist workers (2–4×): `WORKER_QUEUES=bridge-sync-persist,spark-sync-persist`
3. Keep default worker (1×): `WORKER_QUEUES=default` for kickoff, purge, crypto, GIS
4. Monitor `jobs` table depth and `replica_pages` staging rows
5. When replication completes and only incremental sync runs, collapse back to combined workers

**Env tuning for catch-up:**

| Variable | Bridge Default | Spark Default |
|----------|---------------|---------------|
| `*_SYNC_REPLICATION_TOP` | 2000 | 1000 (API cap) |
| `*_SYNC_PERSIST_JOB_CHUNK` | 50 | 50 |

## Troubleshooting

| Symptom | Check | Fix |
|---------|-------|-----|
| `unknown job type` in logs | Legacy Laravel rows in `jobs` | `DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';` |
| Duplicate kickoff every minute | Two schedulers, no advisory lock | Set `SCHEDULER_LEADER_LOCK_ID=913374211` on both |
| Bridge dominates, Spark idle | Single worker, global job ordering | Use fair multi-queue or split fetch workers |
| Spark jobs not running | Missing queue names | Add `spark-sync-fetch,spark-sync-persist` to `WORKER_QUEUES` |
| Login fails after cutover | Password format mismatch | `make seed-admin` (Argon2id hashes) |
| API tokens rejected | Legacy Sanctum format | Re-issue PATs from `/dashboard` (SHA-256) |
| 502 on `/images/*` | idx-images can't reach API | Set API network alias to `idx-api`, check port 8000 |
| `readyz` fails from ATL | Patroni/Tailscale latency | Run `./scripts/verify-patroni-connectivity.sh` from ATL |
| Nginx won't start | Static `proxy_pass` before API up | Use variable interpolation (already fixed in `nginx.idx-images.conf`) |