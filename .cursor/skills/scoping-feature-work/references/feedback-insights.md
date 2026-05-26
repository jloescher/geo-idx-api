# Feedback & Insights Reference

How to gather, track, and act on user feedback for Quantyra IDX.

## Contents
- Feedback Channels
- Scoping Feedback-Driven Features
- Triage Framework
- Anti-Patterns

## Feedback Channels

This is an API product. Feedback does not come through in-app forms — it comes through:

| Channel | Signal | How to detect |
|---------|--------|---------------|
| Audit logs | Error patterns | `request_type` with high error rates |
| Token churn | Revoked tokens | `personal_access_tokens WHERE deleted_at IS NOT NULL` |
| Support escalations | Manual reports | Not tracked in codebase |
| Replication failures | Queue job failures | `jobs` table stuck in `processing` |
| Cache miss spikes | Performance complaints | `mls_proxy_audit_logs WHERE cache_hit='MISS'` rate |

### Key insight queries

```sql
-- Domains with highest error rates (potential integration issues)
SELECT domain_slug, request_type,
       COUNT(*) AS total,
       SUM(CASE WHEN cache_hit = 'MISS' THEN 1 ELSE 0 END) AS misses
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY domain_slug, request_type
HAVING COUNT(*) > 10
ORDER BY misses DESC;
```

```sql
-- Users who created domains but never verified (drop-off signal)
SELECT u.email, d.domain_slug, d.verification_status, d.created_at
FROM users u
JOIN domains d ON d.user_id = u.id
WHERE d.verification_status = 'pending'
  AND d.created_at < NOW() - INTERVAL '3 days'
ORDER BY d.created_at;
```

## Scoping Feedback-Driven Features

### Template: From feedback to feature slice

```
## Feedback: [Source] — [Symptom]

**Root cause:** [What's actually broken/missing]
**Evidence:** [SQL query, log pattern, or support ticket]
**Fix scope:**
- Process: [api/worker/scheduler]
- Change: [what code changes]
- Validation: [how to confirm fix]
- Prevention: [audit log or metric to catch recurrence]
```

### Example: Scope fix for "search returns stale data"

```
## Feedback: Customer report — search returns stale data

Root cause: Replication lag; mirror not refreshed since last_sync_finished_at
Evidence: GET /api/v1/bridge/stats shows last_sync_finished_at > 2 hours ago
Fix scope:
- Process: scheduler + worker
- Change: Reduce MLS_REPLICATION_FRESHNESS_MINUTES from 15 to 5
- Validation: Stats endpoint shows syncs within freshness window
- Prevention: Add staleness alert query to monitoring
```

## Triage Framework

| Priority | Signal | Action |
|----------|--------|--------|
| P0 — Outage | `GET /healthz` failing, `GET /readyz` DB down | Fix infrastructure, check Patroni |
| P1 — Data staleness | `last_sync_finished_at` stale by >2x freshness window | Check scheduler logs, worker queue depth |
| P2 — Auth failures | Login 401s, token revocations spiking | Check DNS verification, token expiry |
| P3 — Feature gap | Customer request for new endpoint/filter | Scope as normal feature work |
| P4 — Nice-to-have | Dashboard UX improvement | Backlog, batch with next sprint |

## Anti-Patterns

### WARNING: Building feedback forms in the dashboard

The dashboard is for domain/token management. Do not add feedback forms, NPS surveys, or in-app chat widgets. Feedback comes through support channels and API error patterns.

### WARNING: Acting on single data points

One customer report is a signal, not a feature. Verify with data: check audit logs for the pattern before scoping work. If the query returns zero rows, the issue may be customer-specific (scope as support, not product).

## See Also

- See the **product-analytics** skill for measurement queries
- See the **queue-postgresql** skill for job failure patterns