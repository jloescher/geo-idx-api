# Content Copy Reference

## Contents
- Copy Surfaces
- Hero Copy Pattern
- Dashboard Microcopy
- Error State Copy
- Anti-Patterns

## Copy Surfaces

All copy is embedded in Go handler functions as inline HTML strings. There is no CMS, no i18n, and no template engine beyond `web.Page()` and `web.LoginPage()`.

| Surface | File | Format |
|---|---|---|
| Landing hero | `internal/handler/marketing/handler.go` | Inline HTML in Go string |
| Dashboard body | `internal/handler/dashboard/handler.go` | `strings.Builder` HTML |
| Login page | `internal/web/layout.go` | Template string |
| Token reveal | `dashboard/handler.go:VerifyTXT` | Inline HTML |
| Invitation page | `dashboard/handler.go:CreateInvitation` | Inline HTML |
| DNS verification error | `dashboard/handler.go:VerifyTXT` | Plain text string |

## Hero Copy Pattern

```go
// Existing — marketing/handler.go
<h1>Quantyra IDX</h1>
<p>MLS proxy, image delivery, and developer setup for your IDX sites.</p>
```

This is a feature description, not a value proposition. For B2B developer tools, the hero should lead with the **outcome** the buyer achieves, not the mechanism.

**Hypothesis:** Changing the tagline from feature-description to outcome-oriented copy increases dashboard click-through:

```go
// new code to add — alternative hero copy for testing
<h1>Ship IDX listings in minutes, not weeks</h1>
<p>One API for MLS data, photos, and parcel geometry — no vendor lock-in.</p>
```

### How to Test Copy Changes

Since there is no A/B framework, use a query parameter or cookie:

```go
// new code to add — simple split test in marketing handler
func (h *Handler) Home(c *fiber.Ctx) error {
    variant := c.Query("v", "a")
    headline := "Quantyra IDX"
    tagline := "MLS proxy, image delivery, and developer setup for your IDX sites."
    if variant == "b" {
        headline = "Ship IDX listings in minutes, not weeks"
        tagline = "One API for MLS data, photos, and parcel geometry — no vendor lock-in."
    }
    // ... render with headline/tagline variables
}
```

Route traffic 50/50 via Cloudflare Workers or link parameters.

## Dashboard Microcopy

### Setup Section

```go
// Existing — dashboard/handler.go
<h1>Setup</h1>
<p>Register domains, verify DNS, and manage API keys.</p>
```

This is functional but does not convey progress or urgency. Consider adding a step indicator:

```go
// new code to add — progress-aware copy
<p>Step 1 of 3: Register your first domain to get a production API key.</p>
```

### Token Reveal

```go
// Existing — VerifyTXT success
<h1>Domain verified</h1>
<p>Save this production token now — it will not be shown again.</p>
```

This is strong copy. The urgency ("will not be shown again") is accurate and drives immediate action. Keep this.

### Invitation Copy

```go
// Existing — CreateInvitation
<h1>Invitation created</h1>
<p>Share this link (shown once):</p>
```

The "shown once" pattern mirrors the token reveal and is effective. Consider adding context about what the invitee will see:

```go
// new code to add — richer invitation copy
<p>Share this link — your colleague will set a password and get immediate staging access.</p>
```

## Error State Copy

### DNS Verification Failure

```go
// Existing — VerifyTXT failure
return c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")
```

This is actionable — it tells the user exactly what to do next. This is good error copy.

### Staging Token Conflict

```go
// Existing — CreateStagingToken
return c.Status(409).SendString("Staging token already exists")
```

This is not actionable. The user does not know where to find their existing staging token.

**Fix:**

```go
// new code to add — actionable error copy
return c.Status(409).SendString("Staging token already exists. Find it in your dashboard under API keys.")
```

## Anti-Patterns

### WARNING: Copy in Multiple Languages Without i18n

**The Problem:**
Hardcoding microcopy changes across 10+ inline HTML strings with no extraction layer.

**Why This Breaks:**
- Copy changes require Go code changes and redeployment
- No way to review copy without reading Go source
- Cannot A/B test without code changes

**The Fix:**
For now, keep all copy in handlers but group copy changes into dedicated commits. If copy iteration accelerates, extract into a `copy` package with string constants per surface.

### WARNING: Marketing Copy in API Error Responses

**The Problem:**
Adding promotional messaging to API error bodies (e.g., "Upgrade to get full GIS access").

**Why This Breaks:**
- API consumers parse error responses programmatically
- Promotional text breaks error parsing
- `Content-Type` is `text/plain` for most dashboard errors, not `text/html`

**The Fix:**
Keep upgrade prompts in HTTP headers (`X-Upgrade-Available: true`) or in the GIS GeoJSON `properties` field, not in error body text. See the **geospatial** skill for GIS response patterns.