# Content Copy Reference

## Contents
- Copy surfaces in the codebase
- Tone and voice guidelines
- Writing effective lifecycle messages
- Anti-patterns

## Copy Surfaces in the Codebase

| Surface | File | Format |
|---|---|---|
| Landing page hero | `internal/handler/marketing/handler.go:19` | Inline HTML string |
| Dashboard heading | `internal/handler/dashboard/handler.go:124` | Inline HTML string |
| Login page | `internal/web/layout.go:42` | Template constant |
| Domain verified confirmation | `internal/handler/dashboard/handler.go:225` | Inline HTML string |
| Invitation created confirmation | `internal/handler/dashboard/handler.go:269` | Inline HTML string |
| TXT verification error | `internal/handler/dashboard/handler.go:214` | Inline string |
| Invalid credentials | `internal/handler/dashboard/handler.go:98` | Inline string |

All copy is embedded in Go source as string literals. There are no external template files, markdown content files, or CMS surfaces.

## Tone and Voice Guidelines

The platform is a **B2B developer tool**. Copy should be:

- **Direct** — state what happened and what to do next
- **Technical but approachable** — users are developers integrating MLS APIs
- **Action-oriented** — every message should end with a clear next step

### DO: Lead with the outcome, follow with the action

```go
// internal/handler/dashboard/handler.go:225 — GOOD
// "Domain verified" (outcome) then "Save this production token now" (action)
body := `<div class="card"><h1>Domain verified</h1><p>Save this production token now — it will not be shown again.</p>...`
```

### DON'T: Use vague or passive language

```go
// BAD — no clear outcome or next step
body := `<div class="card"><h1>Processing complete</h1><p>Your request has been handled.</p>`
```

## Writing Effective Lifecycle Messages

### Per-stage copy guidelines

| Stage | Goal | Tone | Key message |
|---|---|---|---|
| Invitation | Drive registration | Welcome + technical | "You've been invited to Quantyra IDX. Accept to set up your MLS domains and API keys." |
| Registration | Complete signup | Minimal friction | Name + password only. No marketing copy needed. |
| Domain verification | Complete DNS setup | Instructional | "Publish the TXT record, then verify." Include the record value prominently. |
| Token creation | Secure the token | Urgent | "Save now — shown once." This pattern already works well. |
| First API call | Activate usage | Guiding | "Try your first request." Include a curl example with their dataset. |

### Email subject lines (when email is implemented)

Subject lines for this B2B developer audience should:
- Lead with the action required, not the brand name
- Be specific: "Your Quantyra API key is ready" not "Update"
- Include the domain or dataset when relevant: "Verify www.example.com for Quantyra IDX"

### In-dashboard guidance copy

The dashboard currently says: "Register domains, verify DNS, and manage API keys." This is functional but does not guide the user through the sequence. For a step-based approach, consider numbered steps with completed/pending state — see the **ux** skill for empty-state and guidance patterns.

## Anti-patterns

### WARNING: Concatenating user input into HTML

```go
// BAD — XSS if slug contains malicious HTML
b.WriteString("<li><strong>" + slug + "</strong>")
```

**Why This Breaks:** User-provided domain slugs could contain `<script>` tags or event handlers. Even though domain slugs are typically safe, the pattern is wrong.

**The Fix:**

```go
// GOOD — use the existing Esc helper
// internal/handler/dashboard/handler.go:133 — correct pattern already used
b.WriteString(`<li><strong>` + web.Esc(slug) + `</strong>...`
```

### WARNING: Generic error messages

```go
// internal/handler/dashboard/handler.go:98
return c.Status(401).SendString("Invalid credentials")
```

This is correct for auth — it avoids information leakage. But for non-auth errors (like domain verification), be specific about what went wrong and how to fix it. The TXT verification error at line 214 is a good model:

```go
// GOOD — specific, actionable
return c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")
```

## Related Skills

- **frontend-design** — CSS badge, card, and form styling for copy surfaces
- **ux** — empty states, step indicators, and in-dashboard guidance
- **auth-api-token** — token display and security copy patterns