# Conversion Optimization Reference

## Contents
- Conversion funnel stages
- Landing page optimization
- Dashboard setup completion
- One-time secret display pattern
- Anti-patterns

## Conversion Funnel Stages

The platform has a linear funnel with four measurable stages:

1. **Landing** (`/`) — visitor sees hero, must click CTA
2. **Auth** (`/login` or `/invite/:token`) — email/password or invite signup
3. **Setup** (`/dashboard`) — add domain, verify DNS TXT
4. **Activation** — domain verified, production token generated (one-time display)

Each stage has exactly one primary action. The funnel is narrow by design (invite-only).

## Landing Page Optimization

The hero in `marketing/handler.go` has two CTAs:

```go
// existing — two CTAs competing for attention
<div class="hero-actions">
<a class="btn btn-primary" href="/dashboard">Open dashboard</a>
<a class="btn btn-secondary" href="/login">Sign in</a>
</div>
```

### DO: Single primary CTA for unauthenticated visitors

Unauthenticated visitors hitting "Open dashboard" get redirected to `/login` by `requireAuth`. This adds a redirect hop. Instead, detect auth state and show the right CTA:

```go
// new code to add — check session before rendering
sess, _ := h.sessions.Get(c)
uid := sess.Get("user_id")
primaryAction := "/login"
primaryLabel := "Get started"
if uid != nil {
    primaryAction = "/dashboard"
    primaryLabel = "Open dashboard"
}
```

### DON'T: Add navigation links to the hero

The header nav already has Dashboard and Login links. Duplicating them in the hero body dilutes the primary CTA. One clear action per section.

## Dashboard Setup Completion

The dashboard (`dashboard/handler.go:117`) shows four cards. The visual order determines what users do first:

1. Setup (domain list)
2. API keys
3. Add domain
4. Invite user (admin only)

### WARNING: Add Domain is below the fold

The "Add domain" form is the critical activation action but appears as the third card. New users with zero domains see two empty cards before the action they need.

**Fix:** Conditionally reorder. When user has zero domains, show "Add domain" card first:

```go
// new code to add — reorder cards for new users
hasDomains := rows.Next()
if !hasDomains {
    // render "Add domain" card first, then empty state for others
}
```

### DO: Show empty states with clear next action

When domain list and token list are empty, the current code shows an empty `<ul>` with no items. Replace with a call-to-action:

```html
<!-- new code to add — empty state -->
<div class="card"><h2>Get started</h2>
<p>Add your first domain to generate an API key.</p>
<a class="btn btn-primary" href="#add-domain">Add domain</a>
</div>
```

### DON'T: Show "API keys" section before domain exists

Tokens without a verified domain are unusable. The API keys card should appear only after at least one domain exists, or should be collapsed/disabled.

## One-Time Secret Display Pattern

Both `VerifyTXT` and `CreateInvitation` follow the same pattern:

```go
// existing pattern — shown once, never stored plaintext
body := `<div class="card"><h1>Domain verified</h1>
<p>Save this production token now — it will not be shown again.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
```

Key elements:
- Urgent warning copy ("will not be shown again")
- Monospace token box for easy selection
- Single CTA back to dashboard (not to another secret page)
- `data-copy` JS helper exists in `app.js` but is not wired to `.token-box` yet

### DO: Add clipboard copy button to token boxes

```html
<!-- new code to add -->
<div class="token-box" id="token">TOKEN_HERE</div>
<button data-copy="#token" class="btn btn-sm btn-secondary">Copy</button>
```

The `data-copy` listener already exists in `app.js:3-9`.

## Anti-Patterns

### WARNING: Inline HTML without escaping

```go
// BAD — XSS if slug contains HTML
b.WriteString("<li><strong>" + slug + "</strong></li>")

// GOOD — always escape user content
b.WriteString("<li><strong>" + web.Esc(slug) + "</strong></li>")
```

Existing code at `dashboard/handler.go:133` already uses `web.Esc()` for slugs and tokens. Maintain this pattern.

### WARNING: Form without CSRF protection

The login, domain, and token forms use POST without CSRF tokens. Session-based auth with no CSRF is a known risk. If adding new forms, match the existing pattern but flag for future hardening.

## Audit Trail for Conversions

Every API request through `domainAuth` middleware writes to `audit_logs` via `internal/service/audit/`. Use this to measure:

- First API call after token creation (activation metric)
- Domain verification latency (setup friction)
- Token revocation frequency (churn signal)

Query:

```sql
-- new code to add — activation metric
SELECT u.email, MIN(a.created_at) AS first_api_call
FROM audit_logs a JOIN users u ON u.id = a.user_id
GROUP BY u.email ORDER BY first_api_call;
```