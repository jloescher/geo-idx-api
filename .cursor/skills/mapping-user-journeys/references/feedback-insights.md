# Feedback & Insights Reference

## Contents
- Error Signal Extraction
- Upstream Failure Patterns
- DNS Verification Failures
- Cache Miss Analysis
- Token Rejection Patterns
- Monitoring & Alerting Surfaces

---

## Error Signal Extraction

### From Audit Logs

```sql
-- Request types with highest error rates (proxy failures logged as 502)
SELECT request_type,
       COUNT(*) AS total,
       AVG(listing_count) AS avg_count
FROM mls_proxy_audit_logs
GROUP BY domain_slug, request_type
ORDER BY domain_slug, total DESC;
```

### From Application Logs

The API uses `slog` for structured logging. Key error patterns:

| Log Pattern | Meaning | User Impact |
|-------------|---------|-------------|
| `"proxy upstream error"` | Bridge/Spark HTTP failure | 502 to client |
| `"DNS lookup failed"` | TXT verification DNS error | User can't verify domain |
| `"discarded legacy Laravel queue job"` | Old PHP jobs in queue | No user impact (cleaned up) |
| `"scheduler standby"` | Lost advisory lock | No replication until leader reconnects |

---

## Upstream Failure Patterns

### Bridge/Spark Outage Detection

When Bridge or Spark APIs are down, every `RouteUpstreamOnly` and `RouteSplit` search fails with 502. PostGIS-only searches remain unaffected.

```
Client → POST /api/v1/search (Closed status)
  → RouteUpstreamOnly
  → Bridge/Spark HTTP timeout (30s)
  → fiber.StatusBadGateway
  → Client receives 502
```

### Mitigation Chain

1. Cache serves stale data during outage (15-min TTL)
2. PostGIS mirror serves Active/Pending queries independently
3. No fallback for Closed listings — upstream is the only source
4. No circuit breaker — every request attempts upstream

### WARNING: No Circuit Breaker

**The Problem:** Every request to a downed upstream waits for the full 30-second timeout before returning 502.

**Why This Breaks:** During upstream outages, response times spike to 30s for all upstream-dependent requests, potentially exhausting connection pools.

**When You Might Be Tempted:** Adding a simple failure counter that short-circuits to 503 after N consecutive failures would prevent timeout accumulation.

---

## DNS Verification Failures

The highest-friction onboarding step generates specific failure modes:

| Failure | Cause | Error to User |
|---------|-------|---------------|
| TXT not found | Record not yet propagated | 422 `"TXT record not found..."` |
| DNS lookup failed | Network/DNS server issue | 502 `"DNS lookup failed"` |
| Domain not found | Invalid domain ID | 404 `"domain not found"` |

### Common Support Pattern

1. User adds domain in dashboard
2. User publishes TXT record at DNS host
3. User clicks verify immediately → 422 (propagation delay)
4. User tries again later → success

No guidance in the UI about expected propagation time. No automatic retry polling.

---

## Cache Miss Analysis

High cache miss rates indicate either:
- Legitimate new queries (expanding usage)
- Cache TTL too short for the query pattern
- Cache partitioning too granular (per-domain isolation means no cross-domain sharing)

```sql
-- Domains with lowest cache hit rates
SELECT domain_slug,
       COUNT(*) FILTER (WHERE cache_hit = 'HIT') AS hits,
       COUNT(*) AS total,
       ROUND(100.0 * COUNT(*) FILTER (WHERE cache_hit = 'HIT') / NULLIF(COUNT(*), 0), 1) AS hit_pct
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY domain_slug
HAVING COUNT(*) > 10
ORDER BY hit_pct ASC
LIMIT 20;
```

---

## Token Rejection Patterns

Token auth failures at each middleware stage:

| Stage | Error | Root Cause |
|-------|-------|------------|
| Parse Bearer | `"Unauthenticated."` | Missing Authorization header |
| Hash lookup | `"Invalid API token."` | Wrong or revoked token |
| Ability check | `"Token is missing required IDX abilities."` | Token with wrong scope |
| Domain resolve | `"Missing domain identification..."` | No X-Domain-Slug or Referer |
| Domain ownership | `"Domain is not registered, inactive, or not owned by this token."` | Token + domain mismatch |
| TXT verification | `"Domain must be TXT-verified..."` | DNS verification incomplete |

### WARNING: No Error Code Differentiation

**The Problem:** All token errors return generic 403. Clients can't programmatically distinguish between "bad token" and "unverified domain."

**Why This Breaks:** Integrators can't automate recovery. They can't tell the user "verify your domain" vs "check your API key."

**The Fix:** Add error subcodes or use different HTTP status codes per failure type (401 for auth, 403 for authorization, 409 for pending verification).

---

## Monitoring & Alerting Surfaces

### Health Endpoints

| Endpoint | Check | Alert On |
|----------|-------|----------|
| `GET /healthz` | Process liveness | Timeout → restart |
| `GET /readyz` | Postgres + PostGIS connectivity | Failure → unhealthy |

### Replication Monitoring

```sql
-- Replication lag per dataset
SELECT dataset_slug,
       latest_mod - cursor_last_modification_timestamp AS lag,
       replication_in_progress,
       NOW() - last_sync_finished_at AS time_since_sync
FROM listing_sync_cursors
JOIN (...stats query...);
```

Alert when: `lag > MLS_REPLICATION_FRESHNESS_MINUTES` or `time_since_sync > 2 * freshness`.

### Queue Depth

```sql
-- Stale jobs (stuck in processing)
SELECT queue, type, COUNT(*), MIN(created_at)
FROM jobs
WHERE status IN ('pending', 'processing')
  AND created_at < NOW() - INTERVAL '10 minutes'
GROUP BY queue, type;
```

Alert when: pending jobs exceed threshold or processing jobs older than retry window.

---

## Feedback Loop for Improvements

1. Identify the most common error from audit logs
2. Map the error to a specific middleware/handler stage
3. Check if the error message is actionable for the user
4. Add guidance (UI hint, better error message, or auto-recovery)
5. Deploy and monitor error rate change via audit logs

See the **auth-api-token** skill for token lifecycle and error handling.
See the **queue-postgresql** skill for job queue health monitoring.
See the **deploy-coolify** skill for production monitoring setup.