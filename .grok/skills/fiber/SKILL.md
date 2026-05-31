---
name: fiber
description: |
  Configures Fiber v2 HTTP router with middleware support.
  Use when: implementing or refactoring Fiber work, troubleshooting routes, services, database, or aligning new changes with the repository's existing conventions. Also invoked via /fiber.
when-to-use: Fiber routes, middleware, handlers, Fiber app setup, HTTP concerns in idx-api
user_invocable: true
allowed-tools: read_file, search_replace, write, run_terminal_command, list_dir, grep, spawn_subagent, todo_write, ask_user_question
---

# Fiber Skill

This fallback skill keeps Fiber work aligned with the conventions already present in this repository. Prefer extending the closest existing implementation over inventing a new abstraction, and verify neighboring states before finishing.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving fiber, verify against current docs FIRST:


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

### Inspect the current implementation

```sh
rg -n "fiber|routes|services|database" .
rg --files | rg "fiber|routes|services"
```

### Make the smallest compatible change

- Keep transport concerns thin and push reusable business logic into the layers this repo already uses.
- Match the current validation, auth, and error-shaping patterns before introducing new helpers.
- Preserve the existing request and response contract unless the task explicitly requires a change.

### Verify before finishing

- Recheck validation, auth, and error paths alongside the happy path.
- Confirm downstream callers still receive the shape and status semantics they expect.
- Audit logging, retries, or persistence side effects if the change touches them.

## Key Concepts

| Concept | Why it matters | What to check |
|---------|----------------|---------------|
| Existing patterns | Keeps the repo coherent | Start from the nearest matching implementation before editing |
| Scope control | Prevents abstraction creep | Keep the change in the same layer as surrounding code |
| Verification | Catches regressions early | Recheck adjacent states, edge cases, and integration points |
| References | Speeds up repeat work | Use the linked topic files when the task needs deeper guidance |

## Common Patterns

### Routes

**When:** The task touches routes in Fiber work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

### Services

**When:** The task touches services in Fiber work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

### Database

**When:** The task touches database in Fiber work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

## See Also

- [Routes](references/routes.md)
- [Services](references/services.md)
- [Database](references/database.md)
- [Auth](references/auth.md)
- [Errors](references/errors.md)
