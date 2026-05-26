# Growth Engineering Reference

## Contents
- Viral Loop Architecture
- API-Driven Growth Surfaces
- GIS Teaser Engineering
- Automation Opportunities
- Anti-Patterns

## Viral Loop Architecture

The invite system in `internal/handler/dashboard/handler.go` is the primary viral loop:

```
Admin → CreateInvitation → generates /invite/:token URL
  → invitee visits URL → InviteRegisterForm → AcceptInvitation
  → new user → /dashboard → StoreDomain → VerifyTXT → production token
```

### Loop Mechanics

```go
// Existing — invitation creation (admin-only)
func (h *Handler) CreateInvitation(c *fiber.Ctx) error {
    plain, err := h.invitations.Create(c.Context(), uid, c.FormValue("email"))
    // URL shown once
}

// Existing — invitation acceptance
func (h *Handler) AcceptInvitation(c *fiber.Ctx) error {
    err := h.invitations.Accept(c.Context(), c.Params("token"), c.FormValue("name"), c.FormValue("password"))
    return c.Redirect("/login")
}
```

**Current K-factor: likely < 1** because only admins can invite, and there is no in-product prompt to invite after activation. The loop relies on manual admin initiative.

**Hypothesis:** Prompting newly-verified users to invite their team members increases invites per user by 2x. The prompt should appear on the `VerifyTXT` success page, which already has high engagement (token reveal).

### Engineering the Prompt

```go
// new code to add — invite prompt after domain verification
body := `<div class="card"><h1>Domain verified</h1>
<p>Save this production token now — it will not be shown again.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<div class="card" style="margin-top:1rem">
<p>Invite a teammate to manage this domain?</p>
<form method="post" action="/dashboard/invitations" class="inline-form">
<label>Email <input name="email" type="email" placeholder="colleague@company.com"></label>
<button type="submit" class="btn btn-secondary">Send invitation</button>
</form></div>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
```

## API-Driven Growth Surfaces

### Response Headers as Growth Channels

Every API response can include headers that drive upgrades without breaking the contract:

```go
// new code to add — growth headers in GIS handler
if truncated {
    c.Set("X-Teaser-Truncated", "true")
    c.Set("X-Teaser-Max-Features", strconv.Itoa(maxFeatures))
    c.Set("X-Upgrade-Info", "https://idx.quantyralabs.cc/dashboard")
}
```

Client applications can read these headers and display contextual upgrade prompts.

### Comps API as Lead Generator

The Comps engine supports BPO, home value, and investor modes (see `docs/comps-api.md`). Each mode targets a different persona:

| Mode | Persona | Growth potential |
|---|---|---|
| BPO | Real estate agents | High volume, per-listing usage |
| Home value | Homeowners / buyers | Self-service, lower value per call |
| Investor | Real estate investors | Portfolio-level, recurring usage |

**Hypothesis:** Exposing a rate-limited free tier of the home value mode (5 requests/day) generates leads that convert to paid API access for BPO mode.

## GIS Teaser Engineering

The teaser system in `internal/service/gis/teaser.go` is configurable:

```go
// Existing — configurable limits
maxFeatures := cfg.TeaserMaxFeatures    // default 40
decimals := cfg.TeaserCoordDecimals      // default 4
```

These values are per-request, not per-user. Engineering opportunities:

### Per-User Teaser Tracking

```go
// new code to add — track teaser usage per user for rate limiting
func (h *Handler) GISProxy(c *fiber.Ctx) error {
    uid := c.Locals("user_id")
    if uid != nil {
        // Count teaser requests this month
        var count int
        h.db.Pool.QueryRow(c.Context(), `
            SELECT COUNT(*) FROM audit_logs
            WHERE user_id = $1 AND action = 'gis.teaser_request'
            AND created_at > date_trunc('month', NOW())
        `, uid).Scan(&count)
        if count > 100 {
            // Prompt upgrade
        }
    }
}
```

### Teaser as Onboarding Tool

The teaser returns real data but degraded. This means:

1. **No mock data needed** — the product demonstrates itself with live parcel geometry
2. **Upgrade is natural** — users hit the 40-feature limit on real properties
3. **Measurable** — truncated responses are a quantifiable demand signal

## Automation Opportunities

### Scheduled Growth Reports

The scheduler (`cmd/scheduler`) already runs cron jobs. Add a weekly growth summary:

```go
// new code to add — growth report job type
// In scheduler: enqueue weekly on Monday at 09:00
// Job queries audit_logs, domains, tokens and posts a summary
```

See the **queue-postgresql** skill for job enqueue patterns.

### Domain Verification Reminders

Users who add domains but never verify are stuck in the funnel:

```sql
-- Find stale pending domains
SELECT d.id, d.domain_slug, u.email, d.created_at
FROM domains d
JOIN users u ON u.id = d.user_id
WHERE d.verification_status = 'pending'
AND d.created_at < NOW() - INTERVAL '3 days';
```

A scheduled job could email these users with a reminder. See the **queue-postgresql** skill for scheduling.

## Anti-Patterns

### WARNING: In-Process Growth Counters

**The Problem:**
Using Go maps or `sync/atomic` counters to track teaser hits, invite clicks, or conversion events.

**Why This Breaks:**
- Multi-DC deployment (NYC + ATL) means in-process state diverges between instances
- Process restart loses all counter state
- Cannot be queried for analysis

**The Fix:**
All counters go to `audit_logs` in PostgreSQL. The database is the source of truth. See the **cache-postgres** skill for PostgreSQL-backed state patterns.

### WARNING: Growing Without MLS Compliance Review

**The Problem:**
Increasing distribution velocity without reviewing MLS data licensing terms for each new dataset.

**Why This Breaks:**
MLS feeds have specific usage restrictions (display rules, attribution, caching limits). Rapid growth can trigger compliance violations that result in feed termination.

**The Fix:**
Before adding any growth surface that exposes MLS data to new audiences, review the MLS terms for the relevant `dataset_slug`. The `allowed_mls_datasets` column on `domains` exists for this reason.