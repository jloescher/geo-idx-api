# Patroni Workflows Reference

## Contents
- Initial Multi-DC Setup
- Deploy Order
- Failover Procedure
- Phase 2 Read Replica Migration
- Verification Checklist

## Initial Multi-DC Setup

### Prerequisites

- Two hosts: `re-db` (NYC) and `re-node-02` (ATL)
- Tailscale installed on both with routes to Patroni VIP
- Patroni + PostgreSQL + PostGIS installed on both hosts
- DCS (etcd or Consul) accessible from both Patroni nodes

### Step-by-step

1. Install and configure Tailscale on both hosts
2. Verify Tailscale connectivity between hosts
3. Deploy Patroni on both nodes with matching `patroni.yml` scope and name
4. Bootstrap primary node first, then join replica
5. Verify replication: `patronictl list` shows both nodes, replica in `running` state
6. Run `./scripts/verify-patroni-connectivity.sh` from both Coolify hosts

```bash
# On each Coolify server
export DB_HOST=<patroni-primary-on-tailscale> DB_PORT=5432 \
       DB_DATABASE=idx_api DB_USERNAME=... DB_PASSWORD=... DB_SSLMODE=require
./scripts/verify-patroni-connectivity.sh

# Optional: also check API health
API_URL=https://<that-dc-api-host> ./scripts/verify-patroni-connectivity.sh
```

### WARNING: Bootstrapping Both Nodes Simultaneously

**The Problem:**

Starting both Patroni nodes with bootstrap mode at the same time causes split-brain — each thinks it's the primary.

**Why This Breaks:**
1. Both nodes initialize independent PostgreSQL data directories
2. DCS sees two primaries claiming the same scope
3. Manual intervention required to reset one node

**The Fix:**

Bootstrap primary first. Wait for it to be fully ready (`patronictl list` shows `Leader`). Then start replica with `patroni --bootstrap` or let Patroni auto-join.

## Deploy Order

Copy this checklist and track progress:

```markdown
- [ ] Tailscale + `psql` connectivity from both servers
- [ ] Merge/deploy images with scheduler advisory lock configured
- [ ] Create 10 Coolify apps + shared env (see **deploy-coolify** skill)
- [ ] `goose up` once on Patroni primary
- [ ] `make seed-admin` once (not on runtime API env)
- [ ] Start workers (all 4) — verify `FOR UPDATE SKIP LOCKED` works
- [ ] Start schedulers (both) — confirm one leader in logs
- [ ] Start APIs (both DCs) — verify `/healthz` and `/readyz`
- [ ] Start idx-images (both DCs) — verify `/health`
- [ ] Cloudflare geo LB configured with health checks
- [ ] Smoke: replication kickoff in logs, `jobs` table drains, purge legacy rows
```

## Failover Procedure

### Planned Switchover

```bash
# 1. Verify cluster state
patronictl -c /etc/patroni/patroni.yml list

# 2. Switchover to specific node
patronictl -c /etc/patroni/patroni.yml switchover --master <current-primary> --candidate <new-primary>

# 3. Update DB_HOST in Coolify shared env if Tailscale hostname changes
# 4. Restart all Coolify apps to pick up new DB_HOST
# 5. Verify: /healthz, /readyz, worker logs show successful job processing
```

### Automatic Failover

Patroni handles this automatically when the primary becomes unreachable. The replica promotes itself within the DCS TTL (typically 10-30 seconds).

**Post-failover steps:**
1. Check `patronictl list` — confirm new leader
2. Verify Coolify apps reconnected (pgx connection pool handles reconnection)
3. Check scheduler logs — standby should acquire advisory lock
4. Check worker logs — confirm `FOR UPDATE SKIP LOCKED` works against new primary

### WARNING: Forgetting to Update DB_HOST After Failover

**The Problem:**

If automatic failover promotes a different host and `DB_HOST` still points to the old primary (now a replica), all writes fail silently or error.

**Why This Breaks:**
1. Workers cannot reserve jobs (`FOR UPDATE` fails on replica)
2. Scheduler cannot acquire advisory lock
3. API writes (audit, cache) error out

**The Fix:**

Use a **Tailscale VIP or DNS CNAME** that always resolves to the current primary. Alternatively, use Patroni's `PATRONI_RESTAPI_CONNECT_ADDRESS` with a floating IP. Do not hardcode individual host IPs in `DB_HOST`.

## Phase 2 Read Replica Migration

Phase 1 uses primary only. Phase 2 adds `DB_READ_HOST` for API read paths:

```env
# Phase 2 — future, not yet implemented
DB_READ_HOST=<patroni-replica-on-tailscale>
```

### Safe Read Paths (Phase 2)

| Path | Safe for Replica? | Condition |
|------|-------------------|-----------|
| `POST /api/v1/search` (PostGIS) | Yes | Accept stale listings within lag window |
| `GET /api/v1/bridge/stats` | Yes | Eventually consistent |
| Comps mirror `SELECT` | Yes | Read-only |
| `GET /api/v1/gis` | Yes | Parcel data rarely changes |

### WARNING: Routing These to Replica

Never route to replica: worker queue poll, scheduler lock, job enqueue, audit log writes, cache `Put`, domain/token mutations, image cache metadata writes.

Implementation surface: `internal/repository/db.go` — add second pool for `DB_READ_HOST`. Route via explicit read/write methods, not automatic query classification.

## Verification Checklist

After any Patroni change, run this feedback loop:

1. Make changes to Patroni config or failover
2. Validate: `./scripts/verify-patroni-connectivity.sh` from both DCs
3. Validate: `patronictl list` shows expected leader
4. Validate: `GET /healthz` and `GET /readyz` from both API instances
5. Validate: scheduler logs show one leader, one standby
6. Validate: worker logs show successful job processing (no connection errors)
7. If any validation fails, investigate and repeat from step 1
8. Only proceed when all validations pass

```bash
# Quick smoke after changes
patronictl -c /etc/patroni/patroni.yml list
curl -s http://localhost:8000/healthz
curl -s http://localhost:8000/readyz
./scripts/verify-patroni-connectivity.sh
```

## Cross-References

- See the **deploy-coolify** skill for Coolify app matrix and environment setup
- See the **hosting-tailscale** skill for mesh network configuration
- See the **postgresql** skill for schema migrations and PostGIS setup
- See the **deploy-docker** skill for container build and deployment targets