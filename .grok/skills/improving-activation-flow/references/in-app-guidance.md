# In-App Guidance Reference

## Contents
- Dashboard guidance surfaces
- Empty state patterns
- Error state messaging
- API response guidance
- Anti-patterns

## Dashboard Guidance Surfaces

The `/dashboard` is the primary in-app surface. It manages domains and API tokens.

### DO: Show clear next-action guidance on dashboard entry

```go
// new code to add — dashboard state response with next-action hints
type DashboardState struct {
    HasDomains    bool     `json:"has_domains"`
    HasTokens     bool     `json:"has_tokens"`
    HasFirstCall  bool     `json:"has_first_call"`
    NextAction    string   `json:"next_action"`
    NextActionURL string   `json:"next_action_url"`
}

func DetermineNextAction(state DashboardState) string {
    switch {
    case !state.HasDomains:
        return "Add your first domain to get started"
    case !state.HasTokens:
        return "Create an API token for your domain"
    case !state.HasFirstCall:
        return "Make your first API request to verify connectivity"
    default:
        return "Explore search, GIS, and comps endpoints"
    }
}
```

### DON'T: Show marketing copy in the dashboard

Dashboard users are already authenticated. They need operational guidance, not value propositions. "Create an API token" — not "Unlock the power of MLS data."

## Empty State Patterns

Empty states appear when: no domains, no tokens, no listings, no replication data.

### DO: Tie empty states to actionable next steps

| State | Display | Action |
|-------|---------|--------|
| No domains | "No domains authorized" | "Add a domain" button (admin only) |
| No tokens | "No API tokens created" | "Create a token" button |
| No listings | "Replication in progress..." | Show `GET /api/v1/bridge/stats` link |
| Search empty | "No listings match your criteria" | Suggest broadening filters |

### DON'T: Show raw database state to customers

```
// BAD — internal implementation detail exposed
"replica_pages count: 0, replication_in_progress: false"

// GOOD — customer-facing language
"Your MLS data feed is being configured. Check status in a few minutes."
```

## Error State Messaging

### DO: Map internal errors to customer-actionable messages

```go
// new code to add — error message mapper for API responses
func CustomerErrorMessage(err error) string {
    switch {
    case IsDomainNotFound(err):
        return "Domain not authorized. Contact your account administrator."
    case IsTokenRevoked(err):
        return "API token has been revoked. Create a new token from the dashboard."
    case IsReplicationStale(err):
        return "Data is temporarily unavailable. Try again in a few minutes."
    default:
        return "An unexpected error occurred. Please try again."
    }
}
```

### DON'T: Expose internal error details

```go
// BAD — leaks database schema and stack info
return c.Status(500).JSON(fiber.Map{"error": err.Error()})

// GOOD — generic message to client, logged internally
slog.Error("handler error", "error", err, "path", c.Path())
return c.Status(500).JSON(fiber.Map{"error": "Internal server error"})
```

## API Response Guidance

API responses should include hints when the response suggests a next step.

### DO: Include `@odata.nextLink` for paginated results

The Bridge and Spark proxy already passes through OData pagination links. Mirror-backed search should match this pattern.

### DO: Return dataset availability in metadata

```go
// new code to add — include available datasets in search response
type SearchMeta struct {
    Dataset      string `json:"dataset"`
    Source       string `json:"source"`       // "postgis" or "live" or "hybrid"
    TotalResults int    `json:"totalResults"`
}
```

## Anti-patterns

### WARNING: Adding tooltips or modals to a Go API backend

In-app guidance for this project is delivered through API response shapes and the embedded dashboard static assets. The API itself has no JavaScript runtime. Guidance changes go in:

1. API response messages and metadata (Go handlers)
2. Dashboard static assets (`internal/web/static/`)
3. Documentation (`docs/`)

See the **frontend-design** skill for dashboard UI patterns and the **ux** skill for interaction design.