# Strategy & Monetization Reference

## Contents
- Business Model
- Pricing Surfaces
- Tier Architecture
- Revenue Expansion Levers
- Anti-Patterns

## Business Model

Quantyra IDX operates as a **B2B API platform** with invite-only access. Revenue is generated through API access to MLS data, GIS parcel geometry, and Comps/BPO analytics. The platform sits between MLS data providers and IDX website builders.

```
MLS Providers (Bridge/Spark) → Quantyra IDX API → Customer IDX Sites
                                     ↑
                              Domain-verified access
                              Tiered GIS data
                              Comps/BPO engine
```

The invite-only model (`/invite/:token`) means customer acquisition is relationship-driven, not self-service. This is intentional for MLS compliance.

## Pricing Surfaces

There is no public pricing page or billing integration in the codebase. The monetization surfaces that exist:

| Surface | Code location | Monetization potential |
|---|---|---|
| GIS teaser tier | `internal/service/gis/teaser.go` | Free tier with paid upgrade |
| Token scopes | `personal_access_tokens` with `idx:full` / `idx:access` | Scope-based pricing |
| Domain verification | `domains` table with `verification_status` | Per-domain pricing |
| Staging vs production | `CreateStagingToken` / `CreateToken` | Environment-based pricing |
| Comps modes | `POST /api/v1/comps/run` (BPO, home value, investor) | Per-mode pricing |

## Tier Architecture

The codebase implements a two-tier access model:

### Tier 1: Full Access (`idx:full`)

```go
// Existing — tokens created with full scope
plain, _ := h.tokens.Create(c.Context(), uid, "Production", []string{"idx:full"})
```

Full access returns unmodified GIS GeoJSON with all features and full coordinate precision.

### Tier 2: Teaser Access (no `idx:full` scope)

```go
// Existing — teaser.go applies degradation
func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
    if fullAccess {
        return geojson, false
    }
    truncated := truncateFeatureCollection(fc, maxFeatures)
    roundFeatureCollectionCoords(fc, decimals)
}
```

Teaser access is the natural free tier. The degradation is configurable without code changes:

| Config | Default | Monetization lever |
|---|---|---|
| `TeaserMaxFeatures` | 40 | Reduce to increase upgrade pressure |
| `TeaserCoordDecimals` | 4 | Reduce to increase upgrade pressure |

**Hypothesis:** Reducing `TeaserMaxFeatures` from 40 to 10 increases paid tier conversion by 20% without reducing teaser signups, because 10 features still demonstrates value for most parcels.

### Potential Third Tier: Per-MLS Pricing

```go
// Existing — domain creation stores MLS dataset
<label>MLS dataset <input name="mls_dataset" type="text" value="stellar"></label>
```

Each domain is tied to a single MLS dataset. A natural pricing tier:

| Tier | Included | Price signal |
|---|---|---|
| Starter | 1 MLS dataset, teaser GIS | Free or low cost |
| Professional | 1 MLS dataset, full GIS + Comps | Mid-tier |
| Enterprise | Multiple MLS datasets, full GIS + Comps + team invites | Premium |

## Revenue Expansion Levers

### Lever 1: GIS Precision Upgrade

The teaser rounds coordinates to 4 decimal places (~11 meter precision). Cadastral accuracy requires full precision. This is a natural upsell:

```go
// Existing — precision degradation
roundFeatureCollectionCoords(fc, decimals)  // 4 decimals for teaser
```

Real estate professionals need sub-meter precision for boundary disputes, setback calculations, and parcel identification. The upgrade from 11m to <1m precision is a clear value-add.

### Lever 2: Comps API Monetization

The Comps engine supports three modes (see `docs/comps-api.md`):

- **BPO mode**: Broker Price Opinions — per-report pricing
- **Home value mode**: Automated valuations — usage-based pricing
- **Investor mode**: Portfolio analysis — subscription pricing

Each mode has different unit economics. BPO is high-value per call; home value is high-volume.

### Lever 3: Multi-MLS Expansion

```go
// Existing — domain creation supports any dataset slug
_, err := h.db.Pool.Exec(c.Context(), `
    INSERT INTO domains (..., mls_dataset, allowed_mls_datasets, ...)
    VALUES ($1, $2, $3, $4::jsonb, ...)
`, uid, slug, mls, `["`+mls+`"]`)
```

Adding a new MLS market (e.g., Miami, Chicago) creates immediate inventory for existing customers who operate in those markets. The `allowed_mls_datasets` JSONB column supports multi-dataset access per domain.

**Hypothesis:** Existing customers who add a second MLS dataset have 3x higher retention because switching costs increase with data integration depth.

### Lever 4: Team Seats

The invite system supports team growth:

```go
// Existing — admin-only invitation creation
app.Post("/dashboard/invitations", h.requireAuth, h.requireAdmin, h.CreateInvitation)
```

Currently, every invited user gets the same access. Tiered team seats (admin, editor, viewer) create per-seat revenue potential.

## Anti-Patterns

### WARNING: Pricing Without Usage Tracking

**The Problem:**
Introducing pricing tiers without first measuring per-customer API usage patterns.

**Why This Breaks:**
- Cannot set tier boundaries that maximize revenue
- Cannot identify which customers would upgrade
- Cannot predict revenue impact of tier changes

**The Fix:**
Query `audit_logs` for per-customer usage patterns before setting prices:

```sql
SELECT t.tokenable_id,
       t.name AS token_type,
       COUNT(a.id) AS monthly_requests,
       COUNT(DISTINCT DATE(a.created_at)) AS active_days
FROM audit_logs a
JOIN personal_access_tokens t ON t.tokenable_id = a.user_id
WHERE a.created_at > date_trunc('month', NOW()) - INTERVAL '1 month'
GROUP BY t.tokenable_id, t.name
ORDER BY monthly_requests DESC;
```

Use the distribution to set tier boundaries at natural breakpoints.

### WARNING: Free Tier Without Rate Limiting

**The Problem:**
The GIS teaser has feature count and precision caps but no request rate limit.

**Why This Breaks:**
An unauthenticated user can make unlimited teaser requests, consuming database and upstream resources without contributing revenue.

**The Fix:**
Add rate limiting at the Fiber middleware layer for unauthenticated GIS requests. See the **fiber** skill for middleware patterns. The limit can be generous (100 requests/hour) — the goal is abuse prevention, not revenue extraction.

### WARNING: Monetization Without MLS Compliance

**The Problem:**
Charging for MLS data access without verifying that the pricing structure complies with MLS licensing terms.

**Why This Breaks:**
MLS agreements often restrict how data can be resold, who can access it, and how it can be displayed. Non-compliant monetization can result in feed termination.

**The Fix:**
Review Bridge and Spark data licensing agreements before implementing any pricing tier. The `verification_status` on `domains` and the `allowed_mls_datasets` column exist specifically for compliance enforcement.