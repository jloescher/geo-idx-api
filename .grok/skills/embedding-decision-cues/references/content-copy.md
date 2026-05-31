# Content Copy Reference

## Contents
- Copy Location Map
- Voice and Tone
- Writing Decision-Cue Copy
- Anti-Patterns

## Copy Location Map

All user-facing copy in Quantyra IDX lives in Go string literals within handler files — no templates, no CMS, no i18n.

| Copy surface | File | Line context |
|-------------|------|-------------|
| Landing hero title + description | `internal/handler/marketing/handler.go` | `Home()` method |
| Login heading + subtitle | `internal/web/layout.go` | `LoginPage()` |
| Dashboard setup description | `internal/handler/dashboard/handler.go` | `Dashboard()` |
| Domain verification success | `internal/handler/dashboard/handler.go` | `VerifyTXT()` |
| DNS verification failure | `internal/handler/dashboard/handler.go` | `VerifyTXT()` error path |
| Invitation creation | `internal/handler/dashboard/handler.go` | `CreateInvitation()` |
| Staging token conflict | `internal/handler/dashboard/handler.go` | `CreateStagingToken()` |
| Auth error messages | `internal/api/middleware/` | domain/token auth middleware |
| GIS error messages | `internal/service/gis/` | "MLS code not supported", "bbox span exceeds limit" |
| Nav links | `internal/web/layout.go` | `Page()` header |

## Voice and Tone

This is a **developer infrastructure** product. Copy should be:

- **Direct** — no exclamation marks, no hype words
- **Technical** — use exact terminology (DNS, TXT record, API token, MLS dataset)
- **Neutral** — avoid emotional language; trust comes from precision
- **Actionable** — every error message should tell the user what to do next

### Tone Examples

**GOOD:** "TXT record not found. Publish the verification record at your DNS host, then try again."
- States the problem, gives exact next action.

**BAD:** "Oops! We couldn't verify your domain. Please try again later!"
- Vague, patronizing, no actionable next step.

**GOOD:** "Save this production token now — it will not be shown again."
- Clear consequence, urgent but factual.

**BAD:** "Hurry! Copy your token before it disappears forever!"
- Hyperbolic, erodes trust.

## Writing Decision-Cue Copy

### Loss Aversion Pattern

Used when a valuable item (token, invite link) is displayed for the last time.

```
[Value statement]. [Consequence of inaction].
```

Existing: `"Save this production token now — it will not be shown again."`

### Progress Motivation Pattern

Used in multi-step flows to reduce drop-off.

```
[Current step indicator]. [What's remaining]. [Value at completion].
```

The current dashboard lacks this — the setup flow is implicit in the card order (domains → API keys → add domain).

### Scarcity Pattern

Used for limited-availability items.

```
[Item description] ([Availability constraint]).
```

Existing: `"Share this link (shown once):"` in `CreateInvitation()`

### Error Recovery Pattern

Every error should include a next action:

```go
// Existing good example — DNS verification
c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")

// BAD — no recovery path
c.Status(422).SendString("TXT record not found.")
```

## Anti-Patterns

### WARNING: Marketing Copy in Error Responses

**The Problem:** API error responses (4xx/5xx) are consumed by developer code, not humans. Adding marketing copy ("Upgrade to Pro!") in API errors breaks programmatic error handling.

**Why This Breaks:** Client code parses error strings. Marketing copy in `SendString()` responses will leak into logs and error handlers, confusing developers.

**The Fix:** Keep API responses factual. Decision cues belong in the dashboard HTML, not JSON/plaintext API responses. The GIS teaser tier correctly uses HTTP headers (`X-GIS-Teaser`) rather than modifying the response body.

### WARNING: Inconsistent Terminology

**The Problem:** The codebase uses "API keys" in the dashboard heading but "token" everywhere else (token-box, CreateToken, RevokeToken).

**The Fix:** Pick one term. "API token" is more accurate for what these are (Bearer tokens, not key pairs). The dashboard heading should say "API tokens" to match the code and reduce cognitive friction.

1. Audit all user-facing strings for "API key" vs "API token"
2. Standardize to "API token" across dashboard and layout
3. Validate: `go build ./cmd/api`
4. Search codebase: `grep -r "API key" internal/handler/ internal/web/`

See the **frontend-design** skill for CSS and layout patterns used in copy presentation.