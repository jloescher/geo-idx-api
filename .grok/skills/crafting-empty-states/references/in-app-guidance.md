# In-App Guidance Reference

## Contents
- Guidance Surfaces
- Microcopy Patterns
- Error State Guidance
- Anti-Patterns

## Guidance Surfaces

All guidance in this project is server-rendered HTML. There is no tooltip library, no modal system, and no notification framework. Guidance must be inline within the dashboard HTML.

### Primary Guidance Locations

| Location | Type | How to Add |
|----------|------|------------|
| Dashboard `Setup` card | Empty state text | Add conditional block after domain loop |
| Domain list items | Inline status | Existing badges (`badge-verified`, `badge-pending`) |
| Add domain form | Placeholder text | Existing `placeholder="www.example.com"` |
| Token list items | Inline status | Existing badge with creation date |
| Verify TXT error | Error page | Currently plain text — wrap in `web.Page()` |
| Domain verified page | Success guidance | Existing one-time token reveal |

### Inline Help Text Pattern

Use `<p>` with muted styling for contextual guidance near forms:

```go
// new code to add — help text below the MLS dataset field
b.WriteString(`<p class="field-help">Use "stellar" for Bridge Data Output or "beaches" for Spark Platform.</p>`)
```

Corresponding CSS:

```css
/* new code to add — internal/web/static/css/app.css */
.field-help {
    color: var(--muted);
    font-size: 0.8rem;
    margin: 0.25rem 0 0;
}
```

## Microcopy Patterns

### Domain Verification Guidance

The DNS TXT verification step is the highest-friction point. Current messaging:

```go
// existing — handler.go:214
return c.Status(422).SendString(
    "TXT record not found. Publish the verification record at your DNS host, then try again.")
```

### WARNING: Unhelpful Verification Error

**The Problem:** "TXT record not found" tells the user WHAT failed but not WHERE to find the verification value. The `txt_verification_name` and `txt_verification_value` are stored in the database but never shown to the user on the error page.

**The Fix:**

```go
// new code to add — enhanced VerifyTXT error response
body := fmt.Sprintf(`<div class="card"><h1>Verification failed</h1>
<p>DNS TXT record not found. Publish this record at your DNS host:</p>
<table class="dns-table"><tr><th>Name</th><td><code>%s</code></td></tr>
<tr><th>Value</th><td><code>%s</code></td></tr></table>
<p>DNS changes may take a few minutes to propagate.</p>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`,
    web.Esc(txtHost), web.Esc(txtVal))
return c.Status(422).Type("html").SendString(web.Page("Verification failed", body))
```

### Form Placeholder Guidance

```go
// existing — handler.go:159
<label>Token name <input name="name" type="text" placeholder="Production" required></label>
// existing — handler.go:165
<label>Hostname <input name="domain_slug" type="text" placeholder="www.example.com" required></label>
```

Placeholders serve as inline guidance. Keep them brief and realistic.

## Error State Guidance

### DNS Lookup Failure (Upstream Error)

```go
// existing — handler.go:208-209
if err != nil {
    return fiber.NewError(fiber.StatusBadGateway, "DNS lookup failed")
}
```

This returns a Fiber default error page. Wrap it to provide actionable guidance:

```go
// new code to add
body := `<div class="card"><h1>DNS lookup unavailable</h1>
<p>Could not reach DNS servers to verify your TXT record. This is usually temporary.</p>
<p><a class="btn btn-primary" href="/dashboard">Try again from the dashboard</a></p></div>`
return c.Status(fiber.StatusBadGateway).Type("html").SendString(web.Page("DNS Error", body))
```

### Staging Token Already Exists

```go
// existing — handler.go:237
return c.Status(409).SendString("Staging token already exists")
```

This tells the user WHAT happened but not what to do. Consider: "You already have a staging token. Find it in the API keys list on your dashboard."

## Anti-Patterns

- **NEVER** add a JavaScript tooltip library. Use inline `<p>` elements with `--muted` color. The dashboard has minimal JS (`app.js` is ~20 lines for clipboard copy only).
- **NEVER** use `alert()` for guidance. All messages should be in the HTML response body, wrapped in `web.Page()`.
- **AVOID** adding `<script>` blocks to handler-generated HTML for interactive guidance. Keep it server-rendered and static.
- **AVOID** over-explaining. The audience is developers setting up an IDX proxy — they know what DNS is.

## Related Skills

- See the **ux** skill for error state patterns
- See the **frontend-design** skill for typography and spacing
- See the **fiber** skill for error handling middleware