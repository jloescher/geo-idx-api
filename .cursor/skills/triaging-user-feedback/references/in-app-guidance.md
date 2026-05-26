# In-App Guidance Feedback

## Contents
- Guidance Surfaces in the Dashboard
- Error Message Quality
- Copy and Microcopy Patterns
- Guidance Improvement Patterns

## Guidance Surfaces in the Dashboard

The Quantyra IDX dashboard (`internal/handler/dashboard/handler.go`) is the primary in-app surface. It is server-rendered HTML using `internal/web/layout.go` templates with static assets from `internal/web/static/`.

### Current Guidance Points

| Surface | Location | Current Guidance |
|---------|----------|-----------------|
| DNS verification | `handler.go` 422 response | "TXT record not found. Publish the verification record at your DNS host, then try again." |
| Token display | Dashboard token box | One-time display with copy-to-clipboard |
| Login errors | `internal/handler/auth/handler.go` | "Invalid credentials." |
| Auth middleware | `internal/api/middleware/domain_token.go` | "Unauthenticated." / scope-specific errors |
| Domain not found | `handler.go` | "domain not found" |
| DNS lookup failure | `handler.go` | 502 with "DNS lookup failed" |

### CSS Design System for Guidance

`internal/web/static/css/app.css` provides these guidance-relevant components:

```css
/* Status indicators for domain verification */
.badge-pending   /* Pending status */
.badge-verified  /* Verified status */

/* Interactive elements */
.btn-primary     /* Primary CTA */
.btn-secondary   /* Secondary actions */
.btn-sm          /* Smaller buttons */

/* Token display */
.token-box       /* Dashed border token container */
```

## Error Message Quality

### DO: Provide actionable error messages

The DNS verification error is a good example — it tells the user what happened AND what to do:
```go
return c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")
```

### DON'T: Use generic error messages

The auth middleware error "Unauthenticated." tells the user nothing about what's wrong:
```go
// Current: generic
return fiber.NewError(fiber.StatusUnauthorized, "Unauthenticated.")

// Better: actionable
return fiber.NewError(fiber.StatusUnauthorized, "API token required. Create one at /dashboard/api-tokens.")
```

### WARNING: Error messages that expose internals

Avoid returning internal error details to API consumers. The codebase correctly uses structured slog for internal errors while returning user-friendly messages. Maintain this pattern:

```go
// Internal: detailed
w.logger.Error("job failed", "id", job.ID, "type", job.Payload.Type, "error", err)

// External: actionable
return fiber.NewError(fiber.StatusBadGateway, "Upstream MLS service unavailable. Try again in a few minutes.")
```

## Copy and Microcopy Patterns

### Token Creation Flow

Current: Token is displayed once in a `.token-box` with copy-to-clipboard via `data-copy` attribute (`internal/web/static/js/app.js`).

Improvement opportunities:
- Add "This token won't be shown again" warning before creation
- Show example curl command with the new token
- Link to API documentation from the token creation success screen

### Domain Verification Flow

Current: User adds domain → sees pending badge → manually retries verification.

Improvement opportunities:
- Auto-refresh verification status via polling
- Show the exact TXT record value prominently (not buried in instructions)
- Add a "Check again" button with visual feedback

## Guidance Improvement Patterns

### Pattern: Map Every User-Facing Error to a Help Resource

| Error | Current Response | Improvement |
|-------|-----------------|-------------|
| 422 DNS not found | Generic TXT message | Include the exact expected TXT value |
| 401 Unauthenticated | "Unauthenticated." | Link to token creation docs |
| 404 Domain not found | "domain not found" | Suggest checking domain slug |
| 502 DNS lookup | "DNS lookup failed" | Explain retry behavior |

### Pattern: Progress Indicators for Long Operations

The replication pipeline has observable stages (`replica_pages` → `listings`). The stats endpoint (`GET /api/v1/bridge/stats`) exposes this. Surface it in the dashboard for domains waiting for initial data population.

## Related Skills

- See the **frontend-design** skill for dashboard UI patterns
- See the **ux** skill for microcopy and accessibility patterns
- See the **auth-api-token** skill for auth error surface details