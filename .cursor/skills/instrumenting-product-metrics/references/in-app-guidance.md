# In-App Guidance Reference

## Contents
- Dashboard as the Guidance Surface
- API Response Headers as Guidance
- Error Message Design as Guidance
- Anti-Patterns

## Dashboard as the Guidance Surface

idx-api's in-app surface is the **invite-only dashboard** at `/dashboard` — server-rendered HTML via `internal/handler/dashboard/handler.go`. This is the only place where contextual guidance can appear.

### Dashboard Routes and Guidance Opportunities

| Route | Current State | Guidance Opportunity |
|-------|---------------|---------------------|
| `GET /dashboard` | Lists domains + tokens | Show activation checklist, next steps |
| `POST /dashboard/domains` | Creates pending domain | Explain DNS TXT verification |
| `POST /dashboard/domains/:id/verify-txt` | DNS check | Show next step after verification |
| `POST /dashboard/api-tokens` | Creates token | Show API usage examples with new token |
| `DELETE /dashboard/api-tokens/:id` | Deletes token | Warn about active integrations |

### No Frontend Framework

The dashboard uses embedded static HTML from `internal/web/static/`. There is no React, Vue, or SPA framework. Guidance must be:

1. Server-rendered in Go templates or inline HTML
2. Embedded via `embed.FS` (see `internal/web/`)
3. Simple — no client-side JS frameworks available

### Pattern: Guidance in Handler Response

```go
// new code to add — flash message after domain verification
func (h *Handler) VerifyTXT(c *fiber.Ctx) error {
    // ... verification logic ...
    if verified {
        session.Set("flash", "Domain verified! Your production API token has been created. Try your first request:")
        session.Set("flash_type", "success")
    }
    return c.Redirect("/dashboard")
}
```

## API Response Headers as Guidance

The API communicates operational state through HTTP headers — machine-readable guidance for integrators:

| Header | Values | Set In | Purpose |
|--------|--------|--------|---------|
| `X-IDX-Cache` | `HIT`, `MISS` | `internal/handler/bridge/handler.go` | Guide caching strategy |
| `X-Dataset` | `stellar`, `beaches` | MLS middleware | Confirm which MLS feed responded |

### Pattern: Add Guidance Headers for New Features

```go
// new code to add — usage tier header
c.Set("X-IDX-Usage-Tier", "standard")
c.Set("X-IDX-RateLimit-Remaining", "995")
```

## Error Message Design as Guidance

API errors in `internal/handler/bridge/handler.go` and `internal/api/middleware/` are the primary guidance integrators receive. Errors should be actionable.

### DO: Actionable Error Responses

```go
// GOOD — tells the user exactly what to do
c.Status(403).JSON(fiber.Map{
    "error": "Domain not verified. Complete DNS TXT verification at /dashboard.",
    "domain": slug,
})
```

### DON'T: Vague Errors

```go
// BAD — no actionable information
c.Status(403).JSON(fiber.Map{"error": "Forbidden"})
```

The domain token middleware (`internal/api/middleware/domain_token.go`) returns specific errors: `"domain not found"`, `"domain is not verified"`, `"invalid token"`, `"token expired"`. New endpoints should follow this pattern.

## Anti-Patterns

### WARNING: Client-Side Guidance for an API Product

idx-api is an API proxy. Integrators don't use a browser to call `/api/v1/search`. Guidance belongs in:
1. **API error messages** — actionable JSON errors
2. **HTTP headers** — `X-IDX-*` custom headers
3. **Dashboard** — the only HTML surface for account setup

Do NOT add tooltip/popover patterns meant for browser-based SPAs.

### WARNING: Overloading Dashboard HTML

The dashboard is server-rendered with basic HTML. Do not add complex JS-driven guidance (tours, modals, hotspots). Keep guidance to:
- Flash messages after mutations
- Status indicators (verified/pending, token count)
- Plain text next-step hints

See the **fiber** skill for response rendering and session management.
See the **auth-api-token** skill for domain verification and token creation flows.