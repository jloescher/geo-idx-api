# Conversion Optimization Reference

## Contents
- Activation Funnel
- Landing Page Surface
- Dashboard Conversion Points
- GIS Freemium Gate
- Anti-Patterns

## Activation Funnel

The idx-api conversion funnel has four measurable stages:

```
Landing (/) → Signup (/invite/:token) → Domain Setup (/dashboard) → First API Call (token auth)
```

Each stage maps to a handler in `internal/handler/dashboard/handler.go`:

| Stage | Handler | Measurement point |
|---|---|---|
| Landing view | `marketing.Handler.Home` | Page load (no tracking yet) |
| Invitation accepted | `AcceptInvitation` | User row inserted |
| Domain added | `StoreDomain` | Domain row with `pending` status |
| DNS verified | `VerifyTXT` | Status → `verified`, production token created |
| First API call | Any authenticated endpoint | `audit_logs` row |

**Key insight:** Domain verification is the activation moment. `VerifyTXT` is the only handler that both confirms the domain AND auto-generates a production token in a single response. This is the highest-value conversion point.

## Landing Page Surface

The hero page in `internal/handler/marketing/handler.go` is minimal:

```go
// Existing — marketing/handler.go
body := `<section class="hero">
<h1>Quantyra IDX</h1>
<p>MLS proxy, image delivery, and developer setup for your IDX sites.</p>
<div class="hero-actions">
<a class="btn btn-primary" href="/dashboard">Open dashboard</a>
<a class="btn btn-secondary" href="/login">Sign in</a>
</div>
</section>`
```

**WARNING: No analytics tracking on the landing page**

The hero renders server-side HTML with zero JavaScript tracking. No `pageview`, no `CTA click`, no `referrer` capture. This means:

1. You cannot measure which traffic sources drive dashboard visits
2. You cannot A/B test headline copy without adding instrumentation
3. You cannot calculate landing → signup conversion rates

**Fix:** Add a lightweight tracking pixel or server-side referrer logging before the hero renders.

## Dashboard Conversion Points

`internal/handler/dashboard/handler.go` contains the primary conversion surfaces:

### Domain Verification Flow

```go
// Existing — VerifyTXT auto-creates production token on DNS success
ok, err := dns.VerifyTXT(c.Context(), txtHost, txtVal)
if ok {
    plain, _ := h.tokens.Create(c.Context(), uid, "Production", []string{"idx:full"})
    // Shows token once — highest-value conversion event
}
```

**Optimization opportunity:** The token is displayed once and never again. Add a "copy to clipboard" confirmation (`app.js` already has this) and track whether the user actually copies it.

### Staging Token Friction

```go
// Existing — CreateStagingToken limits to one staging token per user
if exists > 0 {
    return c.Status(409).SendString("Staging token already exists")
}
```

This creates deliberate friction: staging is easy, production requires domain verification. This is intentional gating.

## GIS Freemium Gate

`internal/service/gis/teaser.go` implements the freemium tier:

```go
// Existing — applyTeaser caps features and precision for non-full-access callers
func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
    if fullAccess {
        return geojson, false
    }
    truncated := truncateFeatureCollection(fc, maxFeatures)  // default 40
    roundFeatureCollectionCoords(fc, decimals)                // default 4 decimals
}
```

**Conversion lever:** The `truncated` return value indicates the user hit the limit. This is the ideal trigger for an upgrade prompt or audit event.

**Hypothesis:** If we return a `X-Teaser-Truncated: true` response header when features are capped, client applications can display their own upgrade CTAs without modifying the API contract.

## Anti-Patterns

### WARNING: Measuring Vanity Metrics

**The Problem:**
Tracking "API calls per day" without segmenting by token type (staging vs production, `idx:full` vs `idx:access`) conflates exploration with activation.

**Why This Breaks:**
A user generating 100 staging calls and zero production calls looks active but has not converted.

**The Fix:**
Segment all metrics by `token_name` and `scope` from `personal_access_tokens`. Production token creation is the conversion event; staging activity is pre-conversion.

### WARNING: Adding Client-Side Conversion Tracking Without Server Validation

**The Problem:**
Relying on JavaScript `onclick` events to track conversion assumes the client executes the script.

**Why This Breaks:**
- API consumers (server-to-server) never load JavaScript
- Browser extensions block tracking scripts
- The dashboard uses server-rendered HTML, not a SPA

**The Fix:**
Track conversions server-side in the handler, not in `app.js`. Use `audit_logs` as the source of truth. Client-side tracking is supplementary, never primary.

## Workflow Checklist

Copy this checklist for each conversion optimization experiment:
- [ ] Identify the conversion surface (handler/route)
- [ ] Define the conversion event (row inserted, status changed, token created)
- [ ] Add server-side audit logging before the response
- [ ] Query `audit_logs` for baseline metric
- [ ] Implement the change
- [ ] Run for 7 days minimum
- [ ] Query `audit_logs` for comparison
- [ ] Document result in the experiment log