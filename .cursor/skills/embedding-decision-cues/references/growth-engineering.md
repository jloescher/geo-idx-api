# Growth Engineering Reference

## Contents
- Growth Levers Available
- Staging Token as Growth Tool
- GIS Teaser as Upgrade Engine
- Invitation Virality Coefficient
- Anti-Patterns

## Growth Levers Available

Quantyra IDX has limited but focused growth mechanisms:

| Lever | Implementation | Growth mechanism |
|-------|---------------|-----------------|
| Staging tokens | `CreateStagingToken()` | Frictionless trial before domain commitment |
| GIS teaser tier | `applyTeaser()` | Data-preview drives upgrade desire |
| Invite-only | `CreateInvitation()` | Exclusivity + controlled distribution |
| Multi-MLS support | `?dataset=stellar\|beaches` | Expanded value per user |

## Staging Token as Growth Tool

The staging token flow in `CreateStagingToken()` is the primary growth tool:

```go
// internal/handler/dashboard/handler.go
// Staging tokens get idx:full scope without domain verification
// This lets developers test the API before committing to DNS setup
func (h *Handler) CreateStagingToken(c *fiber.Ctx) error {
    // One staging token per user (409 if exists)
    plain, err := h.tokens.Create(c.Context(), uid, "Staging", []string{"idx:full"})
    return c.SendString("Staging token: " + plain)
}
```

**Growth insight:** The 409 conflict ("Staging token already exists") is actually a **positive signal** — it means the user already has a staging token. Consider surfacing this as a progress cue: "You already have a staging token. Ready to verify a domain?"

## GIS Teaser as Upgrade Engine

The teaser tier in `internal/service/gis/teaser.go` creates upgrade motivation through **data fidelity degradation**, not feature blocking:

```go
// Full data vs teaser comparison:
// Full:     unlimited features, full coordinate precision
// Teaser:   40 features, 4 decimal places (~11m accuracy)
//
// The gap is noticeable in cadastral work (parcel boundaries)
// but sufficient for site-level evaluation (is this a lot?)
```

### DO: Make the teaser boundary meaningful

The teaser should show enough value to demonstrate capability but not enough to replace full access. The default 40 features / 4 decimal places achieves this for parcel data.

### DON'T: Set teaser caps so low that users can't evaluate value

If users can't see meaningful patterns in 40 features, the teaser fails as a conversion tool. Monitor GIS usage via audit logs to calibrate.

## Invitation Virality Coefficient

The invitation system has a built-in viral coefficient of exactly 1 (admin invites N users, each of whom can only use the platform — they cannot invite others):

```go
// Only admins can invite — non-linear viral growth is impossible by design
func (h *Handler) requireAdmin(c *fiber.Ctx) error {
    var isAdmin bool
    err := h.db.Pool.QueryRow(c.Context(),
        `SELECT is_admin FROM users WHERE id = $1`, uid).Scan(&isAdmin)
    if err != nil || !isAdmin {
        return fiber.NewError(fiber.StatusForbidden, "admin only")
    }
    return c.Next()
}
```

This is intentional for an MLS platform — controlled access is a compliance requirement, not a growth limitation.

### Growth Checklist

Copy this checklist and track progress:
- [ ] Measure staging token creation rate vs domain verification rate (conversion lag)
- [ ] Track GIS teaser response headers to estimate upgrade demand
- [ ] Audit invitation TTL (168h default) against actual acceptance timing
- [ ] Verify multi-MLS routing works for both datasets
- [ ] Monitor `GET /api/v1/bridge/stats` for replication health (affects API value)

## Anti-Patterns

### WARNING: Removing Invite Gate for Growth

**The Problem:** Opening registration to drive signups.

**Why This Breaks:** MLS data access has legal/compliance requirements. Open registration without domain verification creates uncontrolled access to licensed listing data. The invite-only system exists for MLS compliance, not artificial scarcity.

**The Fix:** Keep invite-only. Growth comes from making the invited experience excellent, not from opening the gate.

### WARNING: Aggressive Teaser Degradation

**The Problem:** Setting teaser caps so aggressive (e.g., 5 features, 1 decimal) that the data becomes useless.

**Why This Breaks:** Users who can't evaluate the data won't upgrade — they'll leave. The teaser must demonstrate enough value to create desire for the full product.

**The Fix:** Monitor conversion from `idx:access` to `idx:full`. If conversion is low, increase teaser caps. If conversion is high and full-access users aren't churning, the caps are well-calibrated.

See the **auth-api-token** skill for invitation and token management patterns.
See the **geospatial** skill for GIS tier configuration.