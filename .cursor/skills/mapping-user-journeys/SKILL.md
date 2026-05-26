---
name: mapping-user-journeys
description: |
  Maps in-app journeys across the Quantyra IDX API surface — dashboard onboarding,
  domain verification, API token lifecycle, search/comps/GIS request flows, and
  replication monitoring. Identifies friction points in handler chains, middleware
  gates, and error states. Use when: tracing a user flow end-to-end, debugging
  auth or domain verification failures, evaluating onboarding drop-off, auditing
  error message quality, or adding new steps to an existing journey.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Mapping User Journeys Skill

Trace end-to-end flows through the Quantyra IDX API — from dashboard login to
first API call, from search request to cached response. This skill maps every
user-facing surface: the invite-only dashboard (`/dashboard`), the MLS proxy
API (`/api/v1/*`), image delivery (`/images/*`), and background replication.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving mapping-user-journeys, verify against current docs FIRST:



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

### Journey: New Customer Onboarding

```
Admin seeds account → User receives invite link
  → GET /invite/:token (registration form)
  → POST /invite/:token (create account + password)
  → Redirect to /login
  → POST /login (session auth)
  → Redirect to /dashboard
  → POST /dashboard/domains (register domain + select MLS dataset)
  → POST /dashboard/domains/:id/verify-txt (DNS TXT verification)
  → POST /dashboard/api-tokens (create production token)
  → First API call with Bearer token + X-Domain-Slug
```

### Journey: API Request (Search)

```
Client → POST /api/v1/search
  → DomainToken() middleware: validate Bearer + domain + abilities
  → MLSAccess() middleware: check domain allowed_datasets
  → search.Handle(): parse SearchRequest, DecideRoute()
  → RoutePostgresOnly: query PostGIS listings
  → RouteUpstreamOnly: proxy to Bridge/Spark
  → RouteSplit: parallel PostGIS + upstream, merge
  → MergeMirrorListing(): reassemble flat RESO JSON
  → audit.Log(): write to mls_proxy_audit_logs
  → JSON response
```

## Key Concepts

| Concept | Location | User Impact |
|---|---|---|
| Domain verification gate | `middleware.go` → `"Domain must be TXT-verified"` | API tokens blocked until DNS verified |
| Hybrid search routing | `search/service.go` → `DecideRoute()` | Active/Pending → PostGIS; Closed → upstream |
| Cache partition | `cache.WebPartition(slug, feed, type)` | Per-domain isolation, 15-min TTL |
| Fair queue reservation | `queue.ReserveFair` | Bridge backlog can't starve Spark |
| Audit trail | `audit.Log(c, requestType, count, cacheHit)` | Every proxied request tracked |

## Common Patterns

### Tracing a Friction Point

1. Identify the failing handler from the route (`routes.go`)
2. Walk the middleware chain (domain auth → MLS access → handler)
3. Check the error string — most are explicit (e.g., `"Domain is not registered, inactive, or not owned by this token."`)
4. Look for state gates: `verification_status`, `is_active`, `allowed_mls_datasets`

### Evaluating Onboarding Drop-off

1. Map the journey steps (invite → register → login → add domain → verify → create token → first call)
2. Count state transitions that require external action (DNS verification, API call)
3. Identify steps with no error recovery path (staging token already exists → 409)

## See Also

- references/activation-onboarding.md
- references/engagement-adoption.md
- references/in-app-guidance.md
- references/product-analytics.md
- references/roadmap-experiments.md
- references/feedback-insights.md

## Related Skills

- See the **auth-api-token** skill for token lifecycle details
- See the **proxy-web** skill for MLS proxy caching patterns
- See the **cache-postgres** skill for cache layer internals
- See the **queue-postgresql** skill for job processing flows
- See the **geospatial** skill for PostGIS query patterns
- See the **fiber** skill for HTTP handler and middleware patterns