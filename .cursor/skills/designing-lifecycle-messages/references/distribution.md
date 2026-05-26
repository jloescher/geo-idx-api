# Distribution Reference

## Contents
- Current delivery channels
- Email delivery architecture
- In-dashboard messaging
- Anti-patterns

## Current Delivery Channels

The platform has **two** active message channels today:

| Channel | Implementation | Reach |
|---|---|---|
| Dashboard HTML | Inline string literals in handlers | Users who are logged in |
| API responses | JSON/plain text error messages | API consumers (developers) |

**No email delivery is implemented.** `MAIL_*` environment variables exist in `.env.example` but no code sends email. The invitation system generates a link that admins must share manually.

## Email Delivery Architecture

When email is added, it must go through the existing PostgreSQL job queue — not inline in handlers. See the **queue-postgresql** skill for queue patterns.

### Queue job types for lifecycle email

```go
// new code to add — proposed job types
// "email.send"           — single email delivery
// "email.sequence.start" — begin a drip sequence for a user
// "email.sequence.next"  — advance to the next email in a sequence
```

### Why queue-based delivery

1. SMTP is slow (1-30s per message). Blocking HTTP handlers on SMTP causes timeouts.
2. Workers provide automatic retry on transient failures.
3. The scheduler can enqueue drip sequence emails on a cron, keeping sequences decoupled from user actions.
4. The existing `audit_logs` table can track email delivery status alongside API calls.

### Sequence timing for B2B developer onboarding

| Delay after trigger | Email | Purpose |
|---|---|---|
| Immediate | Welcome + invitation link | Drive registration |
| +1 hour (if domain not verified) | Domain setup reminder | Reduce verification drop-off |
| +24 hours (if no API call) | First API call guide | Drive activation |
| +7 days (if inactive) | Tips and docs links | Re-engagement |

Store sequence state in a `user_lifecycle` table (new) or extend the existing `users` table with lifecycle tracking columns. The scheduler checks state on each cron tick and enqueues the next email if conditions are met.

## In-Dashboard Messaging

Dashboard messages are the primary channel today. They render synchronously as HTML — no queue needed.

### Existing message touchpoints

```go
// internal/handler/dashboard/handler.go:225 — post-verification confirmation
// Shows one-time token in a styled card

// internal/handler/dashboard/handler.go:269 — post-invitation confirmation
// Shows one-time invite link in a styled card

// internal/web/layout.go:43 — login page subtitle
// "Access your MLS domains and API keys."
```

### Adding new in-dashboard messages

Place new messages within the existing card structure. Use `web.Esc()` for any dynamic content. Use existing CSS classes (`card`, `badge-*`, `btn-*`, `token-box`) for consistent styling. See the **frontend-design** skill.

## Anti-patterns

### WARNING: Email delivery in API request path

Never send email directly in a Fiber handler. The Go runtime will block the HTTP response on SMTP I/O. Even with goroutines, unbounded email sends can exhaust connections or leak on server shutdown. Always enqueue through the PostgreSQL job queue and let workers handle delivery.

### WARNING: Storing plaintext email content in the jobs table

The `jobs` table stores `payload` as JSONB. If email payloads contain tokens, invitation links, or API keys, these values are readable by anyone with database access. Hash or omit sensitive values from the job payload; regenerate them in the worker if needed.

## Related Skills

- **queue-postgresql** — enqueue and process email delivery jobs
- **scheduler** — cron-based drip sequence triggers
- **cache-postgres** — cache email templates
- **go** — Go service patterns for email sender module