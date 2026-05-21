# Roadmap & Experiments Reference

## Contents
- Configuration-Driven Features
- Search Route Experiments
- Dataset Configuration
- Replication Tuning
- Feature Flags (Env-Based)

---

## Configuration-Driven Features

Most feature behavior is controlled by environment variables rather than feature flags. Changes require redeployment, not runtime toggles.

### Key Config Surfaces

| Config | Env Var | Default | Effect |
|--------|---------|---------|--------|
| `Bridge.SyncFullProperty` | `BRIDGE_SYNC_FULL_PROPERTY` | `true` | Full vs select-mode Bridge sync |
| `Bridge.SyncIncludeMedia` | `BRIDGE_SYNC_INCLUDE_MEDIA` | `true` | Media inline vs separate |
| `MLS.StellarEnabled` | `MLS_STELLAR_ENABLED` | `true` | Enable Bridge/Stellar replication |
| `MLS.BeachesEnabled` | `MLS_BEACHES_ENABLED` | `true` | Enable Spark/Beaches replication |
| `MLS.ReplicationFreshnessMinutes` | `MLS_REPLICATION_FRESHNESS_MINUTES` | `15` | Sync frequency |
| `MLS.LocalMirrorRollingMonths` | `MLS_LOCAL_MIRROR_ROLLING_MONTHS` | varies | Data retention window |

---

## Search Route Experiments

The hybrid search router (`internal/service/search/service.go` → `DecideRoute()`) is the primary experiment surface:

### Route Decision Logic

```go
// Current routing rules (simplified):
func DecideRoute(req SearchRequest) Route {
    if req.PriceReducedWithinDays > 0 { return RouteUpstreamOnly }
    if hasMixedStatuses(req)           { return RouteSplit }
    if onlyActivePending(req)          { return RoutePostgresOnly }
    if onlyClosed(req)                 { return RouteUpstreamOnly }
    return RoutePostgresOnly           // default
}
```

### Experimentation Vectors

| Change | Config Surface | Impact |
|--------|---------------|--------|
| Shift more queries to PostGIS | Expand mirror columns | Reduced upstream cost |
| Add new status to PostGIS route | `DecideRoute()` logic | Faster for those statuses |
| Change default status filter | `SearchRequest` defaults | Changes most users' experience |
| Adjust geo radius limits | PostGIS query bounds | Performance vs coverage |

---

## Dataset Configuration

Each domain has `allowed_mls_datasets` controlling which MLS feeds it can access:

```go
// internal/repository/domain.go
func (r *DomainRepo) AllowedDatasets(d *domain.Domain) []string
```

Dataset routing in requests:

| Parameter | Purpose | Values |
|-----------|---------|--------|
| `?dataset=stellar` | Bridge/Stellar feed | Default for most domains |
| `?dataset=beaches` | Spark/Beaches feed | Requires Spark enablement |

---

## Replication Tuning

Replication behavior is controlled by chunk sizes and batch limits:

| Variable | Bridge Default | Spark Default | Effect |
|----------|---------------|---------------|--------|
| `*_SYNC_REPLICATION_TOP` | 2000 | 1000 | Rows per replication page |
| `*_SYNC_INCREMENTAL_TOP` | 200 | 1000 | Rows per incremental page |
| `*_SYNC_PERSIST_JOB_CHUNK` | 50 | 50 | Rows per persist job |
| `*_SYNC_UPSERT_CHUNK` | 250 | 250 | Rows per SQL upsert |

### Replication Flow for Experiments

```
Scheduler (every minute)
  → mls.replication_kickoff
  → Check freshness: last_sync_finished_at + MLS_REPLICATION_FRESHNESS_MINUTES
  → If stale: enqueue bridge.fetch_page / spark.fetch_page
  → Worker: fetch MLS data → write replica_pages (gzip staging)
  → Worker: bridge.persist_chunk / spark.persist_chunk
  → Upsert to listings table
  → Update listing_sync_cursors
```

---

## Feature Flags (Env-Based)

### WARNING: No Runtime Feature Flags

**The Problem:** All feature toggles are env vars. Changing behavior requires redeploying the affected service (api, worker, or scheduler).

**Why This Breaks:** Can't do gradual rollouts, A/B tests, or instant kill switches. Every config change is a full deploy cycle.

**The Fix:** For critical experiments, consider a database-backed feature flag table that the API checks on each request. Simple approach:

```sql
-- new code to add
CREATE TABLE feature_flags (
    name TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT false,
    allowed_domains TEXT[] -- NULL = all domains
);
```

### Current Toggle Pattern

```go
// Existing pattern: static env-based check
if !cfg.MLS.StellarEnabled {
    // Skip Stellar replication entirely
}
```

This is appropriate for infrastructure-level toggles but not for per-user experiments.

---

## Experiment Checklist

Copy this checklist when planning a new experiment:

- [ ] Identify the config surface (env var vs code change)
- [ ] Determine blast radius (all users vs per-domain)
- [ ] Plan rollback (env var revert vs redeploy)
- [ ] Add audit logging for the new behavior
- [ ] Define success metric (latency, cache hit rate, engagement)
- [ ] Set measurement window (minimum data collection period)
- [ ] Document the experiment in `docs/`

See the **go** skill for Go environment configuration patterns.
See the **deploy-coolify** skill for deployment and env var management.
See the **queue-postgresql** skill for replication queue internals.