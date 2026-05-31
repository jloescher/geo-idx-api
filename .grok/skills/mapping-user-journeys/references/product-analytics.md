# Product Analytics Reference

## Contents
- Audit Trail
- Replication Stats
- Search Analytics
- Cache Performance
- Token Usage Tracking
- Analytics Gaps

---

## Audit Trail

Every proxied API request is logged via `internal/service/audit/logger.go`:

```sql
INSERT INTO mls_proxy_audit_logs
  (domain_slug, token_name, request_type, listing_count, ip_address, user_id, cache_hit)
VALUES ($1, $2, $3, $4, $5, $6, $7)
```

| Field | Source | Use |
|-------|--------|-----|
| `domain_slug` | Authenticated domain | Per-domain activity |
| `token_name` | Bearer token name | Per-token usage |
| `request_type` | Handler label (e.g., `"properties"`, `"search"`) | Feature usage |
| `listing_count` | Result count | Volume analysis |
| `ip_address` | `c.IP()` | Geographic distribution |
| `user_id` | Authenticated user | Per-user analytics |
| `cache_hit` | `"HIT"` / `"MISS"` | Cache efficiency |

### Coverage

| Endpoint | Audited | Request Type Label |
|----------|---------|--------------------|
| `GET /api/v1/properties` | Yes | Via proxy audit |
| `POST /api/v1/search` | Yes | `"search"` |
| `GET /api/v1/gis` | Yes | Via proxy audit |
| `POST /api/v1/comps/run` | Partial | Delegated to comps service |
| `GET /images/*` | Varies | Domain auth dependent |

---

## Replication Stats

`GET /api/v1/bridge/stats` (`internal/service/sync/stats.go`) returns per-dataset health:

```json
{
  "dataset_slug": "stellar",
  "active_pending": 42500,
  "latest_modification": "2025-05-20T12:00:00Z",
  "cursor_last_modification_timestamp": "2025-05-20T11:45:00Z",
  "last_sync_finished_at": "2025-05-20T11:46:00Z",
  "incremental_window_end": null,
  "replication_in_progress": false
}
```

Key metrics:

| Metric | SQL Source | Meaning |
|--------|-----------|---------|
| `active_pending` | `COUNT(*) FILTER (WHERE status IN ('active','pending'))` | Mirror size |
| `latest_modification` | `MAX(modification_timestamp)` from `listings` | Data freshness |
| `cursor_last_modification_timestamp` | `listing_sync_cursors` | Sync cursor position |
| `last_sync_finished_at` | `listing_sync_cursors` | Last successful sync |
| `replication_in_progress` | `listing_sync_cursors` | Active sync flag |

### Replication Lag Calculation

```sql
-- Lag = latest_modification - cursor_last_modification_timestamp
-- Fresh if within MLS.ReplicationFreshnessMinutes (default 15 min)
```

---

## Search Analytics

Search results include metadata for funnel analysis:

- `total` — total matching results (for pagination depth tracking)
- `has_more` — indicates further pages exist
- `next_skip` — pagination cursor

Track search patterns via audit logs:

```sql
-- Most popular search filters (by request volume)
SELECT request_type, COUNT(*), AVG(listing_count)
FROM mls_proxy_audit_logs
WHERE request_type = 'search'
GROUP BY 1 ORDER BY 2 DESC;
```

Search routing distribution (not directly tracked — infer from response latency):
- `RoutePostgresOnly` → fast mirror hits
- `RouteUpstreamOnly` → upstream dependency
- `RouteSplit` → hybrid

---

## Cache Performance

Proxy cache stats are derivable from audit `cache_hit` column:

```sql
-- Cache hit rate per domain
SELECT domain_slug,
       COUNT(*) AS total,
       COUNT(*) FILTER (WHERE cache_hit = 'HIT') AS hits,
       ROUND(100.0 * COUNT(*) FILTER (WHERE cache_hit = 'HIT') / COUNT(*), 1) AS hit_rate
FROM mls_proxy_audit_logs
GROUP BY domain_slug
ORDER BY hit_rate ASC;
```

Cache TTL affects analytics:
- `Bridge.ListingsCacheTTL` = 900s — frequent refresh
- `Bridge.LookupCacheTTL` = 720h — near-static

---

## Token Usage Tracking

`personal_access_tokens.last_used_at` updated on every API call:

```sql
-- Dormant tokens (unused in 30+ days)
SELECT name, tokenable_id, last_used_at
FROM personal_access_tokens
WHERE last_used_at < NOW() - INTERVAL '30 days'
ORDER BY last_used_at ASC;
```

---

## Analytics Gaps

### WARNING: No Search Filter Analytics

**The Problem:** `SearchRequest` fields are parsed and used for querying but not logged. No visibility into which filters users actually use.

**Why This Breaks:** Can't determine if geographic search, price filters, or property types drive engagement. Can't optimize the most-used query paths.

**The Fix:** Log a sanitized summary of search filters to audit or a dedicated analytics table.

### WARNING: No Funnel Tracking

**The Problem:** No event tracking for onboarding steps (domain added → verified → token created → first call). Each step is independent with no linking event chain.

**Why This Breaks:** Can't measure onboarding completion rate or identify where users drop off.

**The Fix:** Add an onboarding events table or tag audit logs with journey step labels.

### Missing Surfaces

| Gap | Impact |
|-----|--------|
| No search filter logging | Can't optimize popular queries |
| No onboarding funnel | Can't measure activation rate |
| No error rate tracking by endpoint | Can't identify failing journeys |
| No latency percentiles | Can't detect degraded experience |
| No geographic usage data | Can't optimize multi-DC routing |

See the **cache-postgres** skill for cache layer details.
See the **queue-postgresql** skill for job queue observability.