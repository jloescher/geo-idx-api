# Measurement and Testing Reference

## Contents
- Existing Observability
- Tracking Content Effectiveness
- A/B Testing Constraints
- Anti-Patterns

## Existing Observability

The project uses `slog` for structured logging. All API requests go through audit logging (`audit_logs` table). This is the measurement foundation.

```go
// Audit logging exists in internal/handler/ — every authenticated request is logged
// Key fields: domain, token scope, endpoint, response status, timestamp
// This data answers: "Which docs drive the most API calls?"
```

Key tables for content measurement:

| Table | What it measures |
|-------|-----------------|
| `audit_logs` | API usage by endpoint, domain, and token scope |
| `tokens` | Token creation rate (proxy for signup conversion) |
| `domains` | Domain registration (proxy for new customer acquisition) |

## Tracking Content Effectiveness

Without a dedicated analytics integration, infer content performance from API behavior:

1. **Docs effectiveness**: Track spike in `audit_logs` for an endpoint after publishing its doc
2. **Dashboard CTA effectiveness**: Compare token creation rate before/after dashboard copy change
3. **Release note reach**: Correlate merge timestamp with API usage changes in `audit_logs`

```sql
-- Token creation rate (weekly) — proxy for content conversion
SELECT date_trunc('week', created_at) AS week,
       COUNT(*) AS tokens_created
FROM tokens
GROUP BY week
ORDER BY week DESC;
```

## A/B Testing Constraints

This is a backend API — no client-side A/B framework. Testing approaches:

1. **Docs A/B**: Publish two versions of a doc, route via URL parameter, measure referral API usage
2. **Dashboard copy A/B**: Use feature flags or deploy-time toggle for dashboard text variants
3. **Error message A/B**: Swap error copy between deploys, measure support ticket volume

## Anti-Patterns

### WARNING: Measuring Vanity Metrics

**The Problem:** Tracking page views on docs without connecting to API activation.

**Why This Breaks:** 10k doc views that produce 0 API calls means the content is attracting the wrong audience or failing to convert.

**The Fix:** Always pair traffic metrics with downstream API behavior from `audit_logs`. A doc is effective when readers make their first API call within 24 hours.

### WARNING: No Baseline Before Content Change

**The Problem:** Publishing new dashboard copy without knowing the current token creation rate.

**The Fix:** Before any content change, capture a 2-week baseline from `audit_logs` and `tokens`. Compare post-change against this baseline.