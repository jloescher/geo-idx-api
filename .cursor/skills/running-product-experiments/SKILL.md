---
name: running-product-experiments
description: |
  Guides implementation of product experiments, feature rollouts, and metrics instrumentation
  for the idx-api Go backend. Use when: adding feature flags, running A/B tests on API behavior,
  instrumenting activation funnels, rolling out new MLS/GIS features to subsets of domains,
  measuring adoption of comps or search endpoints, or gating access to new product surfaces.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Running Product Experiments

idx-api is a Go backend with no experiment framework. Feature control lives in `internal/config/config.go` (env-var booleans like `MLS_STELLAR_ENABLED`, `MLS_BEACHES_ENABLED`). The audit log (`internal/service/audit/logger.go`) captures proxy request metadata. The dashboard (`internal/handler/dashboard/handler.go`) manages domains, tokens, and invitations. Experiments must be wired through these surfaces — no frontend A/B, no client-side event tracking.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving running-product-experiments, verify against current docs FIRST:



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

### Verified Existing Pattern — env-var feature flag

```go
// internal/config/config.go — existing pattern
StellarEnabled: envBool("MLS_STELLAR_ENABLED", true),
BeachesEnabled: envBool("MLS_BEACHES_ENABLED", true),

// internal/service/sync/kickoff.go — consumption
if !h.cfg.MLS.StellarEnabled {
    slog.Info("stellar disabled, skipping")
    return nil
}
```

### New Code Pattern — DB-backed per-domain feature flag

```go
// new code to add — migrations/YYYYMMDDHHMMSS_feature_flags.sql
// CREATE TABLE feature_flags (
//   id          SERIAL PRIMARY KEY,
//   flag_key    TEXT NOT NULL UNIQUE,
//   domain_slug TEXT,              -- NULL = global
//   enabled     BOOLEAN NOT NULL DEFAULT FALSE,
//   rollout_pct SMALLINT DEFAULT 100,  -- 0-100
//   active_from TIMESTAMPTZ,
//   active_until TIMESTAMPTZ,
//   created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
// );
// CREATE INDEX idx_feature_flags_key_domain ON feature_flags (flag_key, domain_slug);
```

## Key Concepts

| Concept | Usage | Pattern |
|---------|-------|---------|
| Env-var flag | Simple on/off for all tenants | `config.go` struct field |
| DB-backed flag | Per-domain or gradual rollout | `feature_flags` table + repository method |
| Audit event | Proxy request tracking | `audit.Logger.Log(c, requestType, &count, &hit)` |
| Deterministic hash | Stable % rollout per domain | `crc32.ChecksumIEEE([]byte(domainSlug)) % 100 < rolloutPct` |
| Migration-first | Schema changes before code | `migrations/` Goose SQL files |

## Common Patterns

### Evaluate a feature flag at request time

**When:** Gating a new endpoint or behavior behind a flag

```go
// new code to add — in handler or middleware
func (h *Handler) isEnabled(c *fiber.Ctx, flagKey string) bool {
    domain := c.Locals("domain_slug")
    flag, err := h.flags.Get(c.Context(), flagKey, domain)
    if err != nil {
        slog.Error("flag eval failed", "flag", flagKey, "err", err)
        return false // fail closed
    }
    return flag.Enabled
}
```

### Extend audit log with experiment metadata

**When:** Tracking which variant a request saw

```go
// new code to add — extend audit logger call
h.audit.Log(c, "search", &count, &cacheHit)
// Add experiment_id / variant columns to mls_proxy_audit_logs
// via migration, then include in Log() call
```

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **auth-api-token** skill for token scoping and permission gates
- See the **cache-postgres** skill for caching experiment configurations
- See the **queue-postgresql** skill for async event processing
- See the **postgresql** skill for migration patterns and JSONB queries
- See the **go** skill for Go project conventions
- See the **fiber** skill for HTTP middleware and route registration