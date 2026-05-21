# Tailscale Networking Patterns

## Contents
- SSL Mode Auto-Detection
- Shared Environment Across DCs
- Scheduler Advisory Lock
- Connectivity Verification
- Container Networking vs Tailscale

---

## SSL Mode Auto-Detection

`config.go:defaultDBSSLMode()` auto-selects SSL mode based on `DB_HOST`. This means switching from local dev to multi-DC requires **only** changing `DB_HOST` — no manual `DB_SSLMODE` override needed.

```go
// internal/config/config.go:311-322 — existing code
func defaultDBSSLMode() string {
    if v := os.Getenv("DB_SSLMODE"); v != "" {
        return v
    }
    host := strings.TrimSpace(env("DB_HOST", "127.0.0.1"))
    switch host {
    case "", "127.0.0.1", "localhost", "postgres", "::1":
        return "disable"
    default:
        return "require"
    }
}
```

**DO:** Set `DB_HOST` to the Patroni Tailscale IP or hostname; let auto-detection handle SSL.

**DON'T:** Hardcode `DB_SSLMODE=disable` in production Coolify env — the Tailscale tunnel is encrypted but PostgreSQL still requires SSL for credential protection.

---

## Shared Environment Across DCs

All 10 Coolify apps (2 API, 4 worker, 2 scheduler, 2 idx-images) share the **same** environment block. Key variables that matter for Tailscale connectivity:

```env
DB_HOST=<patroni-primary-on-tailscale>
DB_PORT=5432
DB_DATABASE=idx_api
DB_USERNAME=...
DB_PASSWORD=...
DB_SSLMODE=require

SCHEDULER_LEADER_LOCK_ID=913374211
SCHEDULER_STANDBY_POLL_SECONDS=15
```

**Why shared env:** Workers in ATL poll the same `jobs` table as NYC workers over Tailscale. Queue items are processed `FOR UPDATE SKIP LOCKED` — any worker across either DC can claim a job. No per-DC queue partitioning.

**DON'T** point different DCs at different databases. The PostgreSQL job queue and advisory locks require a single primary. See the **queue-postgresql** skill for queue polling details.

---

## Scheduler Advisory Lock

Two schedulers must not run cron independently. The Go scheduler acquires a PostgreSQL session advisory lock on startup:

```go
// SCHEDULER_LEADER_LOCK_ID maps to pg_try_advisory_lock(913374211)
// internal/config/config.go:297-300
Scheduler: SchedulerConfig{
    LeaderLockKey:       envSchedulerLeaderLockKey(),       // default 913374211
    StandbyPollInterval: time.Duration(envInt("SCHEDULER_STANDBY_POLL_SECONDS", 15)) * time.Second,
},
```

| Log line | Meaning |
|----------|---------|
| `scheduler leader acquired` | Holds the lock, running cron |
| `scheduler standby, waiting for leader lock` | Peer waiting for failover |

**Failover:** When the leader's PostgreSQL connection drops (host restart, network blip), the advisory lock releases automatically. The standby acquires it within `SCHEDULER_STANDBY_POLL_SECONDS`.

### WARNING: Running Schedulers Without Lock

Double-enqueued cron jobs cause:
1. Duplicate replication kickoff → wasted MLS API calls, potential rate limiting
2. Double cache purges
3. Duplicate GIS probes and crypto refreshes

If you see interleaved duplicate log lines from two schedulers, verify `SCHEDULER_LEADER_LOCK_ID` is set and identical on both.

---

## Connectivity Verification

`scripts/verify-patroni-connectivity.sh` is the canonical smoke test. Run from **each** Coolify host:

```bash
# Required env
export DB_HOST=<tailscale-patroni-ip>
export DB_PORT=5432
export DB_DATABASE=idx_api
export DB_USERNAME=idx_api
export DB_PASSWORD=<secret>
export DB_SSLMODE=require

# Run connectivity check
./scripts/verify-patroni-connectivity.sh

# Optionally also hit /readyz on the local API
API_URL=https://idx-api-nyc.example.com ./scripts/verify-patroni-connectivity.sh
```

The script runs:
1. `psql` with `SELECT version()` and `pg_is_in_recovery()` — confirms primary, not replica
2. Optional `GET /readyz` — confirms API can reach PostGIS

**DO:** Run from **both** re-db (NYC) and re-node-02 (ATL) to verify Tailscale routing works from each DC.

**DON'T:** Skip `DB_SSLMODE=require`. The script defaults to `require`, which is correct for Tailscale-connected Patroni.

---

## Container Networking vs Tailscale

Two separate networking layers operate in this project — do not confuse them:

| Layer | Scope | Example |
|-------|-------|---------|
| Docker bridge network | Per-host container-to-container | `idx-api:8000` (nginx.idx-images.conf → local API) |
| Tailscale mesh VPN | Host-to-host (cross-DC) | `DB_HOST=100.x.y.z` (container → host Tailscale → Patroni) |

`nginx.idx-images.conf` uses Docker DNS (`resolver 127.0.0.11`) to find the local API. This stays on the Docker bridge — no Tailscale involvement. Tailscale is only used for containers reaching the remote Patroni primary via the host's network stack.

### WARNING: Exposing DB Port Publicly

NEVER open PostgreSQL's port (5432) to the public internet, even with SSL. Tailscale provides authenticated, encrypted host-to-host connectivity. The firewall should allow 5432 only on the Tailscale interface.

---

## Cross-References

- See the **deploy-coolify** skill for Coolify app matrix and shared env setup
- See the **deploy-patroni** skill for Patroni cluster topology and Tailscale integration
- See the **queue-postgresql** skill for `FOR UPDATE SKIP LOCKED` and fair queue rotation across DCs
- See the **postgresql** skill for DSN construction and connection pooling