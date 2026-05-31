# Activation & Onboarding Reference

## Contents
- Onboarding Flow
- Empty State Patterns
- Setup Checklist
- Anti-Patterns

## Onboarding Flow

The activation path is: admin sends invite → user registers at `/invite/:token` → user logs in at `/login` → user lands on `/dashboard` with empty domain and token lists.

The critical gap: after login, the dashboard shows "Setup" with empty `<ul>` lists. New users see no guidance on what to do first.

### Current Invite Acceptance

```go
// internal/handler/dashboard/handler.go:282-288
func (h *Handler) AcceptInvitation(c *fiber.Ctx) error {
    err := h.invitations.Accept(c.Context(), c.Params("token"), c.FormValue("name"), c.FormValue("password"))
    if err != nil {
        return c.Status(400).SendString(err.Error())
    }
    return c.Redirect("/login")
}
```

### WARNING: Silent Registration Error

**The Problem:**

```go
// BAD — plain text error, no page chrome, no link back
return c.Status(400).SendString(err.Error())
```

**Why This Breaks:** After a failed registration (expired token, weak password), the user sees a bare error string on a white page with no navigation. They must manually navigate back.

**The Fix:**

```go
// new code to add
body := `<div class="card"><h1>Registration failed</h1><p>` + web.Esc(err.Error()) + `</p>
<p><a class="btn btn-primary" href="/login">Back to login</a></p></div>`
return c.Status(400).Type("html").SendString(web.Page("Error", body))
```

## Empty State Patterns

### Domain List Empty State

```go
// new code to add — after domain list loop in Dashboard()
if domainCount == 0 {
    b.WriteString(`<div class="empty-state">
    <p><strong>No domains registered.</strong> Add your first domain below to start using the MLS proxy.</p>
    <ol class="setup-steps">
    <li>Add a domain hostname</li>
    <li>Publish the DNS TXT verification record</li>
    <li>Click Verify TXT — your production API key appears</li>
    </ol></div>`)
}
```

### API Token List Empty State

```go
// new code to add — after token list loop in Dashboard()
if tokenCount == 0 {
    b.WriteString(`<p class="muted">No API keys yet. Verify a domain to get your production token, or create a staging token below.</p>`)
}
```

## Setup Checklist

Track activation with a three-step progress indicator derived from existing database state:

| Step | Condition | Check |
|------|-----------|-------|
| 1. Add domain | `COUNT(domains) > 0` | `SELECT COUNT(*) FROM domains WHERE user_id = $1` |
| 2. Verify domain | `verification_status = 'verified'` for any domain | Same query, filter status |
| 3. Get API key | `COUNT(personal_access_tokens) > 0` | `SELECT COUNT(*) FROM personal_access_tokens WHERE tokenable_id = $1` |

### Rendering the Checklist

```go
// new code to add — at the top of Dashboard() body, before domain list
hasDomain := false
hasVerified := false
hasToken := false
// ... query checks ...

b.WriteString(`<div class="setup-progress"><h2>Getting started</h2><ol>`)
stepClass := func(done bool) string {
    if done { return "step-done" }
    return "step-pending"
}
b.WriteString(`<li class="` + stepClass(hasDomain) + `">Add a domain</li>`)
b.WriteString(`<li class="` + stepClass(hasVerified) + `">Verify DNS ownership</li>`)
b.WriteString(`<li class="` + stepClass(hasToken) + `">Save your API key</li>`)
b.WriteString(`</ol></div>`)
```

## Anti-Patterns

- **NEVER** add client-side state management for onboarding progress. The dashboard is server-rendered; derive state from PostgreSQL queries.
- **NEVER** use `localStorage` or cookies for "dismissed" flags. If you need persistent dismissal, add a column to `users` or use the session store.
- **AVOID** multi-page wizards. The dashboard already has all forms on one page — enhance inline guidance instead of splitting into separate routes.
- **AVOID** auto-redirecting to a "welcome" page. The dashboard IS the welcome page. Add the onboarding content directly into the `Dashboard()` handler.

## Related Skills

- See the **fiber** skill for route registration patterns
- See the **auth-api-token** skill for token and domain verification flows
- See the **postgres** skill for query patterns