# In-App Guidance Reference

## Contents
- Guidance Surfaces in idx-api
- Dashboard Guidance Patterns
- API Response Guidance
- Anti-Patterns

---

## Guidance Surfaces in idx-api

idx-api has no SPA frontend — guidance must work through:

| Surface | Mechanism | File |
|---------|-----------|------|
| Dashboard HTML | Server-rendered templates | `internal/web/layout.go` |
| API responses | JSON error/info messages | Handler return values |
| Marketing page | Static HTML | `internal/handler/marketing/handler.go` |
| Health/status | `/healthz`, `/readyz`, `/api/v1/bridge/stats` | Various handlers |

## Dashboard Guidance Patterns

### DO: Show contextual next steps in dashboard

```go
// new code to add — in Dashboard handler, compute onboarding progress
func (h *Handler) Dashboard(c *fiber.Ctx) error {
    // Existing: load domains, tokens
    // Add: compute next step
    nextStep := ""
    if len(domains) == 0 {
        nextStep = "add_domain"
    } else if !hasVerifiedDomain(domains) {
        nextStep = "verify_domain"
    } else if len(tokens) == 0 {
        nextStep = "create_token"
    }
    // Pass nextStep to template
}
```

### DO: Use HTTP status codes and structured error messages for API guidance

```go
// Existing pattern from handler returns
return c.Status(401).JSON(fiber.Map{
    "error": "invalid or missing API token",
})
```

Extend with actionable guidance:

```go
// new code to add
return c.Status(403).JSON(fiber.Map{
    "error":   "domain not verified",
    "action":  "Complete DNS TXT verification at /dashboard",
    "doc_url": "https://idx-api.quantyralabs.cc/docs/auth",
})
```

## API Response Guidance

### DO: Include feature availability in metadata

```go
// new code to add — OpenAPI/endpoint discovery
// GET /api/v1 returns available endpoints and their requirements
{
  "endpoints": {
    "/api/v1/search":  {"auth": "token", "datasets": ["stellar", "beaches"]},
    "/api/v1/gis":     {"auth": "token", "scopes": ["idx:access"]},
    "/api/v1/comps/run": {"auth": "token", "modes": ["bpo", "home_value"]}
  }
}
```

### DON'T: Put guidance only in docs that users never read

API consumers learn from response shapes. If a new feature requires a scope, the 403 response must say which scope — not link to a doc page nobody opens.

## Anti-Patterns

### WARNING: Adding UI guidance via JavaScript in a Go template

The dashboard uses `internal/web/layout.go` string-builder HTML. Adding complex JS interactivity fights the architecture.

**Fix:** Keep guidance as server-rendered HTML fragments. For complex flows (wizards, tooltips), consider a lightweight template engine or static JS widget — but evaluate whether a simpler multi-page flow works first.

See the **fiber** skill for HTTP handler patterns and the **go** skill for template conventions.