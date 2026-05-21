# Feedback & Insights Reference

## Contents
- Feedback Collection
- Signal Sources
- Triage Workflow
- Anti-Patterns

## Feedback Collection

The project has no in-app feedback widget, no support ticket system, and no NPS survey mechanism. Feedback comes through:

1. **Direct communication** — email, Slack, or verbal from MLS administrators
2. **Audit logs** — behavioral signals showing where users struggle
3. **Error responses** — HTTP 422/409/400 patterns from dashboard form submissions
4. **Database state** — abandoned setups (domain added but never verified)

## Signal Sources

### Abandoned Domain Setup

Users who add a domain but never verify it represent a clear friction point:

```sql
SELECT domain_slug, verification_status, created_at, verification_checked_at
FROM domains
WHERE verification_status = 'pending'
  AND created_at < NOW() - INTERVAL '7 days'
ORDER BY created_at;
```

If this query returns rows, the DNS TXT verification flow needs better guidance. See the **in-app-guidance** reference for enhanced verification error messaging.

### Staging Token Without Production Token

```sql
SELECT u.email, COUNT(t.id) AS token_count
FROM users u
JOIN personal_access_tokens t ON t.tokenable_id = u.id
WHERE t.name = 'Staging'
  AND NOT EXISTS (
    SELECT 1 FROM personal_access_tokens t2
    WHERE t2.tokenable_id = u.id AND t2.name != 'Staging'
  )
GROUP BY u.email;
```

Users with only a staging token haven't completed domain verification. They may be stuck on the DNS step.

### Verification Retry Patterns

```sql
SELECT domain_slug, verification_checked_at, verification_status
FROM domains
WHERE verification_checked_at IS NOT NULL
  AND verification_status = 'pending'
ORDER BY verification_checked_at DESC;
```

Multiple `verification_checked_at` updates on a pending domain indicate the user is retrying but failing — the DNS record instructions may be unclear.

### Zero API Activity After Token Creation

```sql
SELECT t.id, t.name, t.created_at,
       (SELECT COUNT(*) FROM audit_logs a WHERE a.token_id = t.id) AS api_calls
FROM personal_access_tokens t
WHERE t.created_at < NOW() - INTERVAL '7 days'
ORDER BY api_calls ASC
LIMIT 20;
```

Tokens with zero API calls after a week suggest integration difficulty. The "Next steps" guidance (see **engagement-adoption** reference) should help.

## Triage Workflow

Copy this checklist and track progress:

- [ ] Run abandoned-setup query weekly (pending domains older than 7 days)
- [ ] Check retry patterns (multiple verification attempts on same domain)
- [ ] Review zero-activity tokens (created but never used)
- [ ] Cross-reference with any direct user feedback
- [ ] Prioritize: DNS verification friction > API integration confusion > other
- [ ] Implement fix in dashboard handler with proper error page wrapping
- [ ] Verify: `go build ./cmd/...` and `go test ./...` pass
- [ ] Deploy and monitor the same queries the following week

## Anti-Patterns

- **NEVER** add a "feedback" button or in-app survey widget. The dashboard serves a small, known set of MLS administrators. Collect feedback through direct communication.
- **AVOID** sending automated emails for abandoned setups. The platform has no email sending infrastructure. Use in-dashboard guidance instead.
- **NEVER** log user feedback in the `audit_logs` table. That table is for authenticated API request tracking. Use a separate mechanism (issue tracker, Slack channel).
- **AVOID** querying production tables during peak replication hours for analytics. Run insight queries during low-traffic periods or against a read replica if available (see **deploy-patroni** skill for Phase 2 read replicas).

## Related Skills

- See the **postgres** skill for query optimization
- See the **auth-api-token** skill for audit log structure
- See the **deploy-patroni** skill for read replica patterns