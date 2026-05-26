---
name: tuning-landing-journeys
description: |
  Improves landing page flow, hierarchy, and conversion paths for the Quantyra IDX platform.
  Use when: editing hero copy, dashboard layout, login/invite forms, domain verification flow,
  CTAs, or any HTML served by marketing/dashboard handlers. Also use when restructuring the
  signup-to-first-API-key journey or adding conversion tracking to platform pages.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Tuning Landing Journeys

Server-rendered Go/Fiber HTML pages with embedded CSS. No SPA framework. All pages are built from `web.Page()` or `web.LoginPage()` wrappers emitting inline HTML strings from handler functions. Conversion tuning means editing Go string literals in handlers, CSS variables in `app.css`, and layout helpers in `internal/web/layout.go`.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving tuning-landing-journeys, verify against current docs FIRST:



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

### Page Surfaces (existing routes)

| Route | Handler file | Layout | Purpose |
|-------|-------------|--------|---------|
| `/` | `marketing/handler.go` | `web.Page()` | Landing hero |
| `/login` | `dashboard/handler.go` | `web.LoginPage()` | Auth form |
| `/dashboard` | `dashboard/handler.go` | `web.Page()` | Domain + token management |
| `/invite/:token` | `dashboard/handler.go` | `web.LoginPage()` | Invite signup form |

### Editing hero copy (existing code)

```go
// internal/handler/marketing/handler.go:18-28
func (h *Handler) Home(c *fiber.Ctx) error {
    body := `<section class="hero">
<h1>Quantyra IDX</h1>
<p>MLS proxy, image delivery, and developer setup for your IDX sites.</p>
<div class="hero-actions">
<a class="btn btn-primary" href="/dashboard">Open dashboard</a>
<a class="btn btn-secondary" href="/login">Sign in</a>
</div>
</section>`
    return c.Type("html").SendString(web.Page("Home", body))
}
```

### Adding a new dashboard section (existing pattern)

```go
// new code to add — follows existing card pattern in Dashboard() at handler.go:124
b.WriteString(`<div class="card"><h2>Section title</h2><p>Description.</p>
<form method="post" action="/dashboard/your-action" class="inline-form">
<label>Field <input name="field" type="text" required></label>
<button type="submit" class="btn btn-primary">Submit</button>
</form></div>`)
```

## Key Concepts

| Concept | Where | Notes |
|---------|-------|-------|
| Page wrapper | `web.Page(title, body)` | Full HTML doc with header, nav, CSS link |
| Login wrapper | `web.LoginPage(body)` | Centered card, no nav header |
| HTML escaping | `web.Esc(s)` | Always use for user-supplied values in inline HTML |
| Design tokens | `app.css` `:root` variables | `--accent`, `--surface`, `--border`, `--radius`, etc. |
| Button styles | `.btn-primary` / `.btn-secondary` / `.btn-sm` | Primary = blue CTA, Secondary = ghost outline |
| Card container | `.card` class | Surface background, bordered, rounded |
| Form layout | `.form-stack` (vertical) / `.inline-form` (horizontal) | Choose based on field count |

## Common Patterns

### Change hero headline and subhead

Edit the string literal in `marketing/handler.go` `Home()`. Keep HTML minimal — no template engine, just raw strings.

### Reorder dashboard cards

Move `b.WriteString(...)` blocks in `Dashboard()`. First card = highest visual priority.

### Add a new CTA button

Use existing CSS classes. Primary for main action, secondary for alternatives:

```html
<a class="btn btn-primary" href="/path">Primary CTA</a>
<a class="btn btn-secondary" href="/path">Secondary CTA</a>
```

### One-time secret display

Follow `VerifyTXT` pattern: generate secret server-side, render in `.token-box`, do NOT store plaintext for re-display.

## See Also

- [conversion-optimization](references/conversion-optimization.md)
- [content-copy](references/content-copy.md)
- [distribution](references/distribution.md)
- [measurement-testing](references/measurement-testing.md)
- [growth-engineering](references/growth-engineering.md)
- [strategy-monetization](references/strategy-monetization.md)

## Related Skills

- See the **fiber** skill for route registration and middleware patterns
- See the **frontend-design** skill for CSS variable system and layout classes
- See the **ux** skill for form states, empty states, and accessibility
- See the **auth-api-token** skill for token creation and domain auth middleware