# Activation & Onboarding Reference

## Contents
- Setup Flow Architecture
- Progressive Setup Pattern
- Empty States
- Invitation Flow
- Anti-Patterns

## Setup Flow Architecture

The dashboard implements a three-step progressive setup: **domain registration → TXT verification → token creation**. Each step gates the next.

**Key files:**
- `internal/handler/dashboard/handler.go` — route handlers and setup logic
- `internal/web/layout.go` — `Page()` and `LoginPage()` template wrappers
- `internal/web/static/css/app.css` — badge and card styles

### Existing Setup Flow

```
User accepts invite → /invite/:token → account created → /login
→ /dashboard → no domains? → add domain form
→ domain added → badge-pending → verify TXT record
→ TXT verified → badge-verified → auto-generate production token
→ token displayed → copy button → integrate API
```

Routes from `dashboard/handler.go`:

| Route | Method | Purpose |
|-------|--------|---------|
| `/dashboard` | GET | Main dashboard with domains and tokens |
| `/dashboard/domains` | POST | Add new domain with MLS dataset |
| `/dashboard/domains/:id/verify-txt` | POST | Trigger TXT verification |
| `/dashboard/api-tokens` | POST | Create production token |
| `/dashboard/api-tokens/staging` | POST | Create staging token |
| `/dashboard/api-tokens/:id` | DELETE | Revoke token |
| `/invite/:token` | GET/POST | Invitation acceptance |

## Progressive Setup Pattern

Dashboard handlers build HTML server-side using `web.Page()`. The template conditionally shows setup steps based on what the user has completed:

```go
// existing pattern in dashboard/handler.go
func (h *Handler) Dashboard(c *fiber.Ctx) error {
    user := h.session.Get(c).Get("user")
    domains := h.domainRepo.ListForUser(user.ID)
    tokens := h.tokenRepo.ListForUser(user.ID)
    // Build cards for domains and tokens
    // Empty states shown when lists are empty
}
```

### Activation Criteria

A user is "activated" when they have:
1. A verified domain (`badge-verified`)
2. At least one active API token
3. First successful API call (tracked in `mls_proxy_audit_logs`)

## Empty States

The dashboard renders inline forms for empty states rather than placeholder messages:

```go
// existing pattern — no domains shows add-domain form
// existing pattern — no tokens shows create-token form
```

CSS supports these with the `.card`, `.form-stack`, and `.badge` classes from `app.css`.

### WARNING: Do Not Replace Inline Forms with Landing Pages

**The Problem:** Adding a full onboarding wizard or multi-page setup flow for a 3-step process.

**Why This Breaks:** The dashboard is invite-only B2B — users arrive knowing what they need. Wizards add friction without improving comprehension.

**The Fix:** Keep inline forms in cards. Show the next action and the reason to continue. Use `badge-pending` / `badge-verified` for status clarity.

## Invitation Flow

Admins create invitations via `/dashboard/invitations` (`requireAdmin` middleware). Invitations are time-limited tokens stored with SHA256 hashes.

```go
// existing pattern — invitation creation
func (h *Handler) CreateInvitation(c *fiber.Ctx) error {
    // Admin-only (requireAdmin middleware)
    // Generate token, hash with SHA256, store with TTL
    // Display invitation link to admin
}
```

Acceptance at `/invite/:token` creates the user account and redirects to `/login`.

## Anti-Patterns

### WARNING: Client-Side Setup Progress Tracking

**The Problem:**

```javascript
// BAD — tracking setup progress in localStorage
localStorage.setItem('setup-step', 'domain-verified')
```

**Why This Breaks:** Multi-device, multi-session users lose progress. Dashboard server-rendered — client state fights the architecture.

**The Fix:** Derive setup state from database: `COUNT(domains) > 0 AND domain.verified = true AND COUNT(tokens) > 0`.

### WARNING: Email-Based Onboarding Drips

**The Problem:** Adding email sequences for a 3-step setup.

**Why This Breaks:** The system has no transactional email infrastructure. Adding one for onboarding emails is over-engineering.

**The Fix:** Use the dashboard itself for guidance. If email nudges are needed later, add to the scheduler (`cmd/scheduler`) as a queued job type.

## Workflow Checklist

Copy this checklist for activation-related changes:
- [ ] Step 1: Identify which setup step the change affects (domain, verify, token)
- [ ] Step 2: Update dashboard handler to reflect new step or condition
- [ ] Step 3: Ensure badge states (`badge-pending`, `badge-verified`) match
- [ ] Step 4: Verify empty state shows correct inline form
- [ ] Step 5: Test with unverified domain and no tokens
- [ ] Step 6: Confirm audit log captures first-use event

See the **auth-api-token** skill for token abilities and domain verification middleware.
See the **ux** skill for dashboard layout and empty state patterns.