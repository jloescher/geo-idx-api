---
name: crafting-empty-states
description: |
  Creates empty states, onboarding affordances, and first-run guidance for the
  invite-only Quantyra IDX dashboard. Use when: adding empty states to dashboard
  lists (domains, tokens), designing first-run flows after invitation acceptance,
  adding inline help or setup progress indicators, improving the domain verification
  or API key creation UX, or crafting error/recovery states for DNS verification
  failures.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Crafting Empty States Skill

Design empty states, first-run guidance, and onboarding affordances for the server-rendered Quantyra IDX dashboard. The dashboard is invite-only, built with Go string-builder HTML templates (`internal/handler/dashboard/handler.go`), dark-theme CSS (`internal/web/static/css/app.css`), and embedded static assets (`internal/web/embed.go`). There is no frontend framework — all HTML is generated in Go handlers.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving crafting-empty-states, verify against current docs FIRST:



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

### Existing Pattern — Domain List (No Empty State)

```go
// internal/handler/dashboard/handler.go:124
b.WriteString(`<div class="card"><h1>Setup</h1><p>Register domains, verify DNS, and manage API keys.</p><ul class="domain-list">`)
for rows.Next() {
    // ... renders <li> per domain
}
b.WriteString(`</ul></div>`)
```

When `rows` is empty, the `<ul>` renders with zero children — invisible to the user.

### New Code Pattern — Empty State for Domain List

```go
// new code to add — replace the domain list section
b.WriteString(`<div class="card"><h1>Setup</h1><p>Register domains, verify DNS, and manage API keys.</p><ul class="domain-list">`)
domainCount := 0
for rows.Next() {
    domainCount++
    // ... existing <li> rendering
}
b.WriteString(`</ul>`)
if domainCount == 0 {
    b.WriteString(`<div class="empty-state"><p>No domains yet. Add your first domain below to get started.</p></div>`)
}
b.WriteString(`</div>`)
```

## Key Concepts

| Concept | Usage | CSS Class |
|---------|-------|-----------|
| Empty state message | Shown when a list has zero items | `.empty-state` (new) |
| Setup progress | Tracks domain verification + token creation steps | `.setup-progress` (new) |
| Inline guidance | Contextual help text near form fields | `<p>` with `--muted` color |
| Error recovery | Actionable message after verification failure | HTTP 422 response from `VerifyTXT` |
| First-run flow | Sequence after invite acceptance → login → dashboard | Redirect chain |

## Common Patterns

### Domain Verification Error State

**When:** DNS TXT record not found after user clicks "Verify TXT".

```go
// existing — internal/handler/dashboard/handler.go:214
return c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")
```

This returns plain text, not wrapped in the page layout. Wrapping it in `web.Page()` provides navigation back to the dashboard.

### Token Creation Success State

**When:** Domain verified, production token shown once.

```go
// existing — internal/handler/dashboard/handler.go:225
body := `<div class="card"><h1>Domain verified</h1><p>Save this production token now — it will not be shown again.</p>`
```

This is a strong one-time-reveal pattern with clear warning.

### First-Run After Invite

The current flow: invite link → registration form → redirect to `/login` → login → `/dashboard` with empty lists. A setup checklist would reduce time-to-first-API-call.

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **ux** skill for interaction patterns and accessibility
- See the **frontend-design** skill for layout, spacing, and visual patterns
- See the **fiber** skill for route registration and middleware
- See the **auth-api-token** skill for token creation and domain verification flow