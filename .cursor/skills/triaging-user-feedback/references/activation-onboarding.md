# Activation & Onboarding Feedback

## Contents
- Onboarding Journey Stages
- Friction Signals by Stage
- Dashboard Activation Patterns
- Quick Win Opportunities

## Onboarding Journey Stages

The Quantyra IDX onboarding flow spans: invite → login → domain setup → DNS verification → token creation → first API call. Each stage has observable friction points in `internal/handler/dashboard/handler.go`.

### Stage 1: Invite and Login

The dashboard is invite-only. Admins create invitations via `/dashboard/invitations`. New users receive credentials and log in at the centered login form.

**Friction signals:**
- Failed login attempts (check slog for `Invalid credentials`)
- Session creation failures
- Users who log in but never add a domain

```sql
-- Users who logged in but never registered a domain (activation drop-off)
SELECT u.id, u.email, u.created_at
FROM users u
LEFT JOIN domains d ON d.user_id = u.id
WHERE d.id IS NULL
  AND u.created_at < NOW() - INTERVAL '3 days'
ORDER BY u.created_at DESC;
```

### Stage 2: Domain Setup and DNS Verification

Domain registration at `/dashboard/domains` requires DNS TXT record publication. This is the highest-friction step.

**Key error surface** (`internal/handler/dashboard/handler.go`):
```go
return c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")
```

**Friction signals:**
- Multiple 422 responses for same domain (DNS confusion)
- Domains stuck in `pending` status for > 24 hours
- Users retrying verification repeatedly

```sql
-- Domains stuck in pending verification
SELECT slug, created_at, NOW() - created_at AS age
FROM domains
WHERE verification_status = 'pending'
  AND created_at < NOW() - INTERVAL '24 hours'
ORDER BY created_at;
```

### Stage 3: Token Creation and First API Call

Token creation at `/dashboard/api-tokens` generates a one-time display. The copy-to-clipboard UX (`internal/web/static/js/app.js` with `data-copy` attribute) reduces friction but the token is shown only once.

**Friction signals:**
- Tokens created but never used (check `mls_proxy_audit_logs`)
- Token revocation shortly after creation (confusion about scopes)
- Multiple tokens created for same domain (unable to find previous token)

## Dashboard Activation Patterns

### DO: Track activation funnel with audit data

```sql
-- Activation funnel: invited → logged in → domain added → verified → token used
SELECT
  COUNT(DISTINCT u.id) AS total_users,
  COUNT(DISTINCT d.user_id) AS with_domain,
  COUNT(DISTINCT CASE WHEN d.verification_status = 'verified' THEN d.user_id END) AS verified,
  COUNT(DISTINCT a.domain_slug) AS made_api_call
FROM users u
LEFT JOIN domains d ON d.user_id = u.id
LEFT JOIN (SELECT DISTINCT domain_slug FROM mls_proxy_audit_logs) a ON a.domain_slug = d.slug;
```

### DON'T: Rely on anecdotal feedback alone

A single user saying "DNS verification is confusing" is a signal. Finding that 40% of domains sit in `pending` for > 48 hours is a priority. Always cross-reference qualitative feedback with audit data.

## Quick Win Opportunities

| Quick Win | Effort | Impact |
|-----------|--------|--------|
| Show DNS instructions inline instead of separate page | Small | Reduces verification retries |
| Add "Test DNS" button that re-checks without page reload | Small | Faster verification loop |
| Display token scope descriptions during creation | Small | Reduces token confusion |
| Show "first API call" example after token creation | Small | Faster time-to-first-call |

## Related Skills

- See the **ux** skill for error message design patterns
- See the **frontend-design** skill for dashboard UI patterns
- See the **auth-api-token** skill for token scope details