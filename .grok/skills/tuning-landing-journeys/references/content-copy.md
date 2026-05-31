# Content Copy Reference

## Contents
- Where copy lives
- Headline patterns
- CTA copy guidelines
- Microcopy for forms
- Error message copy
- Anti-patterns

## Where Copy Lives

All user-facing copy is embedded as Go string literals in handler functions. There is no CMS, no i18n, no template engine.

| Copy location | File | Function |
|---------------|------|----------|
| Landing hero title, subtitle, CTAs | `marketing/handler.go` | `Home()` |
| Login heading, subtitle | `web/layout.go` | `LoginPage()` |
| Dashboard card headings, descriptions | `dashboard/handler.go` | `Dashboard()` |
| Verification success copy | `dashboard/handler.go` | `VerifyTXT()` |
| Token creation copy | `dashboard/handler.go` | `CreateToken()` |
| Invitation copy | `dashboard/handler.go` | `CreateInvitation()` |
| Header nav labels | `web/layout.go` | `Page()` |
| Error messages | `dashboard/handler.go` | various handlers |

## Headline Patterns

Current headlines use product name or task name:

| Page | Headline | Pattern |
|------|----------|---------|
| Landing | "Quantyra IDX" | Product name (weak — describes the product, not the value) |
| Login | "Sign in" | Action-oriented (good) |
| Dashboard | "Setup" | Task-oriented (good) |
| Invite | "Sign in" | Reused from login (confusing — user is registering, not signing in) |

### DO: Lead with outcome, not product name

```html
<!-- BAD — product name tells returning users nothing new -->
<h1>Quantyra IDX</h1>
<p>MLS proxy, image delivery, and developer setup for your IDX sites.</p>

<!-- GOOD — outcome-focused for the ICP (real estate developers) -->
<h1>Ship MLS listings to your site in minutes</h1>
<p>Proxy, search, and image delivery for Bridge and Spark feeds — no infrastructure to manage.</p>
```

### DO: Differentiate registration from login

The invite form at `dashboard/handler.go:273` uses `web.LoginPage()` which wraps with "Sign in" heading. Invitees are creating accounts, not signing in:

```go
// new code to add — dedicated invite layout or parameterized LoginPage
// Option: pass custom heading into LoginPage
func LoginPage(heading, body string) string { ... }
```

## CTA Copy Guidelines

Current CTAs:

| Button text | Context | Assessment |
|-------------|---------|------------|
| "Open dashboard" | Landing hero | Good — action + destination |
| "Sign in" | Landing hero | Redundant with header nav |
| "Sign in" | Login form button | Correct |
| "Create account" | Invite form button | Correct |
| "Create token" | Dashboard form | Functional but could be clearer |
| "Add domain" | Dashboard form | Correct |
| "Send invitation" | Dashboard form (admin) | Correct |
| "Verify TXT" | Dashboard domain list | Correct |
| "Revoke" | Dashboard token list | Correct |

### DO: Use verb + noun pattern

Every CTA should answer "what happens when I click?":

- "Create production token" vs "Create token"
- "Add your domain" vs "Add domain"
- "Verify DNS record" vs "Verify TXT"

### DON'T: Use generic labels

```html
<!-- BAD -->
<button type="submit" class="btn btn-primary">Submit</button>

<!-- GOOD -->
<button type="submit" class="btn btn-primary">Add domain</button>
```

## Microcopy for Forms

### Input labels

Current labels are minimal: "Email", "Password", "Hostname", "MLS dataset", "Token name". This works for a technical audience.

### Placeholder text

Only two placeholders exist:

| Field | Placeholder | Assessment |
|-------|-------------|------------|
| `domain_slug` | "www.example.com" | Good — shows expected format |
| `name` (token) | "Production" | Good — suggests convention |

### DO: Add placeholder for MLS dataset

```html
<!-- new code to add -->
<input name="mls_dataset" type="text" value="stellar" placeholder="stellar or beaches">
```

## Error Message Copy

Error messages are returned as plain text, not rendered in the page layout:

| Handler | Status | Message | Assessment |
|---------|--------|---------|------------|
| `Login` | 401 | "Invalid credentials" | Good — no information leakage |
| `StoreDomain` | 400 | Raw `err.Error()` | BAD — leaks SQL internals |
| `VerifyTXT` | 422 | "TXT record not found..." | Good — actionable guidance |
| `CreateInvitation` | 400 | Raw `err.Error()` | BAD — leaks internals |
| `AcceptInvitation` | 400 | Raw `err.Error()` | BAD — leaks internals |

### WARNING: Raw error exposure

```go
// BAD — database constraint names visible to users
return c.Status(400).SendString(err.Error())

// GOOD — map known errors to user messages
switch {
case strings.Contains(err.Error(), "duplicate"):
    return c.Status(409).SendString("This domain is already registered.")
default:
    return c.Status(400).SendString("Could not process your request.")
}
```

## Anti-Patterns

### WARNING: Copy changes require Go recompilation

All copy lives in Go source. For copy-only changes, the edit → compile → deploy cycle is unavoidable in this architecture. Batch copy changes together to minimize deploy cycles.

### WARNING: No copy versioning or A/B test surface

The string-literal approach has no mechanism for A/B testing headlines or CTAs. Adding experiment support would require a database-backed copy store or feature flag system. See the **measurement-testing** reference.