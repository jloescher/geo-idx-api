# Strategy and Monetization Reference

## Contents
- Monetization Surfaces
- Scope-Based Access Tiers
- Content Strategy for Tier Upsell
- Anti-Patterns

## Monetization Surfaces

This project monetizes through scope-based API access:

| Surface | Access control | Revenue lever |
|---------|---------------|---------------|
| `/api/v1/*` proxy | Domain + API token | Token scope limits |
| `/api/v1/gis` parcels | `idx:access` scope | Teaser → full geometry |
| `/api/v1/comps/run` | Token scope | BPO, home value, investor modes |
| `/images/*` | Domain auth | Volume-based |
| Dashboard | Invite-only | Customer lifetime value |

## Scope-Based Access Tiers

Token scopes control feature access (see `internal/handler/auth`). Content strategy should align with tier boundaries:

```markdown
Tier 1 (Default scope):
  - Bridge/Spark property proxy
  - Basic search
  - Image proxy

Tier 2 (idx:access scope):
  - Full GIS parcel geometry
  - Comps API (BPO mode)
  - Advanced search filters

Tier 3 (Admin):
  - Dashboard access
  - Token management
  - Audit log visibility
```

## Content Strategy for Tier Upsell

Social content beats should create demand for higher tiers:

1. **Tier 1 content**: "Search 50k+ Active listings by bounding box" — drives signups
2. **Tier 2 teaser**: Show GIS parcel boundary in docs, note "full geometry requires `idx:access` scope"
3. **Tier 2 conversion**: Case study content showing comps API generating BPO reports
4. **Tier 3 prestige**: Dashboard screenshots showing audit logs and multi-domain management

## Anti-Patterns

### WARNING: Free-Tier Content That Cannibalizes Paid Features

**The Problem:** Publishing detailed GIS parcel data in docs that the teaser tier intentionally withholds.

**Why This Breaks:** Removes the upgrade incentive. If developers can see the full schema in docs, they don't need the paid scope.

**The Fix:** Docs for gated features show the request/response shape with placeholder data. Full data examples require an authenticated token with the appropriate scope.

### WARNING: Pricing Content Without Scope Mapping

**The Problem:** Marketing "GIS access" without explaining the `idx:access` scope requirement.

**The Fix:** Every monetization-related content beat must include:
1. Which scope is required
2. How to request scope upgrade (dashboard or admin)
3. What the response looks like at each tier

## Checklist

Copy this checklist and track progress:
- [ ] Tier boundaries documented in content beats
- [ ] Teaser content shows partial data with upgrade CTA
- [ ] No gated data published in full in public docs
- [ ] Scope names (`idx:access`) used consistently in all copy
- [ ] Dashboard reflects current tier messaging