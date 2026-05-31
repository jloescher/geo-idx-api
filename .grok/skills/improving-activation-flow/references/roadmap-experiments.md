# Roadmap & Experiments Reference

## Contents
- Experiment constraints in this architecture
- Safe rollout patterns
- Feature flags with PostgreSQL
- A/B test considerations
- Verification checklist

## Experiment Constraints

This is a Go API backend with PostgreSQL state. There is no frontend A/B testing framework. Experiments must be implemented as backend logic with database-backed configuration.

### DO: Use database-backed feature flags

```sql
-- new code to add — feature flag table
CREATE TABLE IF NOT EXISTS feature_flags (
    key TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT false,
    domains TEXT[] DEFAULT '{}',  -- empty = all domains, or list of domain IDs
    config JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

```go
// new code to add — feature flag check
func (r *Repository) IsFeatureEnabled(ctx context.Context, key, domainID string) (bool, error) {
    var enabled bool
    err := r.db.QueryRowContext(ctx,
        `SELECT enabled AND (domains = '{}' OR $2 = ANY(domains))
         FROM feature_flags WHERE key = $1`,
        key, domainID,
    ).Scan(&enabled)
    return enabled, err
}
```

### DON'T: Use environment variables for per-domain experiments

```go
// BAD — requires redeployment to change, applies to all domains
os.Getenv("ENABLE_NEW_SEARCH")

// GOOD — database flag, changeable at runtime, scoped per domain
repo.IsFeatureEnabled(ctx, "new_search_algorithm", domainID)
```

## Safe Rollout Patterns

### Percentage rollout

```go
// new code to add — deterministic rollout by domain ID hash
func (r *Repository) IsRolloutEnabled(ctx context.Context, feature, domainID string, pct int) bool {
    if pct >= 100 {
        return true
    }
    h := fnv.New32a()
    h.Write([]byte(domainID))
    bucket := h.Sum32() % 100
    return int(bucket) < pct
}
```

### Domain allowlist rollout

```sql
-- Enable feature for specific domains first
INSERT INTO feature_flags (key, enabled, domains)
VALUES ('hybrid_search_v2', true, ARRAY['domain-id-1', 'domain-id-2']);
```

### Gradual expansion

```
1. Enable for 2 domains (internal testing)
2. Verify: run integration tests, check audit logs for errors
3. Enable for 10 domains (beta)
4. Verify: compare response times and error rates
5. Enable for 25% (hash-based rollout)
6. Verify: `SELECT COUNT(*) FROM audit_logs WHERE action = 'search.request' AND metadata->>'source' = 'v2'`
7. Enable for 100%
```

## A/B Test Considerations

### DO: Store experiment assignment in metadata

```go
// new code to add — include experiment variant in audit log metadata
metadata := map[string]interface{}{
    "experiment": "search_algorithm",
    "variant":    "v2_spatial",
    "latency_ms": elapsed.Milliseconds(),
}
auditRepo.Record(ctx, "search.request", domainID, metadata)
```

### DON'T: Run A/B tests that change response schema

Clients parse API responses. Returning different JSON shapes per variant breaks integrations. If testing a new search algorithm, return the same response shape — only change the ranking/filtering logic internally.

## Verification Checklist

```
Copy this checklist and track progress:
- [ ] Feature flag table exists and has the experiment key
- [ ] Flag check is performed before the experimental code path
- [ ] Audit logs record the variant assigned to each request
- [ ] Rollback is possible by setting `enabled = false` (no redeployment)
- [ ] Monitoring query exists to compare variants:
      SELECT metadata->>'variant', COUNT(*), AVG((metadata->>'latency_ms')::float)
      FROM audit_logs WHERE action = 'search.request'
      AND created_at > NOW() - INTERVAL '1 day'
      GROUP BY 1;
```

See the **cache-postgres** skill for caching experiment configs and the **queue-postgresql** skill for async experiment processing.