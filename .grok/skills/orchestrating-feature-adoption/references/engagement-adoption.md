# Engagement & Adoption Reference

## Contents
- Feature Discovery Mechanisms
- Tiered Access Model
- Cross-Sell Patterns
- Anti-Patterns

## Feature Discovery Mechanisms

The platform has limited explicit discovery mechanisms. Features are revealed through:

1. **Config flags** — `StellarEnabled`, `BeachesEnabled` control MLS feed availability
2. **Token abilities** — `idx:full` vs `idx:access` gate GIS and advanced features
3. **Dashboard cards** — domain/token management surfaces capabilities

### Existing Discovery Points

| Surface | Mechanism | Code Location |
|---------|-----------|---------------|
| MLS dataset selection | Domain setup dropdown | `dashboard/handler.go` |
| GIS teaser | Limited features in response | `teaser.go` |
| Search endpoint | `POST /api/v1/search` | `bridge/handler.go` |
| Comps/BPO | `POST /api/v1/comps/run` | `comps/handler.go` |
| Bridge stats | `GET /api/v1/bridge/stats` | `bridge/handler.go` |

## Tiered Access Model

The platform uses a two-tier model enforced through middleware and service logic:

### Tier Implementation

```go
// existing pattern — internal/api/middleware/domain_token.go
// Token abilities: "idx:full" or "idx:access"
// Domain identification always grants full access
// setMLSLocals() sets MLSFullAccess boolean in context
```

```go
// existing pattern — internal/service/gis/teaser.go
func requestFullAccess(c *fiber.Ctx) bool {
    // Domain auth → full
    // PAT with idx:full → full
    // PAT with idx:access → teaser (if enabled)
}

func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
    // Truncate to TeaserMaxFeatures (default 40)
    // Round to TeaserCoordDecimals (default 4)
}
```

### Tier Matrix

| Feature | `idx:access` | `idx:full` | Domain Auth |
|---------|-------------|-----------|-------------|
| MLS listings proxy | Yes | Yes | Yes |
| GIS parcels | Teaser (40 features, 4 decimals) | Full | Full |
| Search (PostGIS) | Yes | Yes | Yes |
| Comps/BPO | Yes | Yes | Yes |
| Bridge stats | Yes | Yes | Yes |

## Cross-Sell Patterns

### GIS Teaser as Discovery

The teaser tier is the primary cross-sell mechanism. When `idx:access` tokens receive truncated GIS data, the response signals that more is available:

```go
// existing pattern — teaser.go returns (modifiedGeoJSON, wasTruncated)
// wasTruncated signals the handler to add a response header or adjust cache TTL
```

**Design principle:** The teaser returns useful but incomplete data — enough for a developer to validate the endpoint, not enough for production use.

### WARNING: Do Not Add Paywall UI to API Responses

**The Problem:**

```go
// BAD — injecting upgrade messages into API JSON
response["upgrade_message"] = "Upgrade to idx:full for full GIS access"
```

**Why This Breaks:** API consumers parse JSON programmatically. Injecting marketing content breaks their schemas and trust.

**The Fix:** Use HTTP response headers (`X-Teaser-Applied: true`) or the existing `wasTruncated` boolean from `applyTeaser()`. Let documentation explain the tier difference.

## Feature Flag Pattern

New capabilities should follow the existing config-driven flag pattern:

```go
// existing pattern — internal/config/config.go
type MLSConfig struct {
    StellarEnabled  bool  `env:"MLS_STELLAR_ENABLED" envDefault:"true"`
    BeachesEnabled  bool  `env:"MLS_BEACHES_ENABLED" envDefault:"true"`
}
```

### Adding a New Flag Checklist

- [ ] Add field to appropriate config struct with `env` tag
- [ ] Default to enabled (`envDefault:"true"`) for zero-disruption deploys
- [ ] Check flag early in handler — return 404 or empty response when disabled
- [ ] Document in `docs/` and `.env.example`
- [ ] No database migration needed for boolean flags

## Anti-Patterns

### WARNING: Feature Gates in Business Logic

**The Problem:**

```go
// BAD — scattering feature checks in service layer
func (s *Service) Search(params) {
    if s.cfg.SomeFlag {
        // different code path
    }
    // ... rest of logic
}
```

**Why This Breaks:** Feature checks belong at the handler/middleware boundary, not scattered through business logic. Mixing access control with business rules makes both hard to test.

**The Fix:** Check at the route or handler level. Return early. Service layer remains flag-agnostic.

```go
// GOOD — gate at handler
func (h *Handler) NewFeature(c *fiber.Ctx) error {
    if !h.cfg.MLS.NewFeatureEnabled {
        return c.Status(http.StatusNotFound).SendString("not found")
    }
    return h.svc.NewFeature(c)
}
```

See the **geospatial** skill for GIS teaser tier implementation details.
See the **auth-api-token** skill for token ability middleware.