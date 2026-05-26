---
name: designing-onboarding-paths
description: |
  Designs onboarding paths, checklists, and first-run UI for the Quantyra IDX
  dashboard. Covers activation funnels, progressive setup flows, empty states,
  and first-run detection using server-rendered HTML (Go + Fiber).
  Use when: adding dashboard onboarding steps, first-run checklists, setup
  wizards, empty state guidance, domain verification UX, token issuance flows,
  or activation metrics based on audit logs.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Designing Onboarding Paths

Design activation flows for the invite-only Quantyra IDX dashboard: domain registration, DNS TXT verification, API key issuance, and first API call. The dashboard is server-rendered HTML (Go + Fiber) with embedded CSS/JS — no SPA framework. Onboarding improvements mean Go handler changes and HTML template updates.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving designing-onboarding-paths, verify against current docs FIRST:



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

### Existing Dashboard Handler Pattern

```go
// internal/handler/dashboard/handler.go — existing
func (h *Handler) Dashboard() fiber.Handler {
    return func(c *fiber.Ctx) error {
        userID := getSessionUserID(c)
        domains, _ := h.domainRepo.ListActiveForUser(userID)
        tokens, _ := h.tokenRepo.ListForUser(userID)
        // renders HTML with domains + tokens
    }
}
```

### New Code Pattern — First-Run Detection

```go
// new code to add — detect first-run state in Dashboard handler
func (h *Handler) Dashboard() fiber.Handler {
    return func(c *fiber.Ctx) error {
        userID := getSessionUserID(c)
        domains, _ := h.domainRepo.ListActiveForUser(userID)
        tokens, _ := h.tokenRepo.ListForUser(userID)
        isFirstRun := len(domains) == 0 && len(tokens) == 0
        // pass isFirstRun to template for onboarding checklist visibility
    }
}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| First-run detection | Query user's domains + tokens count | `len(domains) == 0` |
| Progressive setup | Ordered steps: domain → verify → token → first call | Dashboard checklist cards |
| Empty state | Conditional HTML when collections are empty | "Add domain" card when `len(domains) == 0` |
| Activation metric | Audit log query counting first API calls per domain | `SELECT domain_slug, MIN(created_at)` |

## Common Patterns

### Onboarding Checklist

**When:** User has zero domains or zero tokens on first dashboard visit.

```go
// new code to add — checklist state struct
type OnboardingState struct {
    HasDomain       bool
    DomainVerified  bool
    HasToken        bool
    HasFirstAPICall bool
}
```

### Empty State with Next Action

**When:** A collection (domains, tokens) is empty.

```html
<!-- new code to add — match existing card pattern from app.css -->
<div class="card">
    <h2>Add your first domain</h2>
    <p>Register the domain where your IDX site runs.</p>
    <form class="form-stack" method="POST" action="/dashboard/domains">
        <input name="hostname" placeholder="example.com" required>
        <button class="btn btn-primary">Add domain</button>
    </form>
</div>
```

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **ux** skill for UI/UX patterns and accessibility
- See the **fiber** skill for route registration and middleware
- See the **frontend-design** skill for CSS patterns and component styling
- See the **auth-api-token** skill for token creation and domain auth flows
- See the **go** skill for handler patterns and error handling