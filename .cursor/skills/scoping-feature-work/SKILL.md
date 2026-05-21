---
name: scoping-feature-work
description: |
  Breaks features into MVP slices and acceptance criteria for the Quantyra IDX API.
  Use when: planning a new feature, splitting work into shippable slices, writing acceptance
  criteria, estimating scope, deciding what goes in MVP vs later phases, onboarding new
  customers, adding MLS datasets, or scoping dashboard/API/comps/GIS/replication changes.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Scoping Feature Work Skill

Feature scoping for an API-first MLS proxy with a minimal invite-only dashboard. Every feature touches one or more of three processes (`api`, `worker`, `scheduler`) sharing one PostgreSQL database. Scope slices must account for queue jobs, replication pipelines, and multi-DC safety — not just HTTP handlers.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving scoping-feature-work, verify against current docs FIRST:



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

### Slice a new feature — checklist

```
- [ ] Which process(es): api / worker / scheduler?
- [ ] Which tables: new migration or existing schema?
- [ ] Which queue job types (if background work)?
- [ ] Auth boundary: domain+token middleware or session?
- [ ] Multi-DC safe: advisory lock needed? shared state?
- [ ] Acceptance criteria: happy path + error states
- [ ] Rollback: migration down path exists?
```

### Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Process boundary | Every feature maps to 1-3 processes | Search = api only; replication = scheduler + worker + api |
| Queue job type | Background work is a named type string | `bridge.fetch_page`, `mls.replication_kickoff` |
| Dataset routing | `?dataset=stellar\|beaches` parameter | `internal/api/middleware/` resolves per-request |
| Auth tier | Session (dashboard) vs domain+token (API) | `requireAuth` vs `middleware.DomainToken` |
| Feature flag | Environment variable toggle | `MLS_STELLAR_ENABLED`, `MLS_BEACHES_ENABLED` |

## Common Patterns

### Scope a new API endpoint

**When:** Adding a read-only or mutation endpoint under `/api/v1`.

1. Handler in `internal/handler/<domain>/`
2. Service method in `internal/service/<domain>/`
3. Repository method in `internal/repository/`
4. Route in `internal/api/routes.go` under `v1` group (gets `domainAuth` + `mlsAccess` middleware)
5. If background work: new job type + queue registration

### Scope a new MLS dataset

**When:** Adding a third MLS feed beyond Bridge/Spark.

1. Config: new env vars (`NEWMLS_*`) in `internal/config/config.go`
2. Sync: fetch + persist workers in `internal/service/sync/`
3. Queue: new queue names (`newmls-sync-fetch`, `newmls-sync-persist`)
4. Middleware: dataset routing in `internal/api/middleware/`
5. Migration: `listing_sync_cursors` row for new dataset
6. Scheduler: kickoff enqueues new dataset's fetch jobs

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **go** skill for handler/service/repository patterns
- See the **fiber** skill for routing and middleware
- See the **queue-postgresql** skill for job lifecycle
- See the **auth-api-token** skill for domain + token auth
- See the **geospatial** skill for PostGIS feature scope