# Patroni Patterns Reference

## Contents
- Primary-Only Routing (Phase 1)
- Scheduler Advisory Lock
- Worker Job Safety
- Tailscale DNS Configuration
- Anti-Patterns

## Primary-Only Routing (Phase 1)

All idx-api services target the Patroni primary. This is enforced by environment variable `DB_HOST` pointing to the primary's Tailscale hostname.

```env
# Shared env across all 10 Coolify apps
DB_HOST=<patroni-primary-hostname-on-tailscale>
DB_PORT=5432
DB_DATABASE=idx_api
DB_SSLMODE=require
```

### WARNING: Routing Workers to Replicas

**The Problem:**

Workers use `FOR UPDATE SKIP LOCKED` to reserve jobs. This is a write-level lock that **requires the primary**.

**Why This Breaks:**
1. Replica has replication lag — jobs may already be claimed
2. `FOR UPDATE` on a read replica returns an error or stale rows
3. Scheduler advisory lock (`pg_try_advisory_lock`) is session-scoped and only valid on the primary

**The Fix:**

Workers and schedulers **always** connect to the primary, even in Phase 2 when API reads use replicas.

| Consumer | Primary Required? | Reason |
|----------|-------------------|--------|
| Worker (fetch/persist) | **Yes** | `FOR UPDATE SKIP LOCKED`, upserts |
| Scheduler | **Yes** | Advisory lock, job enqueue |
| API writes (audit, cache) | **Yes** | INSERT/UPDATE |
| API reads (search, lookup) | Phase 2 only | `SELECT` with acceptable lag |

## Scheduler Advisory Lock

The scheduler acquires a PostgreSQL session advisory lock to prevent double-enqueue across DCs.

```go
// internal/scheduler/ — simplified pattern
// SCHEDULER_LEADER_LOCK_ID defaults to 913374211
lockID := int64(913374211)
// pg_try_advisory_lock returns true if lock acquired
// Session-scoped: released when connection closes or session ends
```

### WARNING: Two Schedulers Without Lock

**The Problem:**

Two scheduler containers without `SCHEDULER_LEADER_LOCK_ID` will double-enqueue every cron job.

**Why This Breaks:**
1. Both schedulers enqueue `mls.replication_kickoff` every minute
2. Workers process duplicate fetch/persist jobs — wasted MLS API calls and duplicate upserts
3. Cache purge runs twice, crypto pricing fetches twice

**The Fix:**

Always set `SCHEDULER_LEADER_LOCK_ID` when running two schedulers. Logs should show exactly one `scheduler leader acquired` and one `scheduler standby`.

```env
SCHEDULER_LEADER_LOCK_ID=913374211
SCHEDULER_STANDBY_POLL_SECONDS=15
```

## Worker Job Safety

Workers poll `jobs` table using fair reservation across queue names. Bridge backlog cannot starve Spark fetch.

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

Key constraint: at most one `pending`/`processing` `replica_pages` row per `provider`+`dataset` at any time. This prevents concurrent persist collisions on the primary.

## Tailscale DNS Configuration

Patroni nodes communicate over Tailscale. The `DB_HOST` value must be a Tailscale-resolvable hostname, not a public IP.

```bash
# Verify from each Coolify host
tailscale status
ping <patroni-primary-hostname>
psql "postgres://USER:PASS@<patroni-primary-hostname>:5432/idx_api?sslmode=require"
```

### WARNING: Using Public IPs for DB_HOST

**The Problem:**

Setting `DB_HOST` to a public IP bypasses Tailscale encryption and may be blocked by firewall rules.

**Why This Breaks:**
1. Unencrypted PostgreSQL traffic over the internet
2. Firewall rules may only allow Tailscale interface (`tailscale0`)
3. Latency may be higher without Tailscale's optimized routing

**The Fix:**

Always use Tailscale hostnames. Run `./scripts/verify-patroni-connectivity.sh` from both DCs after any network change.

## Anti-Patterns

### WARNING: Manual `pg_ctl promote`

Never manually promote a standby. Use `patronictl switchover` or let Patroni handle automatic failover. Manual promotion desynchronizes the DCS (etcd/Consul) state from PostgreSQL state, causing split-brain.

### WARNING: Patroni on Coolify Hosts

Patroni PostgreSQL nodes run on **separate** infrastructure — not co-located with Coolify containers. From `docs/coolify-deployment.md`:

> Reserve host memory for PostgreSQL if co-located. Patroni cluster nodes are **not** on these Coolify hosts in the multi-DC layout (Tailscale only).

Running Patroni on the same host as Coolify containers causes memory contention and OOM kills during replication bursts.

## Cross-References

- See the **queue-postgresql** skill for job reservation patterns
- See the **deploy-coolify** skill for Coolify app matrix and resource limits
- See the **hosting-tailscale** skill for mesh network setup and route advertisement