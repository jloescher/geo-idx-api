# Measurement & Testing Reference

## Contents
- Data Sources
- Audit Log Schema
- Experiment Framework
- Hypothesis Validation Workflow
- Anti-Patterns

## Data Sources

The idx-api has one primary measurement source and several secondary signals:

| Source | Table | Granularity | Best for |
|---|---|---|---|
| Audit logs | `audit_logs` | Per authenticated request | API usage, token activity, endpoint popularity |
| Job queue | `jobs` | Per background job | Replication throughput, worker health |
| Domain status | `domains` | Per domain | Verification funnel, activation rate |
| Token registry | `personal_access_tokens` | Per token | Token creation/revoke lifecycle |
| GIS teaser config | `config.GISConfig` | Per request | Freemium tier hit rate |

**WARNING: No analytics SDK or external tracking**

The codebase has no Google Analytics, Segment, PostHog, or similar integration. All measurement must come from PostgreSQL queries against the tables above.

## Audit Log Schema

All authenticated API requests are logged to `audit_logs`. This is the primary measurement surface for growth experiments:

```sql
-- Baseline: API calls per token type, last 30 days
SELECT t.name AS token_name,
       COUNT(*) AS requests,
       COUNT(DISTINCT a.user_id) AS unique_users
FROM audit_logs a
JOIN personal_access_tokens t ON t.tokenable_id = a.user_id
WHERE a.created_at > NOW() - INTERVAL '30 days'
GROUP BY t.name
ORDER BY requests DESC;
```

```sql
-- Activation funnel: from user creation to first production API call
SELECT COUNT(*) AS total_users,
       COUNT(d.id) AS with_domain,
       COUNT(d.txt_verified_at) AS verified,
       COUNT(p.id) AS with_production_token
FROM users u
LEFT JOIN domains d ON d.user_id = u.id
LEFT JOIN personal_access_tokens p ON p.tokenable_id = u.id AND p.name = 'Production';
```

## Experiment Framework

There is no A/B testing framework. Experiments are implemented as query-parameter or cookie-based variants in handlers.

### Pattern: Handler-Level Variant

```go
// new code to add — variant selection in a handler
func (h *Handler) SomeAction(c *fiber.Ctx) error {
    variant := c.Query("v", "control")
    switch variant {
    case "treatment_a":
        // treatment logic
    default:
        // control logic
    }
    // Always log the variant for analysis
    h.audit.Log(c.Context(), uid, "experiment.impression", map[string]any{
        "experiment": "landing_hero_copy",
        "variant":    variant,
    })
}
```

### Pattern: Configuration-Driven Experiment

```go
// new code to add — use config for experiment parameters
type GrowthConfig struct {
    TeaserMaxFeatures int `env:"GIS_TEASER_MAX_FEATURES" envDefault:"40"`
    TeaserCoordDecimals int `env:"GIS_TEASER_COORD_DECIMALS" envDefault:"4"`
}
```

Changing `GIS_TEASER_MAX_FEATURES` from 40 to 20 is a valid experiment. Measure the impact via audit logs of teaser requests that return `truncated=true`.

## Hypothesis Validation Workflow

1. Write hypothesis in this format:
   ```
   If we [change], then [metric] will [direction] because [reason].
   ```
2. Query baseline from `audit_logs` or `domains`
3. Implement the change with a variant parameter
4. Deploy to staging first (`APP_ENV=staging`)
5. Run for minimum 7 calendar days
6. Query the same metric, segmented by variant
7. Document result: `docs/experiments/YYYY-MM-DD-experiment-name.md`

### Minimum Viable Experiment

Not every hypothesis needs code. Some can be validated with SQL:

```sql
-- Hypothesis: Users who add a domain within 24h of signup have higher API usage
SELECT
    CASE WHEN d.created_at < u.created_at + INTERVAL '24 hours' THEN 'fast' ELSE 'slow' END AS cohort,
    COUNT(DISTINCT a.id) AS avg_requests
FROM users u
LEFT JOIN domains d ON d.user_id = u.id
LEFT JOIN audit_logs a ON a.user_id = u.id
WHERE u.created_at > NOW() - INTERVAL '90 days'
GROUP BY cohort;
```

If the `fast` cohort has significantly higher usage, prioritize reducing domain-setup friction.

## Anti-Patterns

### WARNING: Statistical Significance Ignored

**The Problem:**
Declaring a winner based on 2 days of data with <50 conversions per variant.

**Why This Breaks:**
Day-of-week effects and small sample sizes produce misleading results. A Monday winner may be a Tuesday loser.

**The Fix:**
Run experiments for at least 7 full calendar days. Use a chi-squared test or simple proportion comparison before declaring significance. For low-traffic surfaces (invite flow), run longer — 14-30 days.

### WARNING: Measuring Only the Top of the Funnel

**The Problem:**
Optimizing landing page click-through without tracking the full funnel to production token creation.

**Why This Breaks:**
A headline that gets more clicks but fewer verified domains is a net negative. The conversion event for this B2B product is **domain verification + production token creation**, not page views.

**The Fix:**
Always measure the full funnel: landing view → invitation accepted → domain added → domain verified → production token created → first API call. The `domains` table with `verification_status` and `txt_verified_at` columns provides this.