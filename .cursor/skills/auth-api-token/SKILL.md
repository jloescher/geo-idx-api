---
name: auth-api-token
description: |
  Manages API token authentication with audit logging for the Quantyra IDX API.
  Use when: creating or validating API tokens, modifying auth middleware, adding audit logging,
  working with domain verification, implementing token scopes (idx:full, idx:access),
  or changing the DomainToken middleware flow.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Auth API Token Skill

API tokens (`idx_` + 64-char hex) are SHA-256 hashed at rest in `personal_access_tokens`. Authentication is dual-mode: **Bearer token** (PAT + domain slug) or **domain header/Referer** (no token). All authenticated MLS traffic is audit-logged to `mls_proxy_audit_logs`.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving auth-api-token, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.

Risk signals this skill can participate in:
- secrets/env wiring: Wire values through the repo config/runtime boundary and typed context. Do not assume process-level env access in edge/serverless runtimes.
- API/auth flow: Trace request context, validation, auth/session state, and response shape before editing. Keep privacy and anti-enumeration semantics stable.



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

### Token Authentication Flow (Existing)

```go
// internal/api/middleware/domain_token.go — Bearer path
// 1. Extract "Bearer idx_..." from Authorization header
// 2. SHA-256 hash → lookup in personal_access_tokens
// 3. Check ability (idx:access or idx:full)
// 4. Resolve domain slug (X-Domain-Slug header or ?domain= query)
// 5. Verify domain ownership + TXT verification
// 6. Set Fiber Locals via setMLSLocals()
```

### Adding a New Protected Route

```go
// new code to add — register route with DomainToken middleware
// See the **fiber** skill for router setup
authGroup := app.Group("/api/v1", middleware.DomainToken(cfg, domainRepo, tokenRepo))
authGroup.Post("/my-endpoint", myHandler)
```

## Key Concepts

| Concept | Usage | Location |
|---------|-------|----------|
| Token format | `idx_` + 64-char hex, SHA-256 at rest | `repository/token.go:HashToken` |
| Abilities | `["idx:full"]` or `["idx:access"]` JSON array | `domain.APIToken.Abilities` |
| Domain verification | TXT DNS record, `verified` or `verified_ghl` status | `domain.Domain.IsVerified()` |
| Context locals | `MLSAuth`, `MLSDomain`, `MLSFullAccess`, etc. | `ctxkeys/keys.go` |
| Audit logging | Fire-and-forget INSERT per request | `audit/logger.go` |
| Legacy tokens | Sanctum `id|secret` format — NOT supported | Returns nil (re-issue required) |

## Common Patterns

### Reading Auth State in Handlers

```go
// Existing pattern — extract from Fiber Locals after middleware
slug, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
fullAccess, _ := c.Locals(ctxkeys.MLSFullAccess).(bool)
var tokenName *string
if tn, ok := c.Locals(ctxkeys.MLSTokenName).(*string); ok {
    tokenName = tn
}
```

### Audit Logging After Handler Logic

```go
// Existing pattern — from internal/service/audit/logger.go
auditLogger.Log(c, "search", &listingCount, &cacheHit)
// nil-safe: does nothing if logger is nil
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- **go** — Go language patterns used throughout
- **fiber** — HTTP router and middleware registration
- **auth-domain** — Domain verification and registration flow
- **postgres** — Token and audit table schemas
- **queue-postgresql** — Background job processing context