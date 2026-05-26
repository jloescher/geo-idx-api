# State Matrix Reference

## Contents
- Why State Matrices Matter
- API Response States
- Dashboard Form States
- Replication/Job States
- State Anti-Patterns

## Why State Matrices Matter

Before editing any interactive surface, define every possible state. Missing states cause: blank screens, stuck spinners, confusing errors, and data loss. A state matrix is a table of (component × state) where every cell must have a defined rendering.

## API Response States

Every API endpoint must handle these states consistently:

| State | HTTP code | Response shape | When |
|-------|-----------|---------------|------|
| Success | 200/201 | `{ "value": [...], "@odata.count": N }` | Normal response |
| Validation error | 400 | `{ "error": { "code": "...", "message": "..." } }` | Bad input |
| Unauthenticated | 401 | `{ "error": { "code": "UNAUTHENTICATED" } }` | Missing token |
| Forbidden | 403 | `{ "error": { "code": "FORBIDDEN", "message": "..." } }` | Wrong scope or domain |
| Not found | 404 | `{ "error": { "code": "NOT_FOUND" } }` | Resource missing |
| Conflict | 409 | `{ "error": { "code": "CONFLICT", "message": "Name already exists" } }` | Duplicate create |
| Upstream timeout | 502/504 | `{ "error": { "code": "UPSTREAM_TIMEOUT" } }` | MLS provider down |
| Rate limited | 429 | `{ "error": { "code": "RATE_LIMITED" } }` | Too many requests |

### WARNING: Inconsistent Error Shapes

**The Problem:**

```go
// BAD — sometimes string, sometimes object, sometimes raw Fiber error
c.SendString("something went wrong")
c.JSON(fiber.Map{"msg": "bad input"})
c.SendStatus(500)
```

**Why This Breaks:** API consumers cannot reliably parse errors. Each endpoint returning a different shape forces consumers to write per-endpoint error handling.

**The Fix:**

```go
// GOOD — consistent error envelope across all endpoints
func ErrorResponse(c *fiber.Ctx, status int, code, message string) error {
    return c.Status(status).JSON(fiber.Map{
        "error": fiber.Map{"code": code, "message": message},
    })
}
```

**When You Might Be Tempted:** Quick prototypes where "just returning a string" feels faster. This debt compounds immediately.

## Dashboard Form States

| Component | Idle | Pending | Success | Error | Disabled |
|-----------|------|---------|---------|-------|----------|
| Submit button | "Create" enabled | "Creating…" disabled | "Created ✓" briefly | "Create" re-enabled | Greyed when prerequisites missing |
| Form fields | Editable | Read-only | Reset or hidden | Editable + error text | Greyed |
| Toast/banner | Hidden | Hidden | Success message | Error with retry link | — |
| Token display | Hidden | Hidden | Token + copy button | Hidden | — |

## Replication/Job States

See the **queue-postgresql** skill for queue implementation. States visible in `GET /api/v1/bridge/stats`:

| `replica_pages` status | Meaning | User-visible indicator |
|------------------------|---------|----------------------|
| `pending` | Queued, waiting for worker | "Replication queued" |
| `processing` | Worker fetching/persisting | "Replication in progress" |
| `completed` | Persisted to `listings` | Not shown (transient) |
| `failed` | Worker error | Error in worker logs, next kickoff retries |

`listings` mirror status: count returned by stats endpoint. No per-row status exposed to API consumers.

## State Anti-Patterns

1. **Missing "empty" state** — A table with zero rows shows nothing. Show "No tokens yet. Create one to get started."
2. **Missing "loading" state** — Content area is blank while fetching. Show a skeleton or spinner.
3. **Missing "disabled" state** — Submit button is clickable when prerequisites aren't met. Disable and explain why.
4. **Conflating states** — Using the same UI for "loading" and "empty" (both show nothing). They are different states requiring different treatment.