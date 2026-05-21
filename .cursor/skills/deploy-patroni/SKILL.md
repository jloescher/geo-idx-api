---
name: deploy-patroni
description: |
  Manages PostgreSQL Patroni cluster configuration, multi-DC failover, and
  Tailscale networking for Quantyra IDX API's PostgreSQL + PostGIS backend.
  Use when: setting up Patroni clusters, configuring multi-DC replication,
  running failover tests, tuning PostgreSQL parameters, verifying Tailscale
  connectivity to Patroni, adding read replicas, or troubleshooting replica lag.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Deploy Patroni Skill

Patroni provides synchronous PostgreSQL replication across two datacenters (NYC `re-db` + ATL `re-node-02`) over Tailscale mesh. All idx-api services (API, worker, scheduler) connect to the **Patroni primary** only in Phase 1 — no read replica routing yet. The scheduler uses a PostgreSQL advisory lock (`pg_try_advisory_lock`) for leader election, making the single-primary constraint critical.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving deploy-patroni, verify against current docs FIRST:



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

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Patroni primary | All writes, queue poll, scheduler lock | `DB_HOST=<patroni-primary-via-tailscale>` |
| Advisory lock | Scheduler leader election across DCs | `SCHEDULER_LEADER_LOCK_ID=913374211` |
| Tailscale mesh | Encrypted transport between DCs | Both hosts must have routes to Patroni VIP |
| `FOR UPDATE SKIP LOCKED` | Worker job reservation (primary only) | Workers cannot safely poll replicas |

## Common Patterns

### Verify connectivity (both DCs)

```bash
# From each Coolify server — uses scripts/verify-patroni-connectivity.sh
export DB_HOST=... DB_PORT=5432 DB_DATABASE=idx_api \
       DB_USERNAME=... DB_PASSWORD=... DB_SSLMODE=require
./scripts/verify-patroni-connectivity.sh
```

### Patroni cluster status

```bash
patronictl -c /etc/patroni/patroni.yml list
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **deploy-coolify** skill for Coolify app creation and shared env
- See the **hosting-tailscale** skill for mesh networking setup
- See the **postgresql** skill for schema, migrations, and PostGIS
- See the **queue-postgresql** skill for `jobs` table and `FOR UPDATE SKIP LOCKED`
- See the **deploy-docker** skill for Dockerfile targets (api, worker, scheduler)