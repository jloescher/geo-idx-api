# Conversion Optimization Reference

## Contents
- Dashboard-to-API-Key Funnel
- Docs-as-Conversion-Surface
- Teaser-to-Full-Access Upgrades
- Anti-Patterns

## Dashboard-to-API-Key Funnel

The primary conversion path: visitor → dashboard signup → API key generation → first API call.

```go
// The auth flow in internal/handler/auth handles domain + token creation
// Conversion friction points: ADMIN_SEED_EMAIL/PASSWORD for initial setup,
// then /dashboard for customer self-service key issuance
```

Conversion optimization targets:
1. **Landing → Dashboard**: Reduce steps between reading docs and creating an API key
2. **Dashboard → First call**: Provide copy-paste example with the new token
3. **First call → Repeat use**: Dataset routing (`?dataset=stellar|beaches`) should work immediately

## Docs-as-Conversion-Surface

`docs/` is the primary top-of-funnel surface for this developer platform. Each doc is a conversion opportunity.

**DO:** Include working curl examples with placeholder tokens in every endpoint doc.
```markdown
GET /api/v1/search?dataset=stellar
Authorization: Bearer YOUR_TOKEN
```

**DON'T:** Link to external auth docs without showing the request inline. Developers drop off at context switches.

## Teaser-to-Full-Access Upgrades

GIS parcels use a teaser tier system (see `docs/gis-api.md`). This is a built-in conversion mechanism:

- Unauthenticated: limited parcel metadata
- `idx:access` scope PAT: full geometry + parcel data
- Conversion trigger: "Want full parcel data? Upgrade your token scope."

Social content beats should align with teaser expansion — announce new data sources when the GIS probe (`gis.probe_sources`) discovers them.

## Anti-Patterns

### WARNING: Announcing Features Before Docs Exist

**The Problem:** Social posts linking to 404s or incomplete docs erode trust immediately.

**Why This Breaks:** Developer audience expects working references. A dead link on launch day means the developer moves on and rarely returns.

**The Fix:** Content beats must list "docs published" as a prerequisite before the announcement beat fires. Use the editorial arc pattern from SKILL.md — docs ship in Week 1, social in Week 2.

### WARNING: Generic CTAs on Developer Content

**The Problem:** "Sign up today!" does not convert developers.

**The Fix:** Use specific, value-driven CTAs: "Search 50k+ Active listings with one POST request — get your API key at /dashboard"

## Checklist

Copy this checklist and track progress:
- [ ] Docs published and verified for new feature
- [ ] curl examples tested with fresh token
- [ ] Teaser tier updated if feature is gated
- [ ] Dashboard banner reflects current beat
- [ ] Analytics event fires on conversion action