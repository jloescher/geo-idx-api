# Content & Copy Reference

## Contents
- Current Copy Surfaces
- Copy Architecture
- Offer Ladder Messaging
- Anti-Patterns

## Current Copy Surfaces

All user-facing copy lives in Go string literals within handlers. No CMS, no MDX, no template engine.

| Surface | File | Current copy |
|---------|------|-------------|
| Landing hero | `internal/handler/marketing/handler.go:19` | "MLS proxy, image delivery, and developer setup for your IDX sites." |
| Login form | `internal/handler/dashboard/handler.go:83` | "Email" / "Password" / "Sign in" |
| API errors | `internal/api/middleware/domain_token.go` | "Invalid API token.", "Missing domain identification." |
| GIS teaser | `internal/service/gis/teaser.go` | Silent truncation — no user-facing copy |

### WARNING: No Copy Layer

All copy is hardcoded in Go source. Changing a headline requires a code deploy. For an API product this is acceptable; for a marketing site with pricing experiments it is not.

**Decision point:** If pricing pages will A/B test, extract copy to a separate layer before building the page. If pricing is stable/sales-led, inline copy in handlers follows the existing pattern.

## Copy Architecture

Follow the project convention: server-side HTML in handler methods, using `web.Page()` layout wrapper.

```go
// Existing pattern — internal/handler/marketing/handler.go:18
body := `<section class="hero">
<h1>Quantyra IDX</h1>
<p>MLS proxy, image delivery, and developer setup for your IDX sites.</p>
<div class="hero-actions">
<a class="btn btn-primary" href="/dashboard">Open dashboard</a>
<a class="btn btn-secondary" href="/login">Sign in</a>
</div>
</section>`
return c.Type("html").SendString(web.Page("Home", body))
```

## Offer Ladder Messaging

### Tier Naming Conventions

The codebase uses `idx:access` and `idx:full` as ability strings. User-facing tier names should be distinct:

| Internal ability | User-facing name | Positioning |
|-----------------|-----------------|-------------|
| `idx:access` | Starter | MLS search + teaser GIS |
| `idx:full` | Pro | Full GIS, comps, all datasets |
| (none) | Enterprise | Custom limits, multi-DC, SLA |

### Value Proposition Per Tier

**Starter (`idx:access`):**
- MLS proxy (Bridge + Spark)
- PostGIS mirror search
- Image delivery
- GIS teaser (40 features, ~11m precision)

**Pro (`idx:full`):**
- Everything in Starter
- Full GIS parcel data (unlimited features, centimeter precision)
- Comps/BPO engine
- All MLS datasets without restriction

**Enterprise (future):**
- Everything in Pro
- Custom replication intervals
- Multi-DC dedicated workers
- Priority queue routing
- SLA guarantees

### Writing Pricing Page Copy

Follow the existing dark theme and card-based layout from `app.css`:

```go
// new code to add — pricing section following existing CSS patterns
body := `<section class="pricing">
<h1>Simple, transparent pricing</h1>
<p class="pricing-subtitle">MLS data infrastructure for your IDX products</p>
<div class="pricing-grid">
  <div class="pricing-card">
    <h2>Starter</h2>
    <p class="price">Free</p>
    <ul><li>MLS search proxy</li><li>Image delivery</li><li>GIS teaser</li></ul>
    <a class="btn btn-secondary" href="/dashboard">Get started</a>
  </div>
  <div class="pricing-card pricing-card--featured">
    <h2>Pro</h2>
    <p class="price">Contact sales</p>
    <ul><li>Full GIS parcels</li><li>Comps & BPO</li><li>All MLS datasets</li></ul>
    <a class="btn btn-primary" href="/dashboard/api-tokens">Upgrade</a>
  </div>
</div>
</section>`
```

## Anti-Patterns

### WARNING: Feature Lists Instead of Outcomes

```html
<!-- BAD — lists what the API does, not what the user gets -->
<li>PostGIS spatial queries</li>
<li>RESO OData proxy</li>
<li>PostgreSQL advisory locks</li>
```

**Why This Breaks:** Technical features are implementation details. Buyers care about outcomes: "Search every active listing in South Florida in under 200ms."

```html
<!-- GOOD — outcome-oriented copy -->
<li>Real-time MLS search across 50,000+ active listings</li>
<li>Sub-200ms spatial queries with local mirror</li>
```

See the **frontend-design** skill for CSS layout and card styling patterns.
See the **ux** skill for dashboard upgrade flow interaction design.