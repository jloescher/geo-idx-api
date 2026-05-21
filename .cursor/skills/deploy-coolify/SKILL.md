---
name: deploy-coolify
description: |
  Manages Coolify deployments for the Quantyra IDX API Go stack, including
  multi-target Docker builds (api, worker, scheduler), PostgreSQL-backed job queues,
  multi-DC topology with Patroni over Tailscale, and Cloudflare geo load balancing.
  Use when: deploying to Coolify, configuring multi-DC, setting up workers or schedulers,
  troubleshooting deployment issues, running migrations in production, scaling replicas,
  configuring environment variables for Coolify apps, or verifying Patroni connectivity.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Deploy Coolify Skill

Deploy the Quantyra IDX API Go stack on Coolify using multi-target Docker builds. The system runs three binaries (`api`, `worker`, `scheduler`) from a single `Dockerfile` plus a separate `Dockerfile.idx-images` for Nginx image edge. All state lives in PostgreSQL — no Redis, no in-process shared state. Multi-DC uses Patroni over Tailscale with PostgreSQL advisory locks for scheduler leader election.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving deploy-coolify, verify against current docs FIRST:



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

### Single-Host Deployment (4 Coolify Apps)

| App | Dockerfile Target | Port |
|-----|-------------------|------|
| idx-api-web | `api` | 8000 |
| idx-api-worker | `worker` | — |
| idx-api-scheduler | `scheduler` | — |
| idx-images | `Dockerfile.idx-images` | 8080 |

### Multi-DC Deployment (10 Coolify Apps)

Two servers (NYC + ATL), shared Patroni primary over Tailscale. See `docs/coolify-deployment.md` §8 for the full app matrix.

## Key Concepts

| Concept | Detail |
|---------|--------|
| Build context | Repository root (`.`) for both Dockerfiles |
| Network alias | API container must be `idx-api` for `nginx.idx-images.conf` |
| Health checks | API: `GET /healthz`, idx-images: `GET /health`, API readiness: `GET /readyz` |
| Image cache | Per-API local disk (`/var/cache/geoidx/images`), NOT shared across DCs |
| Queue fairness | `ReserveFair` rotates across `WORKER_QUEUES` so Bridge cannot starve Spark |
| Scheduler lock | `SCHEDULER_LEADER_LOCK_ID=913374211` — one leader, one standby |

## Common Patterns

### Build and Smoke Locally

```bash
docker build -f Dockerfile --target api -t idx-api:local .
docker build -f Dockerfile.idx-images -t idx-images:local .
docker run --rm -p 8000:8000 --env-file .env idx-api:local
```

### Verify Patroni Connectivity

```bash
export DB_HOST=... DB_PORT=5432 DB_DATABASE=idx_api DB_USERNAME=... DB_PASSWORD=... DB_SSLMODE=require
./scripts/verify-patroni-connectivity.sh
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **docker** skill for Dockerfile patterns and multi-stage builds
- See the **queue-postgresql** skill for job queue internals
- See the **deploy-patroni** skill for Patroni cluster setup
- See the **hosting-tailscale** skill for Tailscale networking
- See the **hosting-coolify** skill for Coolify platform configuration
- See the **cache-postgres** skill for PostgreSQL-based caching patterns