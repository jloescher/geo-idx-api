---
name: designing-lifecycle-messages
description: |
  Designs onboarding and lifecycle email sequences for the Quantyra IDX platform.
  Use when: writing or updating user-facing messaging at lifecycle touchpoints (invitation, domain verification, token creation, API activation, re-engagement), adding email delivery to the invite/domain/token flows, or designing in-dashboard guidance copy in handler HTML templates.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Designing Lifecycle Messages

Design user-facing messages at each stage of the Quantyra IDX lifecycle: invitation → registration → domain verification → API token creation → activation → re-engagement. The platform currently has touchpoints in `internal/handler/dashboard/handler.go` and `internal/handler/marketing/handler.go` but no email delivery — MAIL\_\* env vars are configured but unused.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving designing-lifecycle-messages, verify against current docs FIRST:



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

### Existing touchpoints (no email delivery yet)

```go
// internal/handler/dashboard/handler.go:262 — invitation created, link shown once
link := "/invite/" + plain
body := `<div class="card"><h1>Invitation created</h1><p>Share this link (shown once):</p>...`
```

```go
// internal/handler/dashboard/handler.go:224 — domain verified, token shown once
plain, _ := h.tokens.Create(c.Context(), uid, "Production", []string{"idx:full"})
body := `<div class="card"><h1>Domain verified</h1><p>Save this production token now — it will not be shown again.</p>...`
```

### New code to add — email delivery via queue job

```go
// new code to add — internal/service/email/sender.go
// Wire to existing MAIL_* config; enqueue via PostgreSQL queue.
```

## Key Concepts

| Concept | Usage | Location |
|---|---|---|
| Lifecycle stages | Invite → Register → Verify → Token → Activate → Engage | `internal/service/auth/invitations.go` |
| Inline HTML copy | Dashboard messages rendered as string literals | `internal/handler/dashboard/handler.go` |
| Token one-time display | Tokens shown once, never stored plaintext | `CreateToken`, `CreateStagingToken` |
| Queue-based delivery | Enqueue email jobs for async worker processing | See **queue-postgresql** skill |

## Common Patterns

### Update dashboard inline copy

**When:** Changing any user-facing message in the dashboard.

```go
// internal/handler/dashboard/handler.go:124 — dashboard heading + description
b.WriteString(`<div class="card"><h1>Setup</h1><p>Register domains, verify DNS, and manage API keys.</p>...`)
```

### Add a new email touchpoint

**When:** Adding email delivery to an existing lifecycle stage.

```go
// new code to add — enqueue after invitation creation
// In CreateInvitation handler, after plain token is generated:
// job := queue.Job{Type: "email.send", Payload: map[string]any{...}}
// Enqueue to "default" queue for worker processing
```

## See Also

- [references/conversion-optimization.md](references/conversion-optimization.md)
- [references/content-copy.md](references/content-copy.md)
- [references/distribution.md](references/distribution.md)
- [references/measurement-testing.md](references/measurement-testing.md)
- [references/growth-engineering.md](references/growth-engineering.md)
- [references/strategy-monetization.md](references/strategy-monetization.md)

## Related Skills

- **queue-postgresql** — enqueue email delivery jobs
- **auth-api-token** — token creation and revocation lifecycle
- **frontend-design** — dashboard HTML and CSS patterns
- **ux** — in-dashboard guidance and empty states
- **cache-postgres** — email template caching
- **go** — Go service patterns for email delivery