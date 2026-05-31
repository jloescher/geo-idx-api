# Conversion Optimization Reference

## Contents
- Setup Funnel Analysis
- Decision Cue Placement
- Anti-Patterns
- Dashboard Flow Optimization

## Setup Funnel Analysis

Quantyra IDX has a three-step conversion funnel embedded in `internal/handler/dashboard/handler.go`:

1. **Add domain** — `StoreDomain()` inserts a `pending` domain with TXT verification record
2. **Verify DNS** — `VerifyTXT()` checks DNS, flips status to `verified`, issues production token
3. **Use API token** — Token is displayed once with loss-aversion copy

The funnel has one critical drop-off: **DNS verification**. Users must leave the dashboard, publish a TXT record at their DNS host, then return. The current copy ("TXT record not found. Publish the verification record at your DNS host, then try again.") is functional but lacks urgency or progress reinforcement.

### Conversion Touch Points

| Touch point | File | Current cue | Behavioral lever |
|-------------|------|-------------|-----------------|
| Landing hero | `internal/handler/marketing/handler.go` | "Open dashboard" CTA | Clarity / action |
| Login page | `internal/web/layout.go` | "Access your MLS domains and API keys" | Value proposition |
| Dashboard header | `Dashboard()` | "Register domains, verify DNS, and manage API keys" | Task framing |
| Domain badges | `Dashboard()` | `badge-verified` / `badge-pending` | Status / progress |
| Token display | `VerifyTXT()` | "will not be shown again" | Loss aversion |
| Invite link | `CreateInvitation()` | "shown once" | Scarcity |
| Staging token block | `CreateStagingToken()` | "Staging token already exists" (409) | Exclusivity |

## Decision Cue Placement

### DO: Place loss-aversion cues at commitment points

```go
// Existing pattern in VerifyTXT — production token issuance
// "Save this production token now — it will not be shown again."
// This is effective because it creates urgency at the moment of highest commitment
```

### DON'T: Add urgency cues to low-commitment actions

```go
// AVOID — do not add artificial urgency to staging token creation
// Staging tokens are low-risk; heavy urgency messaging erodes trust
// The current 409 "Staging token already exists" is appropriately neutral
```

### DO: Use status badges for progress visibility

The existing badge system in `app.css` already supports this:

```css
/* Existing — use these for progress signaling */
.badge-verified {
  background: rgba(16, 185, 129, 0.15);
  color: var(--success); /* green */
}
.badge-pending {
  background: rgba(245, 158, 11, 0.15);
  color: var(--warning); /* amber */
}
```

### WARNING: Don't Overload the Hero

**The Problem:** The current hero (`marketing/handler.go`) has no value proposition beyond "MLS proxy, image delivery, and developer setup." For a B2B API, the hero must communicate **what makes this different** in one sentence.

**Why This Breaks:** Developers scanning the page in 3 seconds won't understand the value. Generic descriptions like "developer setup" don't differentiate from competitors.

**The Fix:**

```go
// new code to add — replace hero description with benefit-focused copy
`<p>One API for Bridge and Spark MLS feeds — proxy, PostGIS search, and image delivery with zero infrastructure management.</p>`
```

**When You Might Be Tempted:** When adding feature lists, pricing comparisons, or testimonials to the hero. Don't — this is a developer tool, not a consumer product. Lead with the technical benefit.

## Anti-Patterns

### WARNING: Re-Displaying One-Time Tokens

**The Problem:**

```go
// BAD — storing plain tokens for re-display
plain, _ := h.tokens.Create(c.Context(), uid, "Production", abilities)
// ... store plain text somewhere for later retrieval
```

**Why This Breaks:** The "will not be shown again" pattern creates genuine scarcity. If tokens can be re-retrieved, the urgency cue becomes a lie, and users learn to ignore it.

**The Fix:** Only the SHA-256 hash is stored in `personal_access_tokens`. The plain token is displayed exactly once in the `VerifyTXT` response and never again.

### WARNING: Open Registration Breaks Exclusivity

**The Problem:** The invite-only system in `CreateInvitation()` creates perceived exclusivity. Adding an open signup route would eliminate this.

**The Fix:** All registration flows through `/invite/:token`. The `requireAdmin` middleware gates invitation creation.

## Dashboard Flow Optimization

Copy this checklist and track progress:

- [ ] Audit all user-facing strings in `internal/handler/dashboard/handler.go`
- [ ] Verify loss-aversion copy is accurate (tokens truly shown once)
- [ ] Check GIS teaser response headers communicate upgrade path
- [ ] Confirm badge colors convey correct status semantics
- [ ] Test invitation flow end-to-end for one-time-display truthfulness

1. Make changes to dashboard copy
2. Validate: `go build ./cmd/api` and `go test ./internal/handler/dashboard/...`
3. If build fails, fix and repeat step 2
4. Manually verify copy renders correctly in browser

See the **auth-api-token** skill for token creation and revocation patterns.
See the **ux** skill for dashboard interaction and accessibility guidance.