---
name: improving-activation-flow
description: |
  Optimizes activation steps and time-to-value milestones for Quantyra IDX API.
  Use when: adding onboarding steps, improving first-run experience, reducing time to first successful API call, modifying dashboard token/domain flows, adding activation metrics, or redesigning the invite-only dashboard setup flow.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Improving Activation Flow Skill

Optimize the path from invite to first live listing query on the Quantyra IDX API. The critical path is: admin creates domain → customer receives invite → customer creates API token → customer makes first `GET /api/v1/properties` or `POST /api/v1/search` call → customer sees real MLS data.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving improving-activation-flow, verify against current docs FIRST:



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

### Activation Milestones (Current Flow)

```
1. Admin seeds domain via /dashboard
2. Customer receives domain authorization
3. Customer creates API token (POST /api/v1/dashboard/tokens)
4. Customer makes first MLS request (GET /api/v1/properties?dataset=stellar)
5. Customer verifies replication data (GET /api/v1/bridge/stats)
```

### New Code Pattern — Activation Event Hook

```go
// new code to add — fire an audit event on first successful API call per token
func (s *Service) RecordFirstAPIUse(ctx context.Context, tokenID string) error {
    _, err := s.db.ExecContext(ctx,
        `INSERT INTO audit_logs (action, subject_type, subject_id, created_at)
         VALUES ($1, $2, $3, NOW())
         ON CONFLICT DO NOTHING`,
        "token.first_use", "token", tokenID,
    )
    return err
}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Activation milestone | Measurable step toward value | Token created, first API 200, first listing returned |
| Time-to-value | Duration from domain creation to first successful query | Measure via `audit_logs.created_at` deltas |
| Invite gate | Dashboard is invite-only; admin seeds domains | `internal/handler/auth` domain validation |
| Token scope | API tokens have scopes controlling access | `idx:access`, `idx:admin` |

## Common Patterns

### Add an Activation Checkpoint

**When:** You need to track a new milestone in the onboarding path.

```go
// new code to add — checkpoint helper in a service method
func (s *Service) CheckActivationMilestone(ctx context.Context, domainID, milestone string) (bool, error) {
    var exists bool
    err := s.db.QueryRowContext(ctx,
        `SELECT EXISTS (
            SELECT 1 FROM audit_logs
            WHERE subject_type = 'domain'
              AND subject_id = $1
              AND action = $2
         )`,
        domainID, milestone,
    ).Scan(&exists)
    return exists, err
}
```

### Guard an Activation-Dependent Action

**When:** A feature should only be available after activation is complete.

```go
// new code to add — middleware to check domain has at least one token
func RequireActivatedDomain(db *sql.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        domainID := c.Locals("domain_id").(string)
        var hasToken bool
        _ = db.QueryRowContext(c.Context(),
            `SELECT EXISTS (SELECT 1 FROM tokens WHERE domain_id = $1 AND active = true)`,
            domainID,
        ).Scan(&hasToken)
        if !hasToken {
            return c.Status(403).JSON(fiber.Map{
                "error": "Create an API token before accessing this resource",
            })
        }
        return c.Next()
    }
}
```

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **auth-api-token** skill for token creation and scoping patterns
- See the **ux** skill for dashboard UI patterns
- See the **frontend-design** skill for dashboard layout and empty states
- See the **queue-postgresql** skill for background activation jobs
- See the **cache-postgres** skill for caching activation state
- See the **fiber** skill for middleware and route patterns