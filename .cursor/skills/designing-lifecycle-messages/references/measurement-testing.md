# Measurement and Testing Reference

## Contents
- Existing instrumentation
- Measuring lifecycle conversion
- A/B testing constraints
- Anti-patterns

## Existing Instrumentation

The platform has **audit logging** via `audit_logs` table. Every authenticated API request is logged. This provides:

- Per-domain API usage (proxy, search, GIS calls)
- Per-token activity patterns
- Timestamp data for activation and retention analysis

**What's NOT instrumented:**
- Dashboard page views (no analytics tracking)
- Email delivery events (no email system)
- Funnel stage transitions (no lifecycle tracking)
- A/B test assignments (no experiment framework)

## Measuring Lifecycle Conversion

### SQL queries for funnel analysis

```sql
-- Activation funnel per user
SELECT
  u.id,
  u.created_at,
  (SELECT MIN(created_at) FROM domains WHERE user_id = u.id) AS first_domain,
  (SELECT MIN(txt_verified_at) FROM domains WHERE user_id = u.id) AS first_verified,
  (SELECT MIN(created_at) FROM personal_access_tokens WHERE tokenable_id = u.id) AS first_token,
  (SELECT MIN(created_at) FROM audit_logs WHERE domain_id IN (SELECT id FROM domains WHERE user_id = u.id)) AS first_api_call
FROM users u
ORDER BY u.created_at DESC;
```

```sql
-- Time-to-activate (verified domain + first API call)
SELECT
  EXTRACT(EPOCH FROM (first_api_call - u.created_at)) / 3600 AS hours_to_activate
FROM users u
JOIN LATERAL (
  SELECT MIN(txt_verified_at) AS first_verified FROM domains WHERE user_id = u.id
) d ON d.first_verified IS NOT NULL
JOIN LATERAL (
  SELECT MIN(created_at) AS first_api_call FROM audit_logs
  WHERE domain_id IN (SELECT id FROM domains WHERE user_id = u.id)
) a ON a.first_api_call IS NOT NULL;
```

These queries work against the existing schema. No additional instrumentation needed for baseline measurement.

### Key metrics

| Metric | Definition | Source |
|---|---|---|
| Invitation acceptance rate | Accepted invites / total invites | `user_invitations` table |
| Domain verification rate | Verified domains / total domains | `domains` table |
| Token activation rate | Users with API calls / users with tokens | `audit_logs` + `personal_access_tokens` |
| Time to first API call | First `audit_logs` entry minus `users.created_at` | Both tables |

## A/B Testing Constraints

### Current limitations

1. **No experiment framework.** Copy changes require code changes and redeployment.
2. **No user segmentation.** No way to show different copy to different users without code-level branching.
3. **Inline HTML.** All copy is in Go string literals — there is no template layer to swap variants at runtime.

### Practical A/B approach given constraints

For copy experiments on this codebase:

1. Deploy variant A, measure for N days via `audit_logs` queries
2. Deploy variant B, measure for N days
3. Compare conversion rates using the SQL queries above

This is sequential testing, not simultaneous A/B. It works for high-traffic pages but is slow for low-volume B2B flows.

### For faster experiments

Add a simple variant system:

```go
// new code to add — minimal experiment assignment
// func copyVariant(userID int64, experiment string) string {
//     if userID % 2 == 0 { return "A" }
//     return "B"
// }
```

Even-odd assignment is not statistically rigorous but is sufficient for directional signals on a small B2B user base. Record the assignment in `audit_logs` for post-hoc analysis.

## Anti-patterns

### WARNING: Measuring email open rates via tracking pixels

Tracking pixels are unreliable (blocked by most email clients) and raise privacy concerns for B2B users. Instead, measure email effectiveness by the **downstream action** — did the user complete the lifecycle stage after the email was sent? Use the `audit_logs` and `user_invitations` tables for this.

### WARNING: Adding client-side analytics without consent

If adding dashboard analytics (page views, click tracking), respect the B2B context. Many Quantyra users are developers who block trackers. Server-side measurement via `audit_logs` is more reliable and less intrusive for this audience.

## Related Skills

- **cache-postgres** — cache experiment assignments
- **queue-postgresql** — enqueue analytics events asynchronously
- **go** — SQL query patterns for funnel analysis