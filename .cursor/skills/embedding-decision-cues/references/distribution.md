# Distribution Reference

## Contents
- Distribution Channels
- API-as-Distribution Pattern
- GIS Teaser as Distribution Mechanism
- Documentation as Distribution

## Distribution Channels

Quantyra IDX is a B2B API platform. Distribution channels are limited to:

| Channel | Mechanism | Entry point |
|---------|-----------|-------------|
| Direct invite | Admin sends `/invite/:token` link | `CreateInvitation()` in dashboard |
| API discovery | Public endpoints (`/healthz`, `/openapi.json`) | Unauthenticated |
| GIS teaser | Truncated parcel data for `idx:access` tokens | `applyTeaser()` |
| Documentation | OpenAPI spec served at `/openapi.json` | Unauthenticated |

There are **no** marketing pages, blog, social integrations, email sequences, or referral programs. Distribution happens through the API itself and word-of-mouth among MLS developers.

## API-as-Distribution Pattern

The API surface IS the marketing surface. Every response carries implicit signals:

### GIS Teaser as Lead Generation

```go
// internal/service/gis/teaser.go
// Users with idx:access (not idx:full) see truncated parcel data
// This creates a "try before you buy" experience via the API itself
func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
    if fullAccess { return geojson, false }
    maxFeatures := cfg.TeaserMaxFeatures  // default 40
    decimals := cfg.TeaserCoordDecimals   // default 4
    ...
}
```

The teaser tier is configurable via `GIS_TEASER_MAX_FEATURES` and `GIS_TEASER_COORD_DECIMALS` — these control the "preview" fidelity that drives upgrade decisions.

### OpenAPI Spec as Onboarding

The API serves its own documentation at `/openapi.json` and `/swagger`. This is the primary discovery mechanism for new developers evaluating the platform.

## GIS Teaser as Distribution Mechanism

The teaser tier implements a classic freemium pattern through data fidelity rather than feature gating:

| Tier | Token scope | GIS data | Behavioral cue |
|------|------------|----------|---------------|
| Staging | `idx:full` | Full parcel data, full coords | Risk-free testing |
| Production (unverified domain) | `idx:access` | 40 features, 4-decimal coords | "Preview" — upgrade cue |
| Production (verified domain) | `idx:full` | Full parcel data, full coords | Full value delivered |

### DO: Communicate the teaser boundary clearly

```go
// new code to add — response header signals the upgrade path
if truncated {
    c.Set("X-GIS-Features-Total", strconv.Itoa(totalFeatures))
    c.Set("X-GIS-Features-Shown", strconv.Itoa(cfg.TeaserMaxFeatures))
}
```

### DON'T: Silently truncate without signaling

Silent truncation makes users think the data is complete, removing the upgrade motivation entirely.

## Documentation as Distribution

API docs live in `docs/` and are served via:
- `GET /openapi.json` — OpenAPI 3.1 spec
- `GET /swagger` — Interactive API explorer

These are the primary touchpoints for developers evaluating the platform. Ensure they reflect current capabilities.

### Distribution Checklist

Copy this checklist and track progress:
- [ ] Verify `/openapi.json` documents all current endpoints
- [ ] Confirm GIS teaser headers communicate preview status
- [ ] Check that `/healthz` and `/readyz` return useful status
- [ ] Audit dashboard invitation flow for new-user experience
- [ ] Verify staging tokens work against the sandbox MLS dataset

See the **proxy-web** skill for MLS proxy endpoint patterns.
See the **geospatial** skill for GIS configuration and tier setup.
See the **auth-api-token** skill for token scope definitions.