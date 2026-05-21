# Forms Reference

## Contents
- Form Patterns in idx-api
- API Token Form
- Domain Management Form
- Validation Rules
- Form Anti-Patterns

## Form Patterns in idx-api

The dashboard at `/dashboard` uses embedded static HTML (`internal/web/static/`). Forms submit via standard HTML or fetch calls to Fiber API routes. There is no frontend framework — plain HTML + minimal JS.

### API Token Form

| Field | Type | Validation | Error message |
|-------|------|------------|---------------|
| Token name | text | Required, unique per domain | "A token with this name already exists." |
| Scopes | checkboxes | At least one required | "Select at least one scope." |
| Domain | hidden/select | Must exist, user must own it | "Select a domain first." |

### WARNING: Client-Only Validation

**The Problem:**

```html
<!-- BAD — only validated in JS, server accepts anything -->
<input required oninvalid="this.setCustomValidity('Required')">
```

**Why This Breaks:** API consumers bypass the dashboard form entirely. If the server doesn't validate, bad data enters the system. Client validation is for speed and guidance, not trust.

**The Fix:**

```go
// GOOD — server validates every field
if req.Name == "" {
    return ErrorResponse(c, 400, "VALIDATION_ERROR", "Token name is required")
}
if len(req.Scopes) == 0 {
    return ErrorResponse(c, 400, "VALIDATION_ERROR", "At least one scope is required")
}
```

**When You Might Be Tempted:** "The form already checks this" — but API consumers don't use the form.

## Domain Management Form

| Field | Type | Validation | Error message |
|-------|------|------------|---------------|
| Domain | text | Required, valid hostname, unique | "Enter a valid domain (e.g., example.com)." |
| Allowed feeds | checkboxes | At least one | "Select at least one MLS feed." |

### WARNING: Duplicate Submission

**The Problem:** User double-clicks "Create Domain". Two requests fire. Second one returns 409 conflict but the user is confused.

**The Fix:** Disable the submit button on first click. Re-enable only on error.

```html
<!-- new code to add -->
<form onsubmit="this.querySelector('button').disabled = true">
```

## Validation Rules

Server validation lives in Fiber handlers. See the **fiber** skill for handler patterns.

| Rule | Implementation | Client hint |
|------|---------------|-------------|
| Required fields | `if field == ""` | `required` attribute |
| Unique constraints | DB query before insert | Check on blur (optional) |
| Scope allowlist | Validate against known scopes | List scopes in markup |
| Domain ownership | Verify domain belongs to user | Pre-select owned domains |

## Form Anti-Patterns

1. **No feedback on submit** — User clicks and nothing visible happens. Always show pending state immediately.
2. **Clearing form on error** — User typed a long domain name, validation fails, form clears. Preserve input on error.
3. **Generic "An error occurred"** — Every validation failure shows the same message. Be specific: "Domain already registered" vs "Invalid domain format".
4. **Hidden validation rules** — User discovers requirements by failing. Show format hints before submit: "e.g., example.com".