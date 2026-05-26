---
name: generating-growth-hypotheses
description: |
  Generates channel experiments and growth loops for Quantyra IDX, a B2B MLS proxy API.
  Use when: planning acquisition channels, designing viral loops, instrumenting conversion funnels, building freemium tiers, or proposing growth experiments for the idx-api platform.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Generating Growth Hypotheses

Generate channel experiments and growth loops for Quantyra IDX — a B2B MLS proxy API with invite-only onboarding, GIS teaser tiering, domain-verified production tokens, and a Comps/BPO engine. Hypotheses must map to real code surfaces: the hero page, dashboard, invitation system, GIS freemium gate, and audit log data.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving generating-growth-hypotheses, verify against current docs FIRST:



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

### Hypothesis Template

```text
Hypothesis: If we [action], then [metric] will [change] because [reason].
Surface: [file or route]
Metric source: audit_logs / jobs / GIS teaser config / dashboard form submits
Risk: [what could go wrong]
Minimum viable test: [code change needed]
```

### Growth Loops in This Codebase

| Loop | Trigger | Flywheel | Surface |
|---|---|---|---|
| Domain activation | User adds domain → DNS verify → production token | More domains = more API calls = more value | `dashboard/handler.go:StoreDomain` → `VerifyTXT` |
| Viral invite | Admin sends invite → new user registers → adds own domains | Each user can invite more users | `dashboard/handler.go:CreateInvitation` → `AcceptInvitation` |
| GIS freemium | Unauthenticated request → teaser GeoJSON → upgrade prompt | Teaser shows value, full access requires `idx:full` token | `gis/teaser.go:applyTeaser` |
| Comps demonstration | `/api/v1/comps/run` with BPO mode → investor requests full report | API usage demonstrates ROI | `comps/handler.go:Run` |

## Key Concepts

| Concept | Usage | Surface |
|---|---|---|
| Freemium gate | `applyTeaser` truncates features and rounds coordinates for non-full-access callers | `internal/service/gis/teaser.go` |
| Invite-only growth | Admin-gated invitations via token URL (`/invite/:token`) | `internal/handler/dashboard/handler.go` |
| Domain verification | DNS TXT check triggers production token generation | `dashboard/handler.go:VerifyTXT` |
| Audit trail | All authenticated requests logged to `audit_logs` | `internal/service/audit/` |
| Staging token friction | One staging token per user; production requires domain verification | `dashboard/handler.go:CreateStagingToken` |

## Common Patterns

### Add a Conversion Event to Audit Logs

```go
// new code to add — in the relevant handler after a key action
h.audit.Log(c.Context(), uid, "domain.verified", map[string]any{
    "domain_id": id,
    "domain":    slug,
})
```

### Instrument GIS Teaser Conversion

```go
// new code to add — after applyTeaser returns truncated=true
if truncated {
    h.audit.Log(c.Context(), 0, "gis.teaser_truncated", map[string]any{
        "features_returned": maxFeatures,
        "original_count":    len(feats),
    })
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

- See the **auth-api-token** skill for token creation and revocation flows
- See the **cache-postgres** skill for audit log query patterns
- See the **geospatial** skill for GIS teaser configuration
- See the **fiber** skill for middleware-based conversion tracking
- See the **queue-postgresql** skill for async growth experiment jobs
- See the **proxy-web** skill for MLS proxy conversion paths