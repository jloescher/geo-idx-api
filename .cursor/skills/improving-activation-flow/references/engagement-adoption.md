# Engagement & Adoption Reference

## Contents
- Measuring engagement
- Feature discovery
- Replication health as engagement signal
- Search adoption patterns
- Anti-patterns

## Measuring Engagement

Engagement for an API product means: are customers making API calls, and are those calls succeeding?

### Key metrics from existing tables

```sql
-- API call volume per domain (last 7 days)
SELECT d.hostname, COUNT(a.id) AS calls
FROM audit_logs a
JOIN domains d ON d.id = a.subject_id
WHERE a.created_at > NOW() - INTERVAL '7 days'
  AND a.action = 'api.request'
GROUP BY d.hostname
ORDER BY calls DESC;
```

### DO: Derive engagement from audit logs

The `audit_logs` table already records authenticated API access. Use it — don't add a separate analytics table.

### DON'T: Add external analytics services without checking

The architecture uses PostgreSQL for all state (no Redis, no external analytics). Adding Mixpanel/PostHog introduces a new dependency and network call in the request path. If you need richer analytics, extend `audit_logs` with a `metadata` JSONB column.

## Feature Discovery

Key features customers should discover after activation:

| Feature | Endpoint | Discovery signal |
|---------|----------|-----------------|
| MLS proxy | `GET /api/v1/properties` | First API call |
| Search | `POST /api/v1/search` | Search request in audit logs |
| GIS parcels | `GET /api/v1/gis` | GIS request in audit logs |
| Comps/BPO | `POST /api/v1/comps/run` | Comps request in audit logs |
| Images | `GET /images/*` | Image proxy hit |

### DO: Surface feature availability in dashboard

```go
// new code to add — return feature availability based on token scope
func FeatureAvailability(scopes []string) map[string]bool {
    return map[string]bool{
        "mls_proxy":     contains(scopes, "idx:access"),
        "search":        contains(scopes, "idx:access"),
        "gis":           contains(scopes, "idx:access"),
        "comps":         contains(scopes, "idx:access"),
        "dashboard":     contains(scopes, "idx:admin"),
    }
}
```

## Replication Health as Engagement Signal

If replication is stale, customers cannot get data — engagement drops to zero.

### DO: Monitor replication freshness

```sql
-- Check if replication is running
SELECT dataset_slug,
       replication_in_progress,
       last_sync_finished_at
FROM listing_sync_cursors;
```

The scheduler runs `mls.replication_kickoff` every minute. If `last_sync_finished_at` is stale beyond `MLS_REPLICATION_FRESHNESS_MINUTES` (default 15), the mirror is lagging. See the **queue-postgresql** skill.

### DON'T: Assume data is always fresh

Always check sync status before investigating "no data" issues. The worker must be running for replication to proceed. The scheduler enqueues; the worker executes.

## Search Adoption Patterns

`POST /api/v1/search` uses a hybrid strategy (PostGIS mirror + live MLS fallback). This is the highest-value endpoint for customers.

### DO: Guide customers toward search after initial proxy use

The proxy endpoint (`GET /api/v1/properties`) is the starting point. Search adds spatial filtering, sorting, and PostGIS performance. The natural adoption path is: proxy → search.

### DON'T: Make search the first activation step

Search requires populated `listings` table (replication complete). New customers should start with the live proxy to see data immediately, then migrate to search as their mirror fills.

## Anti-patterns

### WARNING: Adding engagement tracking in the hot path

```go
// BAD — synchronous analytics HTTP call in request handler
func Handler(c *fiber.Ctx) error {
    http.Post("https://analytics.example.com/track", ...) // blocks response
    return c.JSON(data)
}
```

Engagement tracking must be async (audit log write is fine — same DB, same transaction). Never add external HTTP calls in the API request path. See the **fiber** skill for middleware patterns.