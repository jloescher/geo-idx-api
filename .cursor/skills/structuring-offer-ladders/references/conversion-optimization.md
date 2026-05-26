# Conversion Optimization Reference

## Contents
- Existing Conversion Surfaces
- Upgrade Funnel Architecture
- Teaser-to-Full Conversion Pattern
- Anti-Patterns

## Existing Conversion Surfaces

The platform has two touchpoints where conversion happens today:

1. **Landing page** (`internal/handler/marketing/handler.go`) — hero with "Open dashboard" and "Sign in" CTAs. No pricing, no social proof, no benefit stacking.
2. **Dashboard** (`internal/handler/dashboard/handler.go`) — invite-only domain/token management. No usage metrics, no upgrade prompts.

### WARNING: Missing Conversion Infrastructure

**Detected:** No pricing page, no usage metering, no upgrade CTA, no plan comparison
**Impact:** Users on `idx:access` have no in-app path to discover or purchase `idx:full`. The teaser GIS response silently degrades data without explaining what "full access" unlocks.

### Recommended Fix

Add a conversion layer using existing patterns:

```go
// new code to add — in GIS handler, after applyTeaser returns truncated=true
if truncated {
    c.Set("X-GIS-Teaser", "true")
    c.Set("X-GIS-Max-Features", strconv.Itoa(cfg.TeaserMaxFeatures))
    c.Set("X-GIS-Upgrade-URL", "/dashboard/api-tokens")
}
```

## Upgrade Funnel Architecture

The existing auth flow already resolves tiers. The missing piece is a **conversion bridge** between teaser experience and upgrade action.

```
idx:access user → GIS request → teaser applied → response headers hint at upgrade
     → user checks /dashboard → sees token management → creates idx:full token
```

### Current Flow (Manual)

```go
// internal/api/middleware/domain_token.go:52 — tier is resolved here
fullAccess := tokens.HasAbility(tok, "idx:full")
```

The tier is binary. To optimize conversion, the system needs to:

1. **Surface the gap** — show `idx:access` users what they're missing (response headers, dashboard usage panel)
2. **Reduce friction** — one-click upgrade path instead of manual token recreation
3. **Create urgency** — usage meters showing proximity to limits

## Teaser-to-Full Conversion Pattern

The GIS teaser (`internal/service/gis/teaser.go`) is the only active conversion lever. Two dials control the experience:

| Config | Default | Effect |
|--------|---------|--------|
| `GIS_TEASER_MAX_FEATURES` | 40 | Caps parcel features in response |
| `GIS_TEASER_COORD_DECIMALS` | 4 | Reduces coordinate precision (~11m vs ~1cm) |

**Optimization:** Make the teaser informative enough to demonstrate value but degraded enough to create upgrade motivation. The current 4-decimal precision (~11 meter accuracy) is a reasonable teaser — it shows parcel shape without cadastral precision.

### DO: Tease with Context

```go
// new code to add — response that shows value gap
if truncated {
    // Include metadata showing what full access unlocks
    c.JSON(fiber.Map{
        "type":     "FeatureCollection",
        "features": teasedFeatures,
        "teaser": fiber.Map{
            "shown":    maxFeatures,
            "total":    totalFeatures,
            "upgrade":  "Full access returns all features with centimeter precision.",
        },
    })
}
```

### DON'T: Silent Degradation

```go
// BAD — user doesn't know data is incomplete
fc["features"] = feats[:max]  // just silently truncates, no hint
```

**Why This Breaks:** Users who don't know they're on a limited tier assume the API is unreliable. Silent degradation kills trust and produces support tickets instead of upgrades.

## Anti-Patterns

### WARNING: Client-Side Gating

```go
// BAD — plan check only in the response, not the request
func (h *Handler) Search(c *fiber.Ctx) error {
    results := h.service.Search(c.Context(), params)
    if !fullAccess {
        results = results[:5] // truncate in handler
    }
    return c.JSON(results)
}
```

**Why This Breaks:** Expensive work (full search query, upstream MLS call) runs regardless. Plan gating belongs in middleware or early in the handler — before database queries or upstream API calls.

**The Fix:**

```go
// new code to add — gate before expensive work
func (h *Handler) Search(c *fiber.Ctx) error {
    if !requestFullAccess(c) {
        params.Limit = min(params.Limit, starterMaxResults)
    }
    results := h.service.Search(c.Context(), params) // already bounded
    return c.JSON(results)
}
```

See the **auth-api-token** skill for middleware-based access control patterns.
See the **geospatial** skill for GIS teaser implementation details.