# Measurement & Testing Reference

## Contents
- Current Observability
- Plan-Aware Metrics
- A/B Testing Architecture
- Anti-Patterns

## Current Observability

The platform has structured logging (`log/slog`) and health endpoints (`/healthz`, `/readyz`). No analytics, no event tracking, no plan-level metrics.

| Surface | Implementation | Plan visibility |
|---------|---------------|----------------|
| Request logging | `slog` in handlers | No plan/tier in log fields |
| Health checks | `GET /healthz`, `GET /readyz` | N/A |
| Bridge stats | `GET /api/v1/bridge/stats` | Replication state, not usage |
| Queue depth | `jobs` table | Job processing, not per-plan |
| Audit logs | `audit_logs` table | Per-domain, not per-plan |

### WARNING: No Usage Metering

**Detected:** No table or service tracks API calls per token/plan.
**Impact:** Cannot measure conversion funnel (starter → pro), cannot enforce plan limits, cannot identify power users for upsell.

### Recommended: Audit Log Enhancement

The existing `audit_logs` table is the natural foundation for plan-level metrics:

```go
// new code to add — enrich audit logs with plan tier during middleware resolution
func setMLSLocals(c *fiber.Ctx, auth string, d *dom.Domain, tokenName *string, userID *int64, fullAccess bool) {
    c.Locals(ctxkeys.MLSFullAccess, fullAccess)
    // existing code...
    // add plan tier to audit context
    tier := "starter"
    if fullAccess {
        tier = "pro"
    }
    c.Locals(ctxkeys.MLSPlanTier, tier)
}
```

Then aggregate from `audit_logs`:

```sql
-- new code to add — plan conversion funnel
SELECT
  COUNT(CASE WHEN plan_tier = 'starter' THEN 1 END) AS starter_requests,
  COUNT(CASE WHEN plan_tier = 'pro' THEN 1 END) AS pro_requests,
  COUNT(DISTINCT CASE WHEN plan_tier = 'starter' THEN domain_slug END) AS starter_domains,
  COUNT(DISTINCT CASE WHEN plan_tier = 'pro' THEN domain_slug END) AS pro_domains
FROM audit_logs
WHERE created_at > NOW() - INTERVAL '30 days';
```

## Plan-Aware Metrics

### Key Metrics for Offer Ladder

| Metric | Source | Query surface |
|--------|--------|-------------|
| Starter-to-Pro conversion rate | `audit_logs` with plan tier | `COUNT(pro) / COUNT(starter)` per domain |
| GIS teaser hit rate | GIS handler when `truncated=true` | `applyTeaser` return value |
| Per-domain request volume | `audit_logs` grouped by domain | Existing `domain_slug` column |
| Token creation tier | `personal_access_tokens.abilities` | `abilities` JSON field |
| Replication usage | `listings` count per `dataset_slug` | Per-domain dataset access |

### GIS Teaser as Conversion Metric

The existing `applyTeaser` function (`internal/service/gis/teaser.go:24`) returns a `truncated` bool. This is a natural conversion signal:

```go
// new code to add — log teaser hits for funnel analysis
func (h *Handler) Parcel(c *fiber.Ctx) error {
    // ... existing GIS logic ...
    result, truncated := gis.ApplyTeaser(geojson, h.cfg.GIS, fullAccess)
    if truncated {
        h.logger.Info("gis teaser applied",
            slog.String("domain", domainSlug),
            slog.Bool("full_access", fullAccess),
            slog.Int("features_shown", h.cfg.GIS.TeaserMaxFeatures),
        )
    }
    return c.Type("json").Send(result)
}
```

## A/B Testing Architecture

The platform has no A/B framework. For an API product, A/B testing is primarily about:

1. **Teaser threshold testing** — vary `GIS_TEASER_MAX_FEATURES` by domain cohort
2. **Upgrade message testing** — different response headers/body for upgrade CTAs
3. **Pricing page variants** — different copy/layout on `/pricing`

### Minimal A/B via Config

```go
// new code to add — per-domain teaser config for testing
func (h *Handler) getTeaserConfig(domain string) TeaserConfig {
    // control group: 40 features
    // variant A: 20 features (more aggressive teaser)
    // variant B: 60 features (more generous teaser)
    cohort := hashDomainToCohort(domain) // deterministic hash
    return h.cfg.GIS.Experiments[cohort]
}
```

### Anti-Pattern: Client-Side A/B

```go
// BAD — server returns all data, JS hides features
result, _ := json.Marshal(allFeatures) // full data sent
return c.JSON(result) // client truncates
```

**Why This Breaks:** Full data reaches the client regardless. Server-side gating is the only enforceable boundary for an API product.

## Anti-Patterns

### WARNING: Counting Signups Instead of Activation

Measuring conversion at signup misses the real funnel. For an API product:

- **Activation** = first successful API call with real data
- **Engagement** = sustained daily API usage
- **Conversion** = upgrade from `idx:access` to `idx:full`

The audit log is the source of truth for all three.

See the **auth-api-token** skill for audit log structure and token tracking.
See the **geospatial** skill for GIS teaser return values.