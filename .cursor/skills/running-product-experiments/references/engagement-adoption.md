# Engagement & Adoption Reference

## Contents
- Engagement Surfaces
- Measuring API Engagement
- Adoption Signals
- Feature Discovery Patterns
- Anti-Patterns

---

## Engagement Surfaces

idx-api has three user-facing surfaces where adoption can be measured:

| Surface | Entry point | Engagement signal |
|---------|------------|-------------------|
| MLS proxy | `GET /api/v1/properties` | Request volume, cache hit rate |
| Search | `POST /api/v1/search` | Query frequency, result counts |
| GIS proxy | `GET /api/v1/gis` | Parcel lookups, teaser vs full |
| Comps | `POST /api/v1/comps/run` | BPO runs, home value checks |
| Dashboard | `GET /dashboard` | Domain/token management |

## Measuring API Engagement

The audit log (`internal/service/audit/logger.go`) is the primary data source:

```go
// Existing audit capture
func (l *Logger) Log(c *fiber.Ctx, requestType string, listingCount *int, cacheHit *string) {
    // Inserts: domain_slug, token_name, request_type, listing_count, ip_address, user_id, cache_hit
}
```

**Engagement queries:**

```sql
-- Weekly active domains (made ≥1 proxy request)
SELECT date_trunc('week', created_at) AS week,
       COUNT(DISTINCT domain_slug) AS active_domains
FROM mls_proxy_audit_logs
GROUP BY 1 ORDER BY 1;

-- Feature adoption: which endpoints each domain uses
SELECT domain_slug,
       COUNT(*) FILTER (WHERE request_type = 'search') AS searches,
       COUNT(*) FILTER (WHERE request_type = 'gis')    AS gis_lookups,
       COUNT(*) FILTER (WHERE request_type = 'comps')  AS comps_runs
FROM mls_proxy_audit_logs
GROUP BY 1;
```

## Adoption Signals

Track adoption of new features by correlating audit data with deploy dates:

1. **New endpoint launch** — Count distinct `domain_slug` values using it per week
2. **Dataset adoption** — `?dataset=beaches` vs `?dataset=stellar` in request logs
3. **Token type split** — Staging vs production token usage from `tokens` table

```sql
-- Staging-to-production token conversion
SELECT d.slug AS domain,
       COUNT(t.id) FILTER (WHERE t.is_staging)  AS staging_tokens,
       COUNT(t.id) FILTER (WHERE NOT t.is_staging) AS prod_tokens
FROM domains d
LEFT JOIN tokens t ON t.domain_id = d.id
GROUP BY 1;
```

## Feature Discovery Patterns

### DO: Surface feature availability in API responses

```go
// new code to add — include available features in dashboard response
type DashboardData struct {
    Domains       []Domain
    Tokens        []Token
    Features      map[string]bool // e.g., {"comps": true, "gis": true}
}
```

### DON'T: Require users to discover features through error messages

Returning 403 with "Feature not available" is not discovery. Show available features proactively in the dashboard or API docs response.

## Anti-Patterns

### WARNING: Using request logs as a real-time analytics pipeline

The `mls_proxy_audit_logs` table is append-only with no retention policy. Without periodic aggregation or a rollup job, queries will slow as rows grow.

**Fix:** Add a scheduler job to aggregate daily metrics into a summary table (see the **queue-postgresql** skill for job patterns).

See the **product-analytics** skill for deeper metrics patterns.