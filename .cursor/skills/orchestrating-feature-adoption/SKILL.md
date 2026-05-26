---
name: orchestrating-feature-adoption
description: |
  Plans feature discovery, nudges, and adoption flows for the Quantyra IDX platform.
  Use when: adding feature flags, tiered access, onboarding steps, in-app guidance,
  telemetry for adoption metrics, progressive disclosure, GIS teaser tiers, or
  dashboard setup flows. Triggers: "feature adoption", "feature discovery", "nudge",
  "onboarding flow", "teaser tier", "progressive disclosure", "activation metric".
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Orchestrating Feature Adoption

Plan and implement feature discovery, progressive disclosure, and adoption measurement for Quantyra IDX — a multi-MLS proxy with invite-only dashboard, tiered GIS access, and config-driven feature flags.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving orchestrating-feature-adoption, verify against current docs FIRST:



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

### Existing Pattern: Tiered GIS Access

```go
// internal/service/gis/teaser.go — existing teaser tier
func requestFullAccess(c *fiber.Ctx) bool {
    // Domain auth always gets full access
    // PATs without idx:full get teaser-tier
}

func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
    // Caps features to cfg.TeaserMaxFeatures (40)
    // Reduces precision to cfg.TeaserCoordDecimals (4)
}
```

### Existing Pattern: Feature Flags via Config

```go
// internal/config/config.go — existing flag pattern
StellarEnabled  bool  // env: MLS_STELLAR_ENABLED
BeachesEnabled  bool  // env: MLS_BEACHES_ENABLED
```

### New Code Pattern: Adoption Event

```go
// new code to add — adoption event in audit logger
func (l *Logger) LogAdoption(c *fiber.Ctx, event string, meta map[string]any) {
    // INSERT into adoption_events (domain_slug, event, meta, created_at)
    // Fire-and-forget; do not block response
}
```

## Key Concepts

| Concept | Usage | Location |
|---------|-------|----------|
| Token abilities | `idx:full` vs `idx:access` tier gating | `domain_token.go` |
| Config flags | `StellarEnabled`, `BeachesEnabled` | `config.go` |
| Teaser tiers | Limit features/precision for lower tiers | `teaser.go` |
| Audit log | Per-domain/token usage tracking | `audit/logger.go` |
| Setup flow | Domain → verify → token (progressive) | `dashboard/handler.go` |
| Badge system | `.badge-verified`, `.badge-pending` | `app.css` |

## Common Patterns

### Adding a New Feature Flag

**When:** Gating a new capability behind config.

```go
// new code to add — in config.go
type MLSConfig struct {
    // ... existing fields
    CompsEnabled bool `env:"MLS_COMPS_ENABLED" envDefault:"true"`
}

// new code to add — in handler
if !h.cfg.MLS.CompsEnabled {
    return c.Status(http.StatusNotFound).SendString("feature not available")
}
```

### Gating by Token Ability

**When:** Restricting a feature to full-access tokens.

```go
// existing pattern from domain_token.go
fullAccess := c.Locals(ctxkeys.MLSFullAccess).(bool)
if !fullAccess {
    // return teaser or deny
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

- See the **fiber** skill for route registration and middleware patterns
- See the **auth-api-token** skill for token abilities and domain verification
- See the **cache-postgres** skill for caching strategy behind feature access
- See the **ux** skill for dashboard UI patterns and empty states
- See the **geospatial** skill for GIS teaser tier implementation
- See the **frontend-design** skill for badge, card, and form styling