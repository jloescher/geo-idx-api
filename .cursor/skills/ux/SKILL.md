---
name: ux
description: |
  Improves dashboard flows, authentication paths, and API error states for the idx-api platform.
  Use when: adding or editing dashboard HTML/pages, authentication/token flows, API error responses, form validation feedback, CLI output formatting, or any interactive surface exposed to end users or operators.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# UX Skill

Improves interaction quality for dashboard, auth, and API surfaces in the idx-api Go/Fiber platform. The dashboard is server-rendered static HTML embedded via `internal/web/static/`. API consumers expect RESO-compliant JSON. Operators interact via CLI logs and Coolify health endpoints.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving ux, verify against current docs FIRST:



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

## Journey Map

Map the path the user is trying to complete before touching UI code: entry point, decision point, action, server response, confirmation, and recovery. For this repo, inspect forms, dialogs, settings, dashboards, onboarding, checkout, or CLI flows touched by the task.

## User Intent

- Name the user's immediate job and the anxiety/risk around it.
- Preserve the surrounding flow's existing mental model, navigation, and terminology.
- Avoid premature success copy; say what happened, what is pending, and what the user can do next.

## State Matrix

Cover loading, empty, error, disabled, pending, success, and recovery. A flow is incomplete if it only implements the happy path or relies on backend errors surfacing as raw messages.

## Failure + Recovery

- Explain failures in user-safe language.
- Keep privacy and anti-enumeration behavior for auth, account, invite, checkout, and recovery flows.
- Provide a retry, resend, return, or contact-support path when the user can reasonably recover.

## Accessibility Contract

Verify labels, focus states, keyboard flow, semantics, and contrast. Prefer native controls and existing accessible primitives before custom interaction code.

## Microcopy Rules

- Match local product voice and nearby copy.
- Keep labels explicit, errors actionable, and pending states honest.
- Do not use vague copy such as "Something went wrong" when a safe, specific recovery instruction is possible.

## Acceptance Checklist

- The primary journey and all meaningful states are represented.
- Validation, disabled states, loading states, success/failure feedback, and recovery copy are present where relevant.
- The UI remains understandable on mobile/desktop or the target CLI/desktop/mobile surface.
- The implementation uses local component and accessibility patterns.

## Quick Start

### Verified Existing Pattern

Dashboard HTML is embedded static assets served by Fiber. API responses are JSON with RESO field names.

```go
// Fiber route serves embedded dashboard — internal/api/routes.go pattern
app.Get("/dashboard", handler.Dashboard)
```

### New Code Pattern

```go
// new code to add — structured error response for API consumers
c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
    "error": fiber.Map{
        "code":    "INVALID_DATASET",
        "message": "Dataset must be 'stellar' or 'beaches'",
    },
})
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| State matrix | Define before coding any interactive flow | Loading → Empty → Error → Success → Retry |
| Journey map | User intent → precondition → action → outcome | Auth: request → validate domain → check token → respond |
| Microcopy | Concise, state-specific text | "Token created. Copy it now — it won't be shown again." |
| Accessibility | Native semantics, keyboard paths, labels | `<label>` on every form field, focus on error |

## Common Patterns

### API Error Response

**When:** Any non-2xx API response.

```go
// new code to add
c.Status(code).JSON(fiber.Map{
    "error": fiber.Map{
        "code":    "UPSTREAM_TIMEOUT",
        "message": "MLS provider did not respond in time. Try again.",
    },
})
```

### Dashboard Form State

**When:** Dashboard forms (domain management, token creation).

```html
<!-- new code to add — disable submit during pending state -->
<button type="submit" id="createToken" disabled>
  <span data-state="idle">Create Token</span>
  <span data-state="pending" hidden>Creating…</span>
</button>
```

## See Also

- [journey-map](references/journey-map.md)
- [state-matrix](references/state-matrix.md)
- [forms](references/forms.md)
- [accessibility](references/accessibility.md)
- [microcopy](references/microcopy.md)

## Related Skills

- **fiber** — HTTP routing and middleware patterns
- **auth-api-token** — Token auth implementation details
- **auth-domain** — Domain-based auth flow
- **frontend-design** — Visual layout and component patterns
- **go** — Go language patterns used in handlers