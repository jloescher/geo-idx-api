# Tailscale Networking Workflows

## Contents
- Initial Multi-DC Setup
- Post-Deploy Verification
- Scheduler Failover Validation
- Troubleshooting Cross-DC Connectivity

---

## Initial Multi-DC Setup

Deploying Tailscale networking between two Coolify hosts (re-db, re-node-02) for shared Patroni PostgreSQL.

Copy this checklist and track progress:
- [ ] Install Tailscale on both Coolify hosts (re-db, re-node-02)
- [ ] Authenticate Tailscale nodes and verify mutual reachability (`tailscale ping`)
- [ ] Configure Patroni to listen on the Tailscale interface
- [ ] Verify Patroni primary is reachable from both hosts: `psql "host=<tailscale-ip> port=5432 dbname=idx_api user=... sslmode=require"`
- [ ] Run `scripts/verify-patroni-connectivity.sh` from **both** hosts
- [ ] Set shared Coolify env: `DB_HOST=<patroni-tailscale-ip>`, `DB_SSLMODE=require`
- [ ] Set `SCHEDULER_LEADER_LOCK_ID=913374211` in shared env
- [ ] Create 10 Coolify apps per the app matrix (see **deploy-coolify** skill)
- [ ] Start workers (all 4) → schedulers (both) → APIs (both) → idx-images (both)
- [ ] Verify one scheduler logs `leader acquired`, the other logs `standby`

### Validation

```bash
# From re-db (NYC)
./scripts/verify-patroni-connectivity.sh

# From re-node-02 (ATL)
./scripts/verify-patroni-connectivity.sh

# Scheduler logs should show exactly one leader
docker logs idx-scheduler-nyc 2>&1 | grep -E "leader|standby"
docker logs idx-scheduler-atl 2>&1 | grep -E "leader|standby"
```

---

## Post-Deploy Verification

After deploying new container images or configuration changes that touch networking:

1. Check `GET /healthz` on both APIs — confirms container is running
2. Check `GET /readyz` on both APIs — confirms Postgres + PostGIS reachability over Tailscale
3. Verify workers are draining jobs: `SELECT COUNT(*) FROM jobs WHERE queue IN ('bridge-sync-fetch','spark-sync-fetch');`
4. Confirm replication kickoff in scheduler logs (only from leader)

```bash
# Quick smoke from each host
curl -fsS https://idx-api-nyc.example.com/readyz
curl -fsS https://idx-api-atl.example.com/readyz
```

### WARNING: `readyz` Fails from One DC

If `/readyz` succeeds from NYC but fails from ATL, the problem is **Tailscale routing**, not the application. Check:
1. `tailscale status` on the failing host — is the Patroni node connected?
2. `tailscale ping <patroni-ip>` — is there connectivity?
3. Firewall rules — is port 5432 allowed on the Tailscale interface on the Patroni host?
4. `DB_SSLMODE` — must be `require` for remote hosts; auto-detection handles this unless overridden

---

## Scheduler Failover Validation

Test that the standby scheduler takes over when the leader disconnects:

1. Identify which scheduler is leader: check logs for `scheduler leader acquired`
2. Stop the leader container: `docker stop idx-scheduler-nyc`
3. Wait up to `SCHEDULER_STANDBY_POLL_SECONDS` (default 15s)
4. Verify the other scheduler acquires the lock: logs show `scheduler leader acquired`
5. Verify cron jobs continue running from the new leader
6. Restart the original leader — it should now log `scheduler standby`

### WARNING: Advisory Lock Session Binding

PostgreSQL advisory locks are **session-scoped**. If the leader's DB connection drops (Tailscale interruption, Postgres restart, connection pool eviction), the lock releases **immediately**. This is the desired failover behavior. Do not attempt to make the lock resilient across reconnects — the standby must be able to acquire it.

---

## Troubleshooting Cross-DC Connectivity

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `readyz` timeout from ATL only | Tailscale routing down on re-node-02 | `tailscale status`, `tailscale ping <patroni-ip>` |
| Workers idle in ATL | `DB_HOST` wrong or Tailscale not routing | Verify env, check `tailscale ping` |
| Duplicate cron jobs | Two schedulers without advisory lock | Set `SCHEDULER_LEADER_LOCK_ID=913374211` on both |
| `sslmode=disable` error in logs | `DB_SSLMODE` not set, auto-detect picking wrong host | Explicitly set `DB_SSLMODE=require` |
| High latency on ATL worker poll | Tailscale DERP relay instead of direct | Check `tailscale status` for "relay" vs "direct" |
| Image cache misses after failover | Per-DC cache, not shared | Expected behavior; caches warm independently |

### Tailscale Direct vs Relay

Tailscale traffic between DCs should use **direct** connections, not DERP relay. If `tailscale ping` shows high latency or relay:

1. Check firewall allows UDP 41641 on both hosts
2. Verify NAT traversal isn't blocked
3. If direct connection is impossible, relay through the nearest DERP server is acceptable but adds latency

---

## Cross-References

- See the **deploy-coolify** skill for Coolify project setup, app creation, and shared environment
- See the **deploy-patroni** skill for Patroni cluster configuration and Tailscale VIP setup
- See the **deploy-docker** skill for Dockerfile targets and container build
- See the **postgresql** skill for connection pooling and query tuning
- See the **queue-postgresql** skill for worker topology across data centers