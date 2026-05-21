# Engagement & Adoption Reference

## Contents
- Feature Discovery
- Progressive Disclosure
- Adoption Signals
- Anti-Patterns

## Feature Discovery

The dashboard surfaces are: domain management, API token management, and admin invitation. Each has a clear entry point on the single-page dashboard. Discovery is not the problem — completion is.

### Current Dashboard Surfaces

| Surface | Route | Empty State Exists? |
|---------|-------|-------------------|
| Domain list | `GET /dashboard` | No — empty `<ul>` renders invisible |
| API token list | `GET /dashboard` | No — empty `<ul>` renders invisible |
| Add domain form | `GET /dashboard` | Always visible |
| Create token form | `GET /dashboard` | Always visible |
| Invite user (admin) | `GET /dashboard` | Conditionally visible |

### Progressive Disclosure Pattern

Forms are already always visible (no collapse/expand). For new features, hide advanced options behind a toggle or secondary action:

```go
// new code to add — example of progressive disclosure for MLS dataset selection
b.WriteString(`<details><summary>Advanced: MLS dataset</summary>`)
b.WriteString(`<label>MLS dataset <input name="mls_dataset" type="text" value="stellar"></label>`)
b.WriteString(`</details>`)
```

This keeps the primary flow simple while allowing customization.

## Adoption Signals

Track whether users complete the activation path using existing database queries:

### Signal 1: Domain Added Within 24 Hours

```sql
SELECT u.id, u.created_at,
       (SELECT COUNT(*) FROM domains WHERE user_id = u.id) AS domain_count
FROM users u
WHERE u.created_at > NOW() - INTERVAL '7 days'
ORDER BY u.created_at DESC;
```

### Signal 2: Domain Verified

```sql
SELECT COUNT(*) FROM domains WHERE verification_status = 'verified';
```

### Signal 3: API Key Used

```sql
SELECT t.name, COUNT(a.id) AS request_count
FROM personal_access_tokens t
LEFT JOIN audit_logs a ON a.token_id = t.id
GROUP BY t.name;
```

See the **auth-api-token** skill for the audit log schema.

### Signal 4: Staging Token Created

```sql
SELECT COUNT(*) FROM personal_access_tokens
WHERE name = 'Staging' AND tokenable_type = 'App\Models\User';
```

## Engagement Patterns

### After Domain Verification — One-Time Token Reveal

```go
// existing — internal/handler/dashboard/handler.go:225-226
body := `<div class="card"><h1>Domain verified</h1>
<p>Save this production token now — it will not be shown again.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
```

This is the strongest engagement moment. The token-box with dashed accent border draws attention. Consider adding a "Copy token" button using the existing `[data-copy]` JS pattern from `internal/web/static/js/app.js`.

### After All Setup Complete — Next Steps

```go
// new code to add — shown when hasVerified && hasToken
b.WriteString(`<div class="card"><h2>Next steps</h2>
<ul><li><strong>Test your API key:</strong> <code>curl -H "Authorization: Bearer YOUR_TOKEN" ` +
    h.cfg.IDXAPIPublicURL + `/api/v1/properties?dataset=stellar&amp;_limit=1</code></li>
<li><strong>Integrate images:</strong> Use <code>/images/{media_key}</code> for cached MLS photos</li>
<li><strong>Add search:</strong> <code>POST /api/v1/search</code> with PostGIS or live MLS</li></ul></div>`)
```

## Anti-Patterns

- **AVOID** email drip campaigns for onboarding — the platform has no email sending infrastructure. Use in-dashboard guidance only.
- **AVOID** tracking "time on page" or click events — the server-rendered dashboard has no analytics SDK. Use database-level signals (audit logs, row counts) instead.
- **NEVER** show a "tour" overlay. The dashboard is a single page with three forms. A tour adds complexity without value.

## Related Skills

- See the **ux** skill for interaction design patterns
- See the **auth-api-token** skill for audit logging
- See the **frontend-design** skill for card and form styling