# Growth Engineering Reference

## Contents
- Product-led growth surfaces
- Invitation-driven growth
- API usage as retention signal
- Anti-patterns

## Product-Led Growth Surfaces

Quantyra IDX is a **developer platform** with a narrow, well-defined activation path:

```
Admin invites user → User registers → User adds domain → User verifies DNS → User gets token → User makes first API call
```

Each step is a growth lever. The fastest path to "activated" is the shortest path through this sequence.

### Reducing friction at each stage

| Stage | Current friction | Potential improvement |
|---|---|---|
| Invitation | Admin copies link manually | Email the invite link automatically (requires email sender) |
| Registration | Name + password form | Pre-fill email from invitation token |
| Domain verification | DNS TXT record setup | Add common DNS provider instructions (Cloudflare, Route53) |
| Token creation | Manual after verification | Auto-create on domain verification (already done — line 224) |
| First API call | No guidance | Show curl example with the user's domain and token |

## Invitation-Driven Growth

The existing invitation system (`internal/service/auth/invitations.go`) is admin-only:

```go
// internal/handler/dashboard/handler.go:57 — admin gate on invitations
app.Post("/dashboard/invitations", h.requireAuth, h.requireAdmin, h.CreateInvitation)
```

Only admins can invite. The invitation token is a 64-char hex string with a 168-hour TTL:

```go
// internal/service/auth/invitations.go:41 — TTL from config
expires := time.Now().Add(s.cfg.Auth.InvitationTTL)
```

### Growth levers within the invite model

1. **Reduce admin friction:** One-click invite from the dashboard with email delivery (no link copying).
2. **Referral tokens:** Let non-admin users generate limited-scope invite links for their team members.
3. **Domain-scoped invites:** Pre-assign a domain to the invitation so the new user skips the "add domain" step.

### Tracking invitation funnel

```sql
-- Invitation funnel
SELECT
  COUNT(*) AS total_sent,
  COUNT(accepted_at) AS accepted,
  COUNT(*) - COUNT(accepted_at) AS pending_or_expired,
  AVG(EXTRACT(EPOCH FROM (accepted_at - created_at)) / 3600) AS avg_hours_to_accept
FROM user_invitations;
```

## API Usage as Retention Signal

The `audit_logs` table is the retention dataset. Active users make regular API calls through their domains.

```sql
-- Weekly active domains (retention proxy)
SELECT
  DATE_TRUNC('week', created_at)::date AS week,
  COUNT(DISTINCT domain_id) AS active_domains
FROM audit_logs
GROUP BY 1 ORDER BY 1 DESC LIMIT 12;
```

```sql
-- Churn risk: domains with API calls in the last 30 days but none in the last 7
SELECT d.domain_slug, MAX(a.created_at) AS last_call
FROM domains d
JOIN audit_logs a ON a.domain_id = d.id
WHERE a.created_at > NOW() - INTERVAL '30 days'
GROUP BY d.id
HAVING MAX(a.created_at) < NOW() - INTERVAL '7 days';
```

Use churn risk queries to trigger re-engagement emails (when email is implemented). See the **distribution** reference for sequence timing.

## Anti-patterns

### WARNING: Growth tactics that violate MLS compliance

Bridge and Spark MLS data is regulated. Never:
- Use listing data in marketing emails without checking MLS rules
- Display listing details in referral or invitation messages
- Expose listing counts or market statistics as growth hooks without verifying RESO compliance

See the **geospatial** skill and `docs/spark/` for MLS compliance constraints.

### WARNING: Unbounded invitation creation

The current system has no rate limit on admin invitation creation. In a growth context with referral tokens, add a per-user invitation cap to prevent abuse. Check invitations against existing domain limits before allowing new invites.

## Related Skills

- **auth-api-token** — token scopes for referral invites
- **queue-postgresql** — enqueue invitation emails
- **cache-postgres** — cache churn risk calculations
- **geospatial** — MLS data compliance constraints