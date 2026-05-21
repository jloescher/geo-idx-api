# Activation & Onboarding Reference

## Contents
- Activation Funnel
- Current Onboarding Flow
- Gating New Features During Onboarding
- Measuring Activation
- Common Errors

---

## Activation Funnel

The idx-api activation path is entirely server-driven with no frontend SPA:

1. **Invite** — Admin creates invitation (`POST /dashboard/invitations`)
2. **Accept** — User sets name/password (`POST /invite/:token`)
3. **Login** — Session created (`POST /login`)
4. **Add domain** — User registers domain (`POST /dashboard/domains`)
5. **Verify DNS** — TXT record challenge (`POST /dashboard/domains/:id/verify-txt`)
6. **Create token** — User issues API token (`POST /dashboard/api-tokens`)
7. **First API call** — Proxy/search request with token auth

Each step is a measurable drop-off point. The `audit_logs` table and `tokens`/`domains` tables hold the data.

## Current Onboarding Flow

The dashboard (`internal/handler/dashboard/handler.go`) controls onboarding:

```go
// Existing route registration
func (h *Handler) Register(app *fiber.App) {
    app.Get("/login", h.LoginForm)
    app.Post("/login", h.Login)
    app.Get("/dashboard", h.requireAuth, h.Dashboard)
    app.Post("/dashboard/domains", h.requireAuth, h.StoreDomain)
    app.Post("/dashboard/domains/:id/verify-txt", h.requireAuth, h.VerifyTXT)
    app.Post("/dashboard/api-tokens", h.requireAuth, h.CreateToken)
    // ...
}
```

**Key tables:** `users`, `domains`, `tokens`, `invitations`.

## Gating New Features During Onboarding

### DO: Gate at the handler level after auth

```go
// new code to add
func (h *Handler) CreateToken(c *fiber.Ctx) error {
    if !h.cfg.MLS.BeachesEnabled {
        return c.Status(403).JSON(fiber.Map{"error": "Feature not available"})
    }
    // ... existing token creation logic
}
```

### DON'T: Gate in the template or response rendering

The web layer (`internal/web/layout.go`) builds HTML strings. Feature checks here are invisible to API consumers and easily bypassed.

## Measuring Activation

No activation metrics exist yet. To instrument:

1. **Add `activated_at` column to `domains`** — set when domain passes DNS verification
2. **Query time-to-activate** — `now() - created_at` where `activated_at IS NOT NULL`
3. **Track first API call per token** — query `audit_logs` grouped by `domain_slug` for earliest `created_at`

```sql
-- Activation rate by week
SELECT date_trunc('week', d.created_at) AS cohort,
       COUNT(*) AS signups,
       COUNT(d.activated_at) AS activated,
       ROUND(100.0 * COUNT(d.activated_at) / COUNT(*), 1) AS pct
FROM domains d
GROUP BY 1 ORDER BY 1;
```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| Invitation token not found | Expired or already used | Check `invitations.expires_at`; reuse shows 404 |
| DNS verification loop | TXT record value mismatch | Compare stored `txt_challenge` against actual DNS |
| Token created before domain verified | No gate in `CreateToken` | Add `domain.verified_at IS NOT NULL` check before token issuance |

See the **auth-api-token** skill for token scoping patterns.