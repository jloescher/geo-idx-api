---
name: structuring-offer-ladders
description: |
  Frames plan tiers, value ladders, and upgrade logic for the Quantyra IDX API platform.
  Use when: designing pricing tiers, adding plan gating, building upgrade/teaser flows, creating
  pricing pages, extending the idx:access/idx:full dichotomy into a multi-tier offer ladder,
  or adding billing/subscription surfaces to the dashboard.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Structuring Offer Ladders

The platform currently operates a **two-tier access model** (`idx:access` vs `idx:full`) with GIS teaser gating as the sole revenue lever. This skill frames how to evolve that into a structured offer ladder — from adding plan definitions to wiring upgrade flows in the existing Go + Fiber stack.

**Current state:** Invite-only B2B API with domain-based auth. No billing, no pricing page, no subscription table. The only tier boundary is GIS teaser precision/features (`internal/service/gis/teaser.go`).

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving structuring-offer-ladders, verify against current docs FIRST:



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

### Existing Access Tier Pattern

```go
// internal/api/middleware/domain_token.go:52
fullAccess := tokens.HasAbility(tok, "idx:full")
setMLSLocals(c, "token", d, &tok.Name, &user.ID, fullAccess)
```

### New Code: Plan-Aware Gating

```go
// new code to add — extend the existing abilities check into plan tiers
func resolvePlanTier(tok repository.Token) string {
    if tok.HasAbility("idx:full") {
        return "pro" // full GIS, MLS, comps
    }
    if tok.HasAbility("idx:access") {
        return "starter" // teaser GIS, full MLS search
    }
    return "free" // unauthenticated or limited
}
```

## Key Concepts

| Concept | Implementation | Example |
|---------|---------------|---------|
| Access tier | `abilities` JSON on PAT | `["idx:full"]`, `["idx:access"]` |
| Teaser gate | `applyTeaser()` in GIS service | `internal/service/gis/teaser.go:24` |
| Full-access flag | `ctxkeys.MLSFullAccess` local | Set by `domain_token.go` middleware |
| Configurable limits | Env vars in `config.GISConfig` | `GIS_TEASER_MAX_FEATURES`, `GIS_TEASER_COORD_DECIMALS` |
| Marketing surface | `internal/handler/marketing/handler.go` | Server-side HTML, no template engine |

## Common Patterns

### Adding a Tier Boundary

**When:** Introducing a new plan level (e.g., "enterprise" between current tiers).

```go
// new code to add — in middleware, after token resolution
func planFromAbilities(tok repository.Token) PlanTier {
    switch {
    case tok.HasAbility("idx:enterprise"):
        return TierEnterprise
    case tok.HasAbility("idx:full"):
        return TierPro
    case tok.HasAbility("idx:access"):
        return TierStarter
    default:
        return TierFree
    }
}
```

### Pricing Page in Existing Pattern

**When:** Adding a `/pricing` route.

```go
// new code to add — follow marketing/handler.go pattern
func (h *Handler) Pricing(c *fiber.Ctx) error {
    body := `<section class="pricing">
<h1>Plans</h1>
<div class="pricing-grid">
  <div class="pricing-card"><h2>Starter</h2>...</div>
  <div class="pricing-card"><h2>Pro</h2>...</div>
</div></section>`
    return c.Type("html").SendString(web.Page("Pricing", body))
}
```

## See Also

- [conversion-optimization](references/conversion-optimization.md)
- [content-copy](references/content-copy.md)
- [distribution](references/distribution.md)
- [measurement-testing](references/measurement-testing.md)
- [growth-engineering](references/growth-engineering.md)
- [strategy-monetization](references/strategy-monetization.md)

## Related Skills

- See the **auth-api-token** skill for token/abilities implementation details
- See the **cache-postgres** skill for plan-limit caching strategies
- See the **frontend-design** skill for pricing page layout and styling
- See the **ux** skill for upgrade flow interaction patterns
- See the **geospatial** skill for GIS teaser gate internals