# In-App Guidance Reference

## Contents
- Guidance Surfaces
- Copy-to-Clipboard Pattern
- DNS Verification Guidance
- Token Security Warnings
- Conditional Admin Guidance
- Anti-Patterns

## Guidance Surfaces

The Quantyra IDX dashboard is server-rendered HTML with no SPA framework. All in-app guidance is embedded in Go handler string templates (`internal/handler/dashboard/handler.go`) and styled with `internal/web/static/css/app.css`. There is no tooltip library, no tour framework, and no client-side state management.

This means every guidance change requires:
1. Edit the Go handler that builds the HTML string
2. Use existing CSS classes (`.card`, `.badge`, `.token-box`, `.btn`)
3. Rebuild and redeploy

## Copy-to-Clipboard Pattern

The existing `internal/web/static/js/app.js` provides copy-to-clipboard functionality. The token reveal page (`handler.go:225`) uses the `.token-box` class:

```go
// existing — internal/handler/dashboard/handler.go:225
body := `<div class="card"><h1>Domain verified</h1>
<p>Save this production token now — it will not be shown again.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
```

This is the strongest in-app guidance moment: the token is shown exactly once, with a clear warning. The `.token-box` styling (monospace font, dashed blue border) makes it visually distinct from other content.

### Adding Copy Button

```html
<!-- new code to add — enhance token-box with copy button -->
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<button class="btn btn-sm btn-secondary" onclick="navigator.clipboard.writeText(document.getElementById('token').textContent);this.textContent='Copied'">Copy token</button>
```

Keep it inline — no additional JS file needed. The existing `app.js` pattern already uses this approach.

## DNS Verification Guidance

The DNS TXT verification flow (`handler.go:196`) returns a plain text error on failure:

```go
// existing — internal/handler/dashboard/handler.go:214
return c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")
```

This is insufficient guidance — the user does not know what TXT record to publish. The `StoreDomain` handler (`handler.go:180`) generates `_quantyra-verify.{domain}` with a random hex value but does not display it to the user.

### Fix: Show TXT Record After Domain Registration

```go
// new code to add — after StoreDomain insert, redirect to a confirmation page
body := `<div class="card"><h1>Domain registered</h1>
<p>Add this TXT record to your DNS configuration:</p>
<div class="token-box">
<strong>Name:</strong> _quantyra-verify.` + web.Esc(slug) + `<br>
<strong>Value:</strong> ` + web.Esc(val) + `
</div>
<p>DNS propagation may take a few minutes. Return to the dashboard and click <strong>Verify TXT</strong> when ready.</p>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
return c.Type("html").SendString(web.Page("Domain Registered", body))
```

This uses the existing `.token-box` and `.btn-primary` classes — no new CSS required.

## Token Security Warnings

The existing code shows the production token once after verification. Guidance improvements should not change this behavior — showing the token again would weaken security.

### Staging Token Guidance

The staging token response (`handler.go:243`) is plain text:

```go
// existing — internal/handler/dashboard/handler.go:243
return c.SendString("Staging token: " + plain)
```

Wrap it in the page layout so it matches the production token experience:

```go
// new code to add — consistent staging token reveal
body := `<div class="card"><h1>Staging token</h1>
<p>Use this token for development and testing against the staging environment.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
return c.Type("html").SendString(web.Page("Staging Token", body))
```

## Conditional Admin Guidance

Admin-only UI is already conditionally rendered (`handler.go:170`):

```go
// existing — internal/handler/dashboard/handler.go:170
if isAdmin {
    b.WriteString(`<div class="card"><h2>Invite user</h2>...`)
}
```

Follow this pattern for any role-dependent guidance. Query `is_admin` once per dashboard load (already done) and conditionally append HTML strings to the builder.

### Adding Contextual Help for Admins

```html
<!-- new code to add — admin guidance card after invite form -->
<div class="card"><h2>Invite user</h2>
<p>Invited users receive a one-time registration link. Links expire after the configured TTL.</p>
<form method="post" action="/dashboard/invitations" class="inline-form">...</form>
</div>
```

## Anti-Patterns

### WARNING: Client-Side Guidance Frameworks

**The Problem:** Adding a JS tour library (Intro.js, Shepherd, Driver.js) to the server-rendered dashboard.

**Why This Breaks:**
1. The dashboard is Go template strings, not a component tree — tour targeting is fragile
2. Adds a heavy JS dependency for a page with ~4 interactive elements
3. Tour state fights with server-driven state (domain verified → tour step should disappear)

**The Fix:** Use conditional Go template blocks. Show/hide guidance cards based on the same PostgreSQL queries that power the dashboard. No JavaScript required beyond the existing copy-to-clipboard.

### WARNING: Generic Help Text Over Specific Instructions

**The Problem:** Adding tooltip text like "Click here to manage your domains" instead of task-specific guidance.

**Why This Breaks:** The dashboard has a narrow, well-defined purpose (domain + token management). Generic help text adds noise. Users need specific instructions like "Add this TXT record to your DNS" — not "Manage your settings."

**The Fix:** Every guidance string should contain an action the user can take right now, not a description of what the page does.

## Cross-References

- See the **ux** skill for state matrix patterns and accessibility checks
- See the **frontend-design** skill for CSS patterns and the dark theme design system
- See the **auth-api-token** skill for token lifecycle and DNS verification flows
- See the **fiber** skill for handler patterns and HTML response construction