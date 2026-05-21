---
name: hosting-tailscale
description: |
  Configures Tailscale mesh VPN for multi-DC PostgreSQL (Patroni) connectivity
  between Coolify hosts. Use when: setting up multi-DC networking, verifying
  Patroni reachability over Tailscale, troubleshooting cross-DC database
  connectivity, configuring DB_HOST/DB_SSLMODE for remote Patroni, or deploying
  the 10-app multi-DC topology (NYC + ATL).
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__web_reader__webReader
---

# Hosting Tailscale Skill

Tailscale provides the encrypted mesh VPN layer between **re-db** (NYC) and **re-node-02** (ATL) Coolify hosts and their shared **Patroni PostgreSQL** primary. All application containers point `DB_HOST` at the Patroni primary's Tailscale IP; there is no Tailscale inside containers — the host's Tailscale daemon routes traffic.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving hosting-tailscale, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.




Required wiring surfaces:
- runtime/infrastructure config: Dockerfile
- nearest typed request/context boundary
- handler/procedure boundary before external side effects

Side-effect barrier:
- Place guards before external APIs, auth mutations, email sends, analytics events, storage writes, and database mutations.


Fallback policy:
- Prefer provider-native/platform-managed primitives by default when no explicit override exists.
- Follow clear user/project overrides, but mention the native alternative and tradeoff.
- Fallbacks must be durable, multi-instance safe, and atomic under concurrency.

Verification rules:
- [error] native-or-explicit-override: Use the provider-native primitive first unless the user/project explicitly overrides it.
- [error] atomic-fallback: Fallback counters must be atomic under concurrency.

## Quick Start

### Verify Connectivity (Existing Script)

```bash
# From either Coolify host (re-db or re-node-02)
export DB_HOST=100.x.y.z DB_PORT=5432 DB_DATABASE=idx_api
export DB_USERNAME=... DB_PASSWORD=... DB_SSLMODE=require
./scripts/verify-patroni-connectivity.sh

# Also check API health:
API_URL=https://idx-api-nyc.example.com ./scripts/verify-patroni-connectivity.sh
```

### SSL Mode Auto-Detection (Existing Code)

```go
// internal/config/config.go — defaultDBSSLMode() automatically picks the right mode
// DB_HOST=127.0.0.1|localhost|postgres|::1  → sslmode=disable
// DB_HOST=<tailscale IP or any other>        → sslmode=require
```

## Key Concepts

| Concept | Detail | Config Surface |
|---------|--------|----------------|
| Host-level routing | Tailscale runs on Coolify hosts, not inside containers | Host OS Tailscale daemon |
| Patroni primary | All containers connect to one primary via Tailscale IP | `DB_HOST`, `DB_PORT`, `DB_SSLMODE` |
| SSL auto-detect | Non-local `DB_HOST` → `require`; local loopback → `disable` | `config.go:defaultDBSSLMode()` |
| Scheduler leader | Advisory lock prevents double cron across DCs | `SCHEDULER_LEADER_LOCK_ID=913374211` |
| Standby failover | Standby polls every N seconds, takes over if leader disconnects | `SCHEDULER_STANDBY_POLL_SECONDS=15` |
| Image cache split | Per-API local disk; geo-routed idx-images avoids cross-DC image fetches | `IMAGE_CACHE_PATH` |

## Common Patterns

### Multi-DC Environment (Shared Across 10 Coolify Apps)

```env
# Patroni primary via Tailscale — same on all 10 apps
DB_HOST=<patroni-primary-tailscale-hostname>
DB_PORT=5432
DB_DATABASE=idx_api
DB_USERNAME=idx_api
DB_PASSWORD=...
DB_SSLMODE=require

# Scheduler lock — prevents double cron
SCHEDULER_LEADER_LOCK_ID=913374211
SCHEDULER_STANDBY_POLL_SECONDS=15
```

### WARNING: Two Schedulers Without Advisory Lock

Running two scheduler containers (NYC + ATL) without `SCHEDULER_LEADER_LOCK_ID` causes **double-enqueued** replication kickoff, proxy cache purge, and all cron jobs every minute. The lock is non-negotiable for multi-DC.

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **deploy-coolify** skill for Coolify app creation and shared env
- See the **deploy-patroni** skill for Patroni cluster setup and failover
- See the **deploy-docker** skill for Dockerfile targets and build context
- See the **postgresql** skill for DB connection pooling and query patterns
- See the **queue-postgresql** skill for worker queue topology across DCs