# Roadmap & Experiments Reference

## Contents
- Experiment Types for idx-api
- Flag Implementation Tiers
- Rollout Patterns
- Verification Checklist
- Anti-Patterns

---

## Experiment Types for idx-api

| Type | Example | Duration |
|------|---------|----------|
| **Backend behavior** | Different search ranking, cache TTL, persist chunk size | 1–2 weeks |
| **Feature gate** | New comps mode, GIS teaser tier changes | Until stable |
| **Config tuning** | `REPLICATION_FRESHNESS_MINUTES`, `ROLLING_MONTHS` | 1 week |
| **API response shape** | Flat vs nested listing payload, field inclusion | 2–4 weeks |

## Flag Implementation Tiers

### Tier 1: Environment variable (existing pattern)

For global on/off. No migration needed.

```go
// Existing — internal/config/config.go
StellarEnabled: envBool("MLS_STELLAR_ENABLED", true),
```

**When to use:** Features that apply to all domains or all environments uniformly.

### Tier 2: Database-backed per-domain flag

For gradual rollout. Requires migration.

```sql
-- new code to add
CREATE TABLE feature_flags (
    id           SERIAL PRIMARY KEY,
    flag_key     TEXT NOT NULL,
    domain_slug  TEXT,               -- NULL = all domains
    enabled      BOOLEAN DEFAULT FALSE,
    rollout_pct  SMALLINT DEFAULT 100,
    active_from  TIMESTAMPTZ,
    active_until TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT now()
);
CREATE UNIQUE INDEX idx_flags_key_domain ON feature_flags (flag_key, domain_slug);
```

```go
// new code to add — repository method
func (r *FlagRepo) IsEnabled(ctx context.Context, flagKey, domainSlug string) (bool, error) {
    var enabled bool
    err := r.db.QueryRowContext(ctx, `
        SELECT COALESCE(
            (SELECT enabled FROM feature_flags
             WHERE flag_key = $1 AND domain_slug = $2),
            (SELECT enabled FROM feature_flags
             WHERE flag_key = $1 AND domain_slug IS NULL),
            FALSE)
    `, flagKey, domainSlug).Scan(&enabled)
    return enabled, err
}
```

### Tier 3: Deterministic percentage rollout

Same domain always sees the same variant:

```go
// new code to add
import "hash/crc32"

func (r *FlagRepo) IsEnabled(ctx context.Context, flagKey, domainSlug string) (bool, error) {
    var enabled bool
    var rolloutPct int
    err := r.db.QueryRowContext(ctx, `
        SELECT enabled, COALESCE(rollout_pct, 100)
        FROM feature_flags WHERE flag_key = $1 AND (domain_slug = $2 OR domain_slug IS NULL)
        ORDER BY domain_slug DESC NULLS LAST LIMIT 1
    `, flagKey, domainSlug).Scan(&enabled, &rolloutPct)
    if err != nil || !enabled {
        return false, err
    }
    hash := crc32.ChecksumIEEE([]byte(domainSlug)) % 100
    return int(hash) < rolloutPct, nil
}
```

## Rollout Patterns

### Standard rollout sequence

1. Create flag with `enabled = false`, `rollout_pct = 0`
2. Deploy code — feature invisible
3. Set `enabled = true`, `rollout_pct = 10` for canary domains
4. Monitor audit logs for errors/latency
5. Increase to 50%, then 100%
6. Remove flag after full rollout (next migration)

### Kill switch

```sql
-- Instant off for all domains
UPDATE feature_flags SET enabled = false WHERE flag_key = 'new_search_ranking';
```

## Verification Checklist

```
Copy this checklist and track progress:
- [ ] Migration created for feature_flags table (or env var added to config.go)
- [ ] Flag evaluation fails closed (returns false on error)
- [ ] Audit log extended with experiment_id/variant columns
- [ ] Dashboard or admin API allows flag inspection
- [ ] Staging environment tested with flag on and off
- [ ] Rollback plan documented (SQL to disable flag)
- [ ] Post-rollout migration to remove flag scheduled
```

## Anti-Patterns

### WARNING: Non-deterministic rollout

Using `rand.Float32()` for rollout assigns a different variant on every request. The same domain sees both variants, corrupting experiment data.

**Fix:** Always hash a stable identifier (domain_slug, token_name) for deterministic assignment.

### WARNING: Long-lived feature flags

Flags that persist indefinitely become hidden config. Every flag should have a planned removal date.

**Fix:** Set `active_until` and audit quarterly. Remove flags that have been 100% for >30 days.

See the **cache-postgres** skill for caching flag evaluations and the **postgresql** skill for index patterns.