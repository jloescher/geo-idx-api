# Activation & Onboarding Reference

## Contents
- Activation Funnel
- First-Run Detection
- Progressive Setup Steps
- Empty State Patterns
- Anti-Patterns

## Activation Funnel

The Quantyra IDX activation funnel maps to four concrete states queryable from `domains` and `personal_access_tokens`:

1. **Account created** — user row exists in `users` (via `/invite/:token` or `make seed-admin`)
2. **Domain registered** — row in `domains` with `verification_status = 'pending'`
3. **Domain verified** — `verification_status = 'verified'` (DNS TXT confirmed)
4. **First API call** — row in `mls_proxy_audit_logs` with matching `domain_slug`

### Querying Activation Depth

```sql
-- new code to add — measure activation per user
SELECT u.id,
       u.email,
       COUNT(DISTINCT d.id)                                         AS domains,
       COUNT(DISTINCT d.id) FILTER (WHERE d.verification_status = 'verified') AS verified,
       COUNT(DISTINCT t.id)                                         AS tokens,
       (SELECT COUNT(*) FROM mls_proxy_audit_logs a
        WHERE a.domain_slug IN (SELECT domain_slug FROM domains WHERE user_id = u.id)
       ) > 0                                                        AS has_api_call
FROM users u
         LEFT JOIN domains d ON d.user_id = u.id
         LEFT JOIN personal_access_tokens t ON t.tokenable_id = u.id
GROUP BY u.id, u.email;
```

## First-Run Detection

The dashboard handler (`internal/handler/dashboard/handler.go:117`) queries domains and tokens directly. Detect first-run by checking if both collections are empty:

```go
// new code to add — inside Dashboard handler, after existing queries
isFirstRun := true
// rows.Next() returns false when user has zero domains
hasDomains := false
for rows.Next() {
    hasDomains = true
    // ... existing domain rendering
}
if hasDomains {
    isFirstRun = false
}
```

Avoid querying `mls_proxy_audit_logs` on every dashboard load — cache the "has first API call" flag in the session or a user preferences column.

## Progressive Setup Steps

The onboarding sequence is linear: domain → DNS verify → token → first call. Each step gates the next:

| Step | Route | Gate Check | Completion Signal |
|------|-------|-----------|-------------------|
| Register domain | `POST /dashboard/domains` | `requireAuth` | Row in `domains` |
| Verify DNS TXT | `POST /dashboard/domains/:id/verify-txt` | `verification_status = 'pending'` | Status → `verified` |
| Receive token | Auto on verify success (`handler.go:224`) | Domain verified | `personal_access_tokens` row |
| First API call | External (client code) | Valid token + domain | `mls_proxy_audit_logs` row |

### Token Auto-Creation on Verify

The existing code already creates a production token immediately after DNS verification succeeds (`handler.go:224`):

```go
// existing — internal/handler/dashboard/handler.go:224
plain, _ := h.tokens.Create(c.Context(), uid, "Production", []string{"idx:full"})
```

This is the key activation moment — the user sees the token once and must copy it. The token-box CSS class (`app.css:193`) styles it with a monospace dashed-border box for scannability.

## Empty State Patterns

The dashboard currently shows empty `<ul>` containers when domains or tokens are missing. Replace with a guided empty state:

```html
<!-- new code to add — empty state when len(domains) == 0 -->
<div class="card">
    <h2>Add your first domain</h2>
    <p>Register the domain where your IDX site runs. We will verify ownership via a DNS TXT record.</p>
    <form method="post" action="/dashboard/domains" class="inline-form">
        <label>Hostname <input name="domain_slug" type="text" placeholder="www.example.com" required></label>
        <label>MLS dataset <input name="mls_dataset" type="text" value="stellar"></label>
        <button type="submit" class="btn btn-primary">Add domain</button>
    </form>
</div>
```

Match the existing card + inline-form CSS from `app.css`. Do not introduce new CSS classes for onboarding cards — reuse `.card`, `.form-stack`, `.inline-form`, `.btn-primary`.

## Anti-Patterns

### WARNING: Frontend-Only Onboarding State

**The Problem:** Storing onboarding progress in localStorage or a cookie while the real state lives in PostgreSQL.

**Why This Breaks:**
1. State diverges from server truth — user clears browser, onboarding resets but domain is verified
2. Multi-device inconsistency — onboarding shows on phone but not laptop
3. No server-side analytics — you cannot measure funnel conversion from localStorage

**The Fix:** Derive all onboarding state from `domains` and `personal_access_tokens` queries that the dashboard already runs. The handler owns state; the template renders it.

### WARNING: Blocking Activation on Email Verification

**The Problem:** Requiring email confirmation before showing the dashboard.

**Why This Breaks:** The invite-only flow (`/invite/:token`) already validates identity — the admin chose the recipient. Adding email confirmation creates a second gate with no security benefit, and email deliverability issues (spam folder, corporate firewalls) will silently kill activation.

**The Fix:** After `AcceptInvitation` succeeds (`handler.go:282`), redirect straight to `/login`. The user already proved identity by possessing the invite token.

## Cross-References

- See the **auth-api-token** skill for token creation, hashing, and ability scopes
- See the **ux** skill for accessibility and state matrix patterns
- See the **fiber** skill for route registration and middleware chains
- See the **frontend-design** skill for CSS patterns matching the dark theme