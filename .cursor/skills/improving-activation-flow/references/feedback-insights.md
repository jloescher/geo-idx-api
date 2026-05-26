# Feedback & Insights Reference

## Contents
- Feedback collection surfaces
- Audit log analysis for signals
- Error pattern detection
- Support signal queries
- Anti-patterns

## Feedback Collection Surfaces

This API has limited direct feedback channels (no in-app surveys). Feedback arrives through:

| Channel | Source | Storage |
|---------|--------|---------|
| API errors | HTTP 4xx/5xx responses | `audit_logs` with status codes |
| Token revocation | Customers deleting tokens | `tokens` table `active = false` |
| Domain abandonment | Domains with no API calls | `domains` LEFT JOIN `audit_logs` |
| Support requests | External (email, Slack) | Manual triage |

### DO: Infer frustration from behavior

```sql
-- Domains with repeated 4xx/5xx errors (frustration signal)
SELECT d.hostname,
       COUNT(*) AS error_count,
       COUNT(DISTINCT DATE(a.created_at)) AS error_days
FROM audit_logs a
JOIN domains d ON d.id = a.subject_id
WHERE a.action = 'api.error'
  AND a.created_at > NOW() - INTERVAL '7 days'
GROUP BY d.hostname
HAVING COUNT(*) > 5
ORDER BY error_count DESC;
```

### DON'T: Add feedback forms to API responses

API responses are consumed by code, not humans. Adding `"feedback_url"` fields pollutes the RESO-compliant response shape. Put guidance in the dashboard, not in JSON payloads.

## Audit Log Analysis for Signals

### Activation drop-off detection

```sql
-- Domains that created tokens but never made an API call
SELECT d.hostname, d.created_at, t.created_at AS token_created
FROM domains d
JOIN tokens t ON t.domain_id = d.id
WHERE NOT EXISTS (
    SELECT 1 FROM audit_logs a
    WHERE a.subject_id = d.id
      AND a.action = 'api.request'
)
AND d.created_at > NOW() - INTERVAL '30 days';
```

If this list grows, the problem is between token creation and first API use. Check: is the documentation clear? Is the token being used correctly? See the **auth-api-token** skill.

### Feature adoption lag

```sql
-- Domains using proxy but not search (missing feature discovery)
SELECT d.hostname
FROM domains d
WHERE EXISTS (SELECT 1 FROM audit_logs a WHERE a.subject_id = d.id AND a.metadata->>'endpoint' LIKE '/api/v1/properties%')
  AND NOT EXISTS (SELECT 1 FROM audit_logs a WHERE a.subject_id = d.id AND a.metadata->>'endpoint' = '/api/v1/search')
  AND d.created_at < NOW() - INTERVAL '14 days';
```

## Error Pattern Detection

### DO: Categorize errors for actionable insights

```go
// new code to add — error categorization in audit log
func CategorizeError(statusCode int, path string) string {
    switch {
    case statusCode == 401:
        return "auth.failure"
    case statusCode == 403:
        return "auth.forbidden"
    case statusCode == 404 && strings.Contains(path, "/properties"):
        return "data.not_found"
    case statusCode == 429:
        return "rate_limit.hit"
    case statusCode >= 500:
        return "server.error"
    default:
        return "unknown"
    }
}
```

### DON'T: Log full request bodies in audit logs

PII and listing data can end up in audit logs. Log only: action, subject_type, subject_id, status code, endpoint, and error category.

## Support Signal Queries

### High-value domains at risk

```sql
-- Domains with declining API usage (weekly comparison)
WITH this_week AS (
    SELECT subject_id AS domain_id, COUNT(*) AS calls
    FROM audit_logs
    WHERE action = 'api.request'
      AND created_at > NOW() - INTERVAL '7 days'
    GROUP BY subject_id
),
last_week AS (
    SELECT subject_id AS domain_id, COUNT(*) AS calls
    FROM audit_logs
    WHERE action = 'api.request'
      AND created_at BETWEEN NOW() - INTERVAL '14 days' AND NOW() - INTERVAL '7 days'
    GROUP BY subject_id
)
SELECT d.hostname, lw.calls AS last_week, tw.calls AS this_week,
       ROUND(100.0 * (tw.calls - lw.calls) / NULLIF(lw.calls, 0), 1) AS pct_change
FROM domains d
JOIN last_week lw ON lw.domain_id = d.id
LEFT JOIN this_week tw ON tw.domain_id = d.id
WHERE tw.calls IS NULL OR tw.calls < lw.calls * 0.5
ORDER BY pct_change ASC;
```

## Anti-patterns

### WARNING: Using error responses for product feedback

```go
// BAD — adding survey prompts to error responses
return c.Status(500).JSON(fiber.Map{
    "error": "Internal server error",
    "feedback_url": "https://survey.example.com/...", // DO NOT DO THIS
})
```

Error responses are consumed by client code. Keep them machine-readable. Feedback collection belongs in the dashboard or support channels, not in API error payloads. See the **ux** skill for proper error state handling.