---
name: hosting-coolify
description: |
  Configures Coolify server deployments and resource allocation for the Quantyra IDX API Go stack.
  Use when: creating Coolify applications, setting Dockerfile targets (api/worker/scheduler/idx-images),
  allocating CPU/RAM, wiring shared environment variables, configuring multi-DC topology with Patroni,
  setting network aliases for idx-images, or troubleshooting Coolify hosting configuration issues.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Hosting Coolify Skill

Deploys the Quantyra IDX API stack on Coolify using a **single multi-target Dockerfile** producing three Go binaries (`api`, `worker`, `scheduler`) plus a separate Nginx image proxy (`Dockerfile.idx-images`). All state lives in PostgreSQL — no Redis, no shared filesystem between services.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving hosting-coolify, verify against current docs FIRST:



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

### Verified Existing Pattern — Single-Host (4 Apps)

| App | Dockerfile | Target | Port | Health |
|-----|-----------|--------|------|--------|
| idx-api-web | `Dockerfile` | `api` | 8000 | `GET /healthz` |
| idx-api-worker | `Dockerfile` | `worker` | — | process health optional |
| idx-api-scheduler | `Dockerfile` | `scheduler` | — | — |
| idx-images | `Dockerfile.idx-images` | default | 8080 | `GET /health` |

Build context: **repository root** (`.`) for all apps.

### New Code Pattern — Adding a Service Target

```dockerfile
# new code to add — extend Dockerfile with a new target
FROM alpine:3.21 AS new-service
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/new-service /usr/local/bin/new-service
USER nobody
CMD ["/usr/local/bin/new-service"]
```

## Key Concepts

| Concept | Detail | Config |
|---------|--------|--------|
| Multi-target build | One `Dockerfile`, three `FROM` stages | `docker build --target api` |
| Fair worker queues | `ReserveFair` rotates across queue names | `WORKER_QUEUES=default,bridge-sync-fetch,...` |
| Scheduler advisory lock | `pg_try_advisory_lock` prevents double-enqueue | `SCHEDULER_LEADER_LOCK_ID=913374211` |
| Nginx Docker DNS | `resolver 127.0.0.11` for rolling updates | `nginx.idx-images.conf` |
| Image cache | Local disk per DC, not shared | `IMAGE_CACHE_PATH=/var/cache/geoidx/images` |
| Network alias | idx-images proxies to `idx-api:8000` | Coolify container alias = `idx-api` |

## Common Patterns

### Worker Queue Split (Recommended at Scale)

```env
# default-worker (x1) — kickoff, purge, crypto, GIS
WORKER_QUEUES=default
# fetch-worker (x2) — MLS HTTP only
WORKER_QUEUES=bridge-sync-fetch,spark-sync-fetch
# persist-worker (x2-4) — PostgreSQL upsert
WORKER_QUEUES=bridge-sync-persist,spark-sync-persist
```

### Multi-DC Shared Environment

```env
DB_HOST=<patroni-primary-on-tailscale>
DB_PORT=5432
DB_DATABASE=idx_api
SCHEDULER_LEADER_LOCK_ID=913374211
SCHEDULER_STANDBY_POLL_SECONDS=15
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

## See Also

- [patterns](references/patterns.md) — App matrix, resource allocation, networking, anti-patterns
- [workflows](references/workflows.md) — Deploy, migrate, seed, smoke-test, multi-DC setup

## Related Skills

- See the **deploy-coolify** skill for CI/CD pipeline and image publishing
- See the **docker** skill for Dockerfile patterns and multi-stage builds
- See the **deploy-patroni** skill for multi-DC PostgreSQL clustering
- See the **queue-postgresql** skill for job queue internals
- See the **hosting-tailscale** skill for cross-DC networking