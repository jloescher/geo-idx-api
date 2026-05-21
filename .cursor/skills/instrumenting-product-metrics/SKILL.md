---
name: instrumenting-product-metrics
description: |
  Defines product events, funnels, and activation metrics for idx-api.
  Use when: adding event tracking, audit logging, funnel instrumentation,
  activation metrics, conversion tracking, or product analytics queries
  to any handler, service, middleware, or dashboard route.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Instrumenting Product Metrics Skill

idx-api has **one audit surface** — `mls_proxy_audit_logs` — capturing MLS proxy calls only. Dashboard operations, auth events, domain/token lifecycle, GIS, and image proxy are uninstrumented. This skill defines how to add structured product events consistently.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving instrumenting-product-metrics, verify against current docs FIRST:



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

### Verified Existing Pattern

```go
// internal/service/audit/logger.go — current proxy audit
func (l *Logger) Log(c *fiber.Ctx, requestType string, listingCount *int, cacheHit *string) {
    slug, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
    tokenName, _ := c.Locals(ctxkeys.MLSTokenName).(string)
    userID, _ := c.Locals(ctxkeys.MLSUserID).(int64)
    ip := c.IP()
    _, _ = l.db.Pool.Exec(context.Background(), `
        INSERT INTO mls_proxy_audit_logs (domain_slug, token_name, request_type, listing_count, ip_address, user_id, cache_hit)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, nullStr(slug), tokenName, requestType, listingCount, ip, userID, cacheHit)
}
```

### New Code Pattern

```go
// new code to add — product event struct
type ProductEvent struct {
    UserID      int64       `json:"user_id"`
    EventName   string      `json:"event_name"`
    Properties  Map         `json:"properties"`  // JSONB
    DomainSlug  *string     `json:"domain_slug"`
    SessionID   *string     `json:"session_id"`
}

func (l *Logger) LogEvent(ctx context.Context, evt ProductEvent) {
    props, _ := json.Marshal(evt.Properties)
    _, err := l.db.Pool.Exec(ctx, `
        INSERT INTO product_events (user_id, event_name, properties, domain_slug, session_id)
        VALUES ($1, $2, $3, $4, $5)
    `, evt.UserID, evt.EventName, props, evt.DomainSlug, evt.SessionID)
    if err != nil {
        slog.Error("product event write failed", "event", evt.EventName, "error", err)
    }
}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| `request_type` | Categorizes audit log entries | `"search.listings"`, `"comps.run"` |
| `cache_hit` | HIT/MISS tracking per request | `"HIT"`, `"MISS"`, `nil` |
| `domain_slug` | Tenant isolation key from middleware | `c.Locals(ctxkeys.MLSDomainSlug)` |
| `X-IDX-Cache` | HTTP response header for cache status | `c.Set("X-IDX-Cache", "HIT")` |
| Funnel | Sequence of events leading to activation | signup → domain_add → domain_verify → first_api_call |
| Activation | First successful proxied API call after domain verification | `request_type` != nil, `domain_slug` verified |

## Common Patterns

### Adding Audit to a Dashboard Handler

**When:** Dashboard CRUD operations (domain verify, token create/revoke) need tracking.

```go
// new code to add — in dashboard handler after mutation
l.audit.LogEvent(c.Context(), audit.ProductEvent{
    UserID:     user.ID,
    EventName:  "dashboard.token.created",
    Properties: map[string]any{"token_name": req.Name, "abilities": req.Abilities},
    DomainSlug: nil,
})
```

### Querying Proxy Usage by Domain

**When:** Building usage reports or rate-limit decisions from existing data.

```sql
SELECT domain_slug,
       COUNT(*) AS total_requests,
       COUNT(*) FILTER (WHERE cache_hit = 'HIT') AS cache_hits,
       COUNT(DISTINCT request_type) AS unique_endpoints
FROM mls_proxy_audit_logs
WHERE logged_at > NOW() - INTERVAL '30 days'
GROUP BY domain_slug;
```

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **auth-api-token** skill for token lifecycle and auth middleware
- See the **cache-postgres** skill for cache HIT/MISS instrumentation surfaces
- See the **queue-postgresql** skill for worker/scheduler event patterns
- See the **fiber** skill for middleware and request context patterns
- See the **postgresql** skill for JSONB and aggregate query patterns