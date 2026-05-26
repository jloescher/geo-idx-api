# Roadmap & Experiments Reference

## Contents
- Feature Rollout Surfaces
- Experimentation via Config
- A/B Testing Patterns for an API
- Measuring Experiment Impact
- Anti-Patterns

## Feature Rollout Surfaces

idx-api has no feature flag system. Feature rollouts are controlled through:

| Surface | Mechanism | File |
|---------|-----------|------|
| MLS feed enable | `SPARK_*` / `BRIDGE_*` env vars present | `internal/config/config.go` |
| Dataset routing | `?dataset=stellar\|beaches` query param | `internal/api/middleware/mls_access.go` |
| Search mode | `MLS_SEARCH_MODE` env var | `internal/service/search/service.go` |
| Sync expand fields | `MLS_SYNC_EXPAND` / `BRIDGE_SYNC_EXPAND` | `internal/config/config.go` |
| Rolling mirror window | `MLS_LOCAL_MIRROR_ROLLING_MONTHS` | `internal/config/config.go` |
| Cache TTL | `MLS_PROXY_CACHE_RETENTION_DAYS` | `internal/config/config.go` |
| Scheduler lock | `SCHEDULER_LEADER_LOCK_ID` | `internal/config/config.go` |

### Pattern: Per-Domain Feature Gate

```sql
-- new code to add — domain-level feature flags
ALTER TABLE domains ADD COLUMN features JSONB DEFAULT '{}';
-- Store: {"comps": true, "gis": true, "search_mode": "hybrid"}
```

This enables gradual rollout per domain without env var changes.

## Experimentation via Config

### Per-Request Experiment Assignment

For an API product, experiments should be deterministic per domain:

```go
// new code to add — deterministic experiment assignment
func experimentVariant(domainSlug string, experiment string) string {
    h := fnv.New32a()
    h.Write([]byte(domainSlug + experiment))
    v := h.Sum32() % 100
    if v < 50 { return "control" }
    return "treatment"
}
```

### DO: Assign by Domain, Not by Request

```go
// GOOD — same domain always sees same variant
variant := experimentVariant(slug, "search_hybrid_mode")
```

### DON'T: Random Assignment per Request

```go
// BAD — inconsistent experience for the same integrator
if rand.Float32() < 0.5 { /* treatment */ }
```

## A/B Testing Patterns for an API

API experiments differ from frontend experiments. The unit of randomization is the **domain** (tenant), not the user session.

### Experiment Metrics from Audit Data

```sql
WITH experiment_groups AS (
    SELECT slug,
           CASE WHEN ABS(MOD((slug::bytea)::bigint, 2)) = 0
                THEN 'control' ELSE 'treatment' END AS variant
    FROM domains WHERE verified = true
)
SELECT eg.variant,
       COUNT(*) AS requests,
       ROUND(COUNT(*) FILTER (WHERE a.cache_hit = 'HIT')::numeric
             / NULLIF(COUNT(*), 0) * 100, 1) AS cache_hit_pct
FROM experiment_groups eg
JOIN mls_proxy_audit_logs a ON a.domain_slug = eg.slug
WHERE a.logged_at > NOW() - INTERVAL '14 days'
GROUP BY eg.variant;
```

### Checklist for Adding a New Experiment

Copy this checklist and track progress:
- [ ] Define hypothesis and success metric
- [ ] Add feature flag to `internal/config/config.go`
- [ ] Implement deterministic variant assignment by domain
- [ ] Add `experiment_name` and `variant` columns to audit log or event
- [ ] Create SQL query to compare metrics across variants
- [ ] Set experiment duration (minimum 2 weeks for weekly seasonality)
- [ ] Document results in commit message or PR description

### Validate: Audit Log Has Experiment Data

1. Deploy experiment
2. Run: `SELECT DISTINCT properties->>'experiment' FROM product_events LIMIT 10;`
3. Verify both `control` and `treatment` variants appear
4. Only proceed with analysis when both variants have data

## Anti-Patterns

### WARNING: Env Vars for Per-Domain Rollout

`MLS_SEARCH_MODE` applies to ALL domains. For gradual rollout, use per-domain config in the `domains` table, not env vars. Env vars are appropriate for infrastructure-level toggles (on/off for the whole service).

### WARNING: No Experiment Metadata in Audit Log

The current `mls_proxy_audit_logs` schema has no `experiment` or `variant` column. Without this, you cannot measure experiment impact from existing data. Add columns before starting experiments.

See the **postgresql** skill for JSONB feature flag storage patterns.
See the **cache-postgres** skill for cache configuration surfaces.
See the **fiber** skill for middleware-based feature gating.