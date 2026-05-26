# Measurement and Testing Reference

Measure release story impact through Quantyra IDX API's existing observability surfaces.

## Contents
- Available Metrics
- Release Validation Workflow
- Anti-Patterns

## Available Metrics

| Metric | Source | Query |
|--------|--------|-------|
| API request volume | `audit_logs` table | `SELECT COUNT(*) FROM audit_logs WHERE created_at > NOW() - interval '1 day'` |
| Cache hit rate | `X-IDX-Cache` response header | Check proxy response headers |
| Queue depth | `jobs` table | `SELECT queue, COUNT(*) FROM jobs GROUP BY queue` |
| Replication lag | `listing_sync_cursors` | `SELECT dataset_slug, last_modification_timestamp FROM listing_sync_cursors` |
| Active tokens | `personal_access_tokens` | `SELECT COUNT(*) FROM personal_access_tokens WHERE last_used_at > NOW() - interval '7 days'` |
| Domain verifications | `domains` | `SELECT verification_status, COUNT(*) FROM domains GROUP BY verification_status` |
| Listing freshness | `listings` | `SELECT dataset_slug, MAX(modification_timestamp) FROM listings GROUP BY dataset_slug` |

### WARNING: No product analytics instrumentation

**The Problem:** The codebase has no event tracking, funnel analytics, or feature usage metrics beyond audit logs. You cannot measure whether a release story drove adoption of a new feature.

**Why This Breaks:** Without tracking which endpoints users call after a release, you cannot validate whether the narrative landed. You can see request volume changes but not attribute them to specific announcements.

**The Fix:** Use audit log queries as a proxy. Before announcing a feature, capture baseline request counts for the relevant endpoint. Compare 7 days pre- and post-announcement.

## Release Validation Workflow

```markdown
Feedback loop for release stories:
1. Pre-release: Capture baseline metrics (audit_logs count, queue depth, token usage)
2. Deploy: Follow coolify-deployment.md deploy order
3. Verify: `GET /healthz`, `GET /readyz` on both DCs
4. Smoke: Test the announced feature via curl with a staging token
5. Distribute: Send release story to available channels
6. Measure: Compare audit_logs volume for affected endpoints at +1, +7, +30 days
7. Iterate: If adoption is low, revise messaging and redistribute
```

### Quick validation commands

```bash
# new code to add — pre-release baseline
curl -s https://idx-api.quantyralabs.cc/healthz
curl -s https://idx-api.quantyralabs.cc/readyz

# Post-release: verify replication is running
# Check scheduler logs for "scheduler leader acquired" and "enqueued fetch"
```

## Anti-Patterns

### WARNING: Measuring release success by deploy speed

**The Problem:** Treating "deployed without rollback" as the success metric for a release story.

**Why This Breaks:** A clean deploy says nothing about whether API consumers adopted the new feature or understood the breaking change. The release story's goal is adoption, not deployment.

**The Fix:** Define a success metric before writing the story: "50% of active tokens call the new endpoint within 30 days" or "zero support tickets about the auth change within 7 days."

### WARNING: No rollback measurement plan

**The Problem:** Planning the release without defining what "rollback" looks like from a narrative perspective.

**Why This Breaks:** If the release is rolled back, previously distributed stories become inaccurate. Without a measurement plan, you don't know when to pull back the announcement.

**The Fix:** Define rollback triggers before deploy: "If error rate exceeds X% on the new endpoint within 1 hour, roll back and update release notes with 'temporarily unavailable'."

## Integration points

- `internal/service/audit/logger.go` — audit log writes for all authenticated requests
- `internal/service/cache` — cache hit/miss data via `X-IDX-Cache` header
- `GET /api/v1/bridge/stats` — replication status per dataset
- `internal/queue` — job processing metrics via `jobs` table

See the **queue-postgresql** skill for job monitoring.
See the **cache-postgres** skill for cache performance patterns.