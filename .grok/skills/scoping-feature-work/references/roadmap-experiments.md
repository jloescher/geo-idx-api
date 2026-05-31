# Roadmap & Experiments Reference

How to scope, flag, and roll out features in Quantyra IDX.

## Contents
- Feature Flagging Patterns
- Experiment Scoping Template
- Rollout Checklist
- Anti-Patterns

## Feature Flagging Patterns

This project uses environment variables as feature flags. There is no feature flag service.

| Flag | Default | Effect |
|------|---------|--------|
| `MLS_STELLAR_ENABLED` | `true` | Enables Bridge/Stellar dataset routing |
| `MLS_BEACHES_ENABLED` | `true` | Enables Spark/Beaches dataset routing |
| `MLS_LOCAL_MIRROR_ROLLING_MONTHS` | `0` (prod) / `3` (staging) | Controls listing retention window |
| `BRIDGE_SYNC_FULL_PROPERTY` | `true` | Controls Bridge fetch mode |
| `BRIDGE_SYNC_NAV_HYDRATE_AFTER_REPLICATION` | `true` | Backfill expanded nav after replication |

### Pattern: Gated feature with env var

```go
// internal/config/config.go — env-based toggle
// new code to add
type Config struct {
    // ...
    Comps CompsConfig
}

type CompsConfig struct {
    Enabled       bool
    MaxRadiusMiles float64
}
```

Toggle in handler:

```go
// new code to add
if !h.cfg.Comps.Enabled {
    return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
```

## Experiment Scoping Template

```
## Experiment: [Name]

**Hypothesis:** [If we change X, then Y metric will improve by Z%]

**Scope:**
- Process: api / worker / scheduler
- Tables: [existing or new migration]
- Config flag: [ENV_VAR_NAME]
- Measurement SQL: [query to evaluate]

**Slices:**
1. MVP: [minimum viable change, behind flag]
2. Full: [complete feature when experiment succeeds]
3. Cleanup: [remove flag, delete old code path]

**Rollback:** [set ENV_VAR_NAME=false, redeploy]
```

### Example: Scope "hybrid search default" experiment

```
## Experiment: Hybrid search as default for /api/v1/search

Hypothesis: Defaulting to PostGIS mirror search (instead of live MLS proxy)
will reduce P95 latency from 800ms to <200ms for 90% of queries.

Scope:
- Process: api only
- Tables: none (uses existing listings + mls_search_cache)
- Config flag: SEARCH_HYBRID_DEFAULT
- Measurement SQL:
  SELECT request_type, AVG(EXTRACT(EPOCH FROM (created_at - LAG(created_at) OVER (...))))

Slices:
1. MVP: Add SEARCH_HYBRID_DEFAULT env var; when true, default search mode='mirror'
   instead of 'live'. Allow override via request body.
2. Full: Remove live-first path after 2 weeks of parity verification.
3. Cleanup: Remove SEARCH_HYBRID_DEFAULT flag.

Rollback: Set SEARCH_HYBRID_DEFAULT=false; redeploy api container.
```

## Rollout Checklist

```
Copy this checklist and track progress:
- [ ] Migration written with down path (`goose down` safe)
- [ ] Feature behind environment variable flag
- [ ] Staging deployed and verified (`APP_ENV=staging`)
- [ ] Audit logging added for new feature surface
- [ ] Stats/metrics endpoint returns new data
- [ ] Production flag set to `false` (off) on deploy
- [ ] Production flag set to `true` after smoke test
- [ ] Flag removed in follow-up if experiment succeeds
```

## Anti-Patterns

### WARNING: Long-lived feature flags

Env var flags are for rollout control, not permanent configuration. If a flag survives two release cycles, either remove the old code path or promote the flag to a documented config option in `internal/config/config.go`.

### WARNING: Experiments without measurement

Every experiment MUST include a SQL query that evaluates the hypothesis. If you cannot write the measurement query from existing audit data, you need to add instrumentation BEFORE the experiment.

## See Also

- See the **deploy-coolify** skill for staged deployment
- See the **queue-postgresql** skill for job queue patterns