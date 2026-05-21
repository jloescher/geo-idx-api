# Distribution Reference

## Contents
- Distribution Channels
- API-as-Distribution Pattern
- Developer Documentation as Channel
- Multi-MLS as Market Expansion
- Anti-Patterns

## Distribution Channels

Quantyra IDX distributes through three channels, all driven by the API itself:

| Channel | Entry point | Conversion path |
|---|---|---|
| Developer API | `/api/v1/*` endpoints | Staging token → domain verify → production token |
| GIS data teaser | `/api/v1/gis` | Free teaser → `idx:full` upgrade |
| Invite network | `/invite/:token` | Admin sends invite → new user onboarded |

There is no public pricing page, no blog, and no self-service signup. Growth is invite-only and API-driven.

## API-as-Distribution Pattern

The API itself is the primary distribution mechanism. Every response is an opportunity to increase usage:

### GIS Teaser Distribution

```go
// Existing — teaser.go returns capped GeoJSON to non-full-access callers
func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
    if fullAccess {
        return geojson, false  // full fidelity
    }
    truncated := truncateFeatureCollection(fc, maxFeatures)  // 40 features
    roundFeatureCollectionCoords(fc, decimals)                // 4 decimals
}
```

Every unauthenticated or limited-scope GIS request becomes a product demonstration. The data is real but deliberately degraded — this is the most effective distribution channel because:

1. Zero friction: no signup required to see data
2. Real value: users see actual parcel geometry, not a mock
3. Clear upgrade path: more features and higher precision behind `idx:full`

**Hypothesis:** Adding a `X-Teaser-Limit: 40` response header increases upgrade intent because developers see the exact constraint.

### Comps API Distribution

```go
// Existing — comps/handler.go exposes BPO, home value, and investor modes
func (h *Handler) Run(c *fiber.Ctx) error {
    return h.svc.Run(c)
}
```

The Comps engine (`POST /api/v1/comps/run`) supports three modes: BPO, home value, and investor. Each mode targets a different buyer persona. See `docs/comps-api.md` for mode details.

**Hypothesis:** Publishing a Comps API quickstart guide on the landing page (below the fold) increases developer signups by demonstrating a concrete use case beyond raw MLS data.

## Developer Documentation as Channel

The `docs/` directory contains 15+ markdown files. These are currently internal-only (no public doc site). Distribution opportunities:

| Doc | Channel potential | Target audience |
|---|---|---|
| `docs/idx-api-bridge-proxy.md` | Public developer guide | IDX site builders |
| `docs/gis-api.md` | GIS integration tutorial | Map application developers |
| `docs/comps-api.md` | BPO use case guide | Real estate investors |
| `docs/listings-mirror.md` | Data architecture post | Engineering decision-makers |

**Hypothesis:** Publishing `docs/gis-api.md` as a public tutorial with embedded API explorer increases GIS teaser requests by 30%.

## Multi-MLS as Market Expansion

The system supports two MLS feeds with room for more:

```go
// Existing — dashboard/handler.go defaults to "stellar" dataset
<label>MLS dataset <input name="mls_dataset" type="text" value="stellar"></label>
```

Each MLS dataset represents a geographic market. Adding a new dataset (e.g., `miami`, `chicago`) opens a new distribution channel without code changes — only configuration and upstream API access.

**Hypothesis:** Each new MLS dataset added to the platform increases total API usage by the average usage of the existing datasets, because IDX customers typically serve single markets.

### Expansion Loop

```
New MLS dataset → Existing customers in that market activate → Their sites drive more API calls → Demonstrates value to prospects in adjacent markets
```

This is a supply-side growth loop: more data → more usage → more data.

## Anti-Patterns

### WARNING: Self-Service Signup Without Domain Verification

**The Problem:**
Opening public registration without DNS verification creates a path to production tokens with no accountability.

**Why This Breaks:**
- MLS data licensing requires domain-level access control
- Unverified domains cannot be audited for compliance
- Production tokens with `idx:full` scope on unverified domains violate MLS terms

**The Fix:**
Keep invite-only onboarding. The friction is intentional and legally required. Growth comes from making the invite flow faster (e.g., pre-approved email domains), not from removing it.

### WARNING: Distribution Without Measurement

**The Problem:**
The `audit_logs` table tracks authenticated requests but there is no tracking for unauthenticated GIS teaser requests.

**Why This Breaks:**
- Cannot measure the GIS teaser conversion funnel
- Cannot calculate which referral sources drive the most teaser → upgrade conversions
- Cannot demonstrate distribution channel ROI

**The Fix:**
Log teaser requests server-side with a `gis.teaser_request` audit event, including the referrer header and whether truncation occurred. See the **cache-postgres** skill for audit log patterns.