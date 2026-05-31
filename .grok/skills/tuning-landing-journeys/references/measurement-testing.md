# Measurement & Testing Reference

## Contents
- Existing instrumentation
- Activation metrics
- Funnel measurement queries
- Testing constraints
- Anti-patterns

## Existing Instrumentation

| Signal | Storage | Location |
|--------|---------|----------|
| API requests | `audit_logs` table | `internal/service/audit/` |
| Domain verification | `domains.verification_status`, `txt_verified_at` | `internal/repository/domain.go` |
| Token creation/revocation | `personal_access_tokens` table | `internal/repository/token.go` |
| User invitations | `user_invitations` table | `internal/repository/invitation.go` |
| Session auth | Fiber session store (in-memory or cookie) | `dashboard/handler.go` |
| Health | `/healthz`, `/readyz` endpoints | `internal/api/routes.go` |

No client-side analytics (Google Analytics, Plausible, PostHog, etc.) are installed. All measurement is server-side via PostgreSQL tables.

## Activation Metrics

The core activation metric is **time from invitation to first API call**. This spans:

1. Invitation created (`user_invitations.created_at`)
2. Invitation accepted (`user_invitations.accepted_at`)
3. First domain added (`domains.created_at WHERE user_id = X`)
4. Domain verified (`domains.txt_verified_at`)
5. First API call (`audit_logs.created_at WHERE user_id = X`)

### Funnel measurement queries

```sql
-- new code to add — invitation-to-activation funnel
SELECT
  COUNT(*) AS invited,
  COUNT(i.accepted_at) AS accepted,
  COUNT(d.id) AS has_domain,
  COUNT(d.txt_verified_at) AS verified,
  COUNT(a.first_call) AS activated
FROM user_invitations i
LEFT JOIN domains d ON d.user_id = i.invitee_user_id
LEFT JOIN LATERAL (
  SELECT MIN(created_at) AS first_call FROM audit_logs WHERE user_id = i.invitee_user_id
) a ON true;
```

```sql
-- new code to add — domain verification drop-off
SELECT
  COUNT(*) FILTER (WHERE verification_status = 'pending') AS stuck_pending,
  COUNT(*) FILTER (WHERE verification_status = 'verified') AS verified,
  AVG(EXTRACT(EPOCH FROM (txt_verified_at - created_at))/60) AS avg_minutes_to_verify
FROM domains;
```

## Testing Constraints

### No A/B test framework

The server-rendered HTML approach has no built-in A/B testing. Options:

| Approach | Complexity | Recommendation |
|----------|------------|----------------|
| Deploy-toggle copy variants | Low — env var controls which headline renders | Start here |
| Database-backed copy store | Medium — `landing_variants` table with key/value | Scale to later |
| Client-side A/B tool | High — conflicts with server-rendered model | Avoid |

### DO: Start with deploy-toggle experiments

```go
// new code to add — env-controlled copy variant
headline := h.cfg.LandingHeadline() // defaults to "Quantyra IDX"
if headline == "" {
    headline = "Quantyra IDX"
}
body := fmt.Sprintf(`<section class="hero"><h1>%s</h1>...`, web.Esc(headline))
```

Run variant A for a week, measure via `audit_logs`, then deploy variant B.

### DON'T: Use client-side redirect for A/B

Redirecting to different URLs for A/B tests breaks analytics (referrer loss) and adds latency. Render the variant server-side in the same handler.

## Missing Instrumentation

### WARNING: No page view tracking

The platform has no page view analytics. You cannot measure:
- Landing page bounce rate
- Login form abandonment
- Dashboard card interaction (which cards users interact with)
- Time to first action on any page

**Quick win:** Add a lightweight server-side page view log:

```go
// new code to add — middleware in layout.go or routes.go
func pageViewLog(c *fiber.Ctx) error {
    // log path, user_id (if session), timestamp to a page_views table
    return c.Next()
}
```

This preserves the no-client-analytics architecture while giving funnel visibility.

### WARNING: No error tracking for form submissions

Failed form submissions (`StoreDomain`, `CreateToken`) return raw error text. There is no structured error tracking. Add error classification:

```go
// new code to add — classify errors for measurement
type FormError struct {
    Form   string `json:"form"`
    Field  string `json:"field"`
    Reason string `json:"reason"`
}
```

## Anti-Patterns

### WARNING: Measuring activation with API call count

Raw API call count in `audit_logs` includes automated/polling requests. Filter by endpoint and user to measure genuine activation:

```sql
-- BAD — includes health checks, replication, etc.
SELECT COUNT(*) FROM audit_logs WHERE user_id = $1;

-- GOOD — activation-relevant endpoints only
SELECT COUNT(DISTINCT DATE(created_at)) AS active_days
FROM audit_logs
WHERE user_id = $1
  AND endpoint IN ('/api/v1/search', '/api/v1/properties', '/api/v1/comps/run');
```

### WARNING: Session-based auth has no duration tracking

Fiber sessions expire after `cfg.Auth.SessionLifetime` but there is no record of session creation, duration, or expiry. You cannot measure session length or login frequency without additional instrumentation.