# Engagement & Adoption Reference

Measuring and improving how customers use the Quantyra IDX API after activation.

## Contents
- Engagement Signals
- Scoping Engagement Features
- Anti-Patterns

## Engagement Signals

This is an API product — engagement is measured in audit logs and queue throughput, not page views.

| Signal | Source | Query |
|--------|--------|-------|
| API call volume | `mls_proxy_audit_logs` | `COUNT(*) GROUP BY domain_slug` |
| Cache hit rate | `mls_proxy_audit_logs.cache_hit` | `COUNT(CASE WHEN cache_hit='HIT' THEN 1 END) / COUNT(*)` |
| Search usage | Audit `request_type='search'` | Filter by request type |
| Dataset diversity | Audit `domain_slug` + request params | Which MLS datasets get queried |
| Replication health | `GET /api/v1/bridge/stats` | `replication_in_progress`, `last_sync_finished_at` |
| Token churn | `personal_access_tokens` revocations | `WHERE deleted_at IS NOT NULL` |

### Key engagement query

```sql
-- Weekly active domains (made at least one API call)
SELECT domain_slug, COUNT(*) AS requests,
       COUNT(CASE WHEN cache_hit = 'HIT' THEN 1 END) AS cache_hits
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY domain_slug
ORDER BY requests DESC;
```

## Scoping Engagement Features

### Tiered access / teaser mode

The GIS API already uses teaser tiers (`internal/handler/gis/handler.go`). The same pattern can scope new features:

| Tier | Auth requirement | Data depth |
|------|-----------------|------------|
| Public | None | Teaser/limited fields |
| `idx:access` | Domain + token | Full parcel data |
| `idx:full` | Domain + token | All fields + expanded collections |

When scoping a new feature, decide which tier it falls under. Teaser data drives sign-ups; full data drives token usage.

### Scoping template for engagement features

```
Feature: [name]
Current signal: [what audit/logs tell us today]
Hypothesis: [adding X will increase Y signal]
Measurement: [SQL query or endpoint to verify]
Rollback: [feature flag or config toggle to disable]
```

### Example: Scope "saved searches" feature

```
Feature: Saved search alerts (email/webhook on new listings matching criteria)
Current signal: POST /api/v1/search volume in audit logs
Hypothesis: Alerts will increase daily active API calls by 20%
Measurement: New request_type='alert_trigger' in audit logs + daily search count
Rollback: MLS_ALERTS_ENABLED env var, no migration changes
Process: scheduler (cron) + worker (new job type) + api (CRUD endpoints)
```

## Anti-Patterns

### WARNING: Measuring engagement with dashboard page views

This is an API product. Dashboard visits (`GET /dashboard`) do not indicate product engagement. Measure API call volume, search usage, and replication health — not HTML page loads.

### WARNING: Adding engagement features that require new infrastructure

The system uses PostgreSQL for everything (queue, cache, sessions). Do not introduce Redis, Kafka, or a message broker for engagement features. Use the existing PostgreSQL queue (`internal/queue`) for any background alert/notification work.

## See Also

- See the **cache-postgres** skill for cache hit rate patterns
- See the **queue-postgresql** skill for background job patterns
- See the **geospatial** skill for teaser tier implementation