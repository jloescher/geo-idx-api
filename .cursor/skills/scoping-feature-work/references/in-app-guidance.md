# In-App Guidance Reference

Guidance patterns for the Quantyra IDX dashboard and API responses.

## Contents
- Dashboard Guidance Surfaces
- API Response Guidance
- Scoping Guidance Features

## Dashboard Guidance Surfaces

The dashboard is server-rendered HTML (no JS framework). Guidance is inline in the handler output.

| Surface | Location | Current state |
|---------|----------|---------------|
| Domain verification instructions | `VerifyTXT()` response | Error message: "TXT record not found..." |
| Token display | `VerifyTXT()` success | One-time display with copy target |
| Staging token limit | `CreateStagingToken()` | 409 "Staging token already exists" |
| Empty state | `Dashboard()` | No domains / no tokens messaging missing |

### WARNING: No empty-state guidance

**The Problem:** New users see a blank dashboard after login with no calls-to-action. The domain list and token list render empty `<ul>` elements.

**The Fix:** When scoping dashboard work, always include empty-state acceptance criteria:

```
Given user has no domains
When viewing /dashboard
Then "Add your first domain" CTA is visible
And the domain form is pre-focused
```

## API Response Guidance

API responses guide developers integrating the proxy. Two patterns exist:

### Cache headers

```go
// Bridge handler sets cache status
c.Set("X-IDX-Cache", "HIT") // or "MISS"
```

These headers guide integrators toward understanding cache behavior without reading docs.

### Stats endpoint

```go
// internal/api/routes.go:93
v1.Get("/bridge/stats", bridgeH.Stats)
```

`GET /api/v1/bridge/stats` returns replication state per dataset. This is the primary "health dashboard" for API consumers.

## Scoping Guidance Features

### Acceptance criteria for guidance changes

```
Given [user state: new / active / error]
When [user encounters surface]
Then [guidance message appears]
And [message includes next action or link]
```

### Pattern: Inline help in dashboard

The dashboard uses card-based layout (`<div class="card">`). Add guidance as a `<p class="help-text">` inside cards:

```html
<!-- new code to add -->
<p class="help-text">
  Add the hostname where your website is hosted.
  We'll verify ownership via a DNS TXT record.
</p>
```

### Pattern: Error response guidance in API

API errors should include a `docs_url` field pointing to the relevant documentation:

```go
// new code to add
return c.Status(400).JSON(fiber.Map{
    "error":    "Dataset not found",
    "docs_url": cfg.PlatformURL + "/docs/datasets",
})
```

## Anti-Patterns

### WARNING: Blocking guidance behind JavaScript

The dashboard must work without JavaScript. Forms use native `method="post"`. Guidance text is server-rendered HTML. Do not add JS-dependent tooltips, modals, or wizards.

### WARNING: Over-engineering dashboard UX

The dashboard is a self-service tool for domain/token management. It is NOT the product. Scope dashboard changes as small, targeted improvements — not a SPA rebuild.

## See Also

- See the **ux** skill for dashboard design patterns
- See the **frontend-design** skill for CSS and layout patterns