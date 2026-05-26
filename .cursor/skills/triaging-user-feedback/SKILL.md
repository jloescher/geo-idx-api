---
name: triaging-user-feedback
description: |
  Routes user feedback into backlog priorities and quick wins for the Quantyra IDX platform.
  Use when: triaging support tickets, analyzing audit logs for pain points, prioritizing dashboard UX improvements, categorizing feedback from API consumers, evaluating error patterns for product action, or deciding what to ship next based on user signals.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Triaging User Feedback

Routes feedback signals from the Quantyra IDX dashboard (`/dashboard`), API proxy (`/api/v1/*`), audit logs, and support channels into actionable backlog items or quick wins. This is a B2B developer platform — feedback arrives as API errors, dashboard friction, and operational signals, not typical SaaS NPS surveys.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving triaging-user-feedback, verify against current docs FIRST:



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

## Feedback Signal Sources

| Signal | Where to Look | What It Tells You |
|--------|--------------|-------------------|
| Audit logs | `mls_proxy_audit_logs` table, `internal/service/audit/logger.go` | API usage patterns, cache hit rates, which endpoints get traffic |
| API errors | `internal/handler/` error responses, slog structured logs | Authentication friction, missing data, upstream failures |
| Dashboard friction | `internal/handler/dashboard/handler.go`, user sessions | Domain setup pain, token management confusion |
| Queue failures | `internal/queue/worker.go` job error logs | Replication lag, upstream MLS issues, persist failures |
| Stats endpoint | `GET /api/v1/bridge/stats`, `internal/service/sync/stats.go` | Replication health, dataset coverage gaps |

## Triage Framework

### Priority Classification

| Priority | Signal | Example | Action |
|----------|--------|---------|--------|
| P0 — Critical | API down, auth broken | All `/api/v1/search` returning 502 | Fix immediately, no triage needed |
| P1 — High | Repeated errors from multiple users | DNS verification failing consistently | Backlog top, ship this sprint |
| P2 — Medium | Feature gap with workaround | No staging token visibility in dashboard | Backlog, next sprint |
| P3 — Low | Nice-to-have, single user | Copy-to-clipboard on a new field | Backlog backlog, quick win when convenient |
| Quick Win | < 1 hour, high visibility | Better error message on token revocation | Ship immediately |

### Feedback Type Matrix

| Type | Source | Routing |
|------|--------|---------|
| Bug report | Error logs, support email | → Verify in audit logs → Reproduce → P0/P1 |
| Feature request | Dashboard feedback, API consumer asks | → Validate against `audit_logs` usage data → P2/P3 |
| Confusion/UX | Support tickets about dashboard flows | → Check `internal/handler/dashboard/` for copy/clarity → Quick win |
| Performance complaint | Stats endpoint, queue depth | → Check `replica_pages` / `listings` counts → P1 |
| Data quality | MLS consumer reports wrong data | → Check `modification_timestamp` / sync cursor → P1 |

## Quick Start

### Audit Log Analysis

```sql
-- Find most common error patterns by request type
SELECT request_type, COUNT(*) AS hits,
       COUNT(CASE WHEN cache_hit = 'miss' THEN 1 END) AS misses
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY request_type
ORDER BY hits DESC;
```

### Dashboard Friction Check

```go
// internal/handler/dashboard/handler.go — key user-facing error surfaces:
// - DNS verification: c.Status(422).SendString("TXT record not found...")
// - Domain errors: fiber.NewError(fiber.StatusNotFound, "domain not found")
// - Auth errors: fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials.")
```

## Common Patterns

### Pattern: Extract Pain Points from Audit Logs

```sql
-- Domains with highest cache miss rates (possible performance issue)
SELECT domain_slug,
       COUNT(*) AS total_requests,
       ROUND(COUNT(CASE WHEN cache_hit = 'miss' THEN 1 END)::numeric / COUNT(*) * 100, 1) AS miss_pct
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY domain_slug
HAVING COUNT(*) > 10
ORDER BY miss_pct DESC;
```

### Pattern: Map Error to User Journey Stage

| Journey Stage | Error Surface | Dashboard Route |
|---------------|--------------|-----------------|
| Onboarding | DNS TXT verification failure | `/dashboard/domains` |
| Token setup | Token creation/revocation errors | `/dashboard/api-tokens` |
| First API call | Domain auth middleware rejection | `GET /api/v1/properties` |
| Search usage | PostGIS query failures | `POST /api/v1/search` |
| Image loading | Image proxy cache miss | `/images/*` |
| Ongoing sync | Replication lag | `GET /api/v1/bridge/stats` |

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **auth-api-token** skill for token auth patterns that drive feedback
- See the **frontend-design** skill for dashboard UX improvements
- See the **queue-postgresql** skill for job failure triage
- See the **cache-postgres** skill for cache miss analysis
- See the **ux** skill for error message design