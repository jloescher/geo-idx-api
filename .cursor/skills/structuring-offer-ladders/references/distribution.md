# Distribution Reference

## Contents
- Current Distribution Model
- API-as-Product Distribution
- Go-to-Market Channels
- Anti-Patterns

## Current Distribution Model

Quantyra IDX is an **invite-only B2B API**. Distribution flows through:

1. **Direct sales** — admin invites via `internal/service/auth/invitation.go`
2. **DNS-verified domains** — users register domains, verify TXT records, get API access
3. **Dashboard tokens** — self-service token creation at `/dashboard/api-tokens`

No public signup exists. The `CreateInvitation` route is admin-only (`h.requireAdmin` middleware at `internal/handler/dashboard/handler.go:57`).

## API-as-Product Distribution

The offer ladder distribution model for an API product differs from SaaS:

| Distribution lever | Current state | Opportunity |
|-------------------|--------------|-------------|
| Public docs | `docs/` markdown, OpenAPI spec | Convert readers → signups |
| Self-serve signup | Blocked (invite-only) | Open `idx:access` tier for self-serve |
| API response headers | None | Add `X-Plan-Limit`, `X-Plan-Remaining` |
| Dashboard | Domain/token management only | Usage analytics, upgrade prompts |
| Referral | None | Domain-level referrals with shared limits |

### Opening Self-Serve for Starter Tier

The existing `idx:access` tier is a natural free/low-cost entry point. Distribution expansion path:

1. **Phase 1 (current):** Invite-only, sales-led, custom onboarding
2. **Phase 2:** Open `idx:access` for self-serve signup — full MLS search, teaser GIS
3. **Phase 3:** Public pricing page, usage-based billing for `idx:full`

### API Response Marketing

Every API response is a distribution touchpoint:

```go
// new code to add — rate limit headers that signal plan value
c.Set("X-RateLimit-Limit", strconv.Itoa(plan.RateLimit))
c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
if remaining < plan.RateLimit/10 {
    c.Set("X-Upgrade-URL", "https://"+h.cfg.Idx.PlatformURL+"/pricing")
}
```

## Go-to-Market Channels

For an MLS data API, effective channels map to the buyer journey:

| Channel | Entry point | Conversion path |
|---------|-----------|-----------------|
| Developer docs | `/openapi.json`, `docs/` | Read → signup → `idx:access` token |
| RESO community | Industry standards body | Referral → invite → custom onboarding |
| IDX platform integrations | `IDX_PLATFORM_URL` config | Platform users → API keys → paid tier |
| GIS/comp demos | GIS teaser responses | Teaser → upgrade CTA → `idx:full` |

### Anti-Pattern: Marketing Site Separate from API

```
// BAD — separate marketing site that doesn't reflect API reality
marketing-site.com/pricing → Stripe checkout → API key emailed
```

**Why This Breaks:** For an API product, the best marketing is the API itself. Response headers, interactive docs, and working examples convert better than landing pages.

**The Fix:** Keep marketing embedded in the API app. The current architecture (single Go binary serving both `/api/v1/*` and `/dashboard`) is correct.

## Anti-Patterns

### WARNING: Public Signup Without Rate Limits

```go
// BAD — opening signup without plan enforcement
func (h *Handler) Register(c *fiber.Ctx) error {
    // creates user with idx:full — no quota, no limits
    user := h.createUser(email, password)
    h.createToken(user.ID, "idx:full") // gives full access immediately
}
```

**Why This Breaks:** Open signup + no rate limits = abuse vector. If opening self-serve, `idx:access` with enforced quotas must be the default.

See the **auth-api-token** skill for token creation and ability assignment.
See the **cache-postgres** skill for rate limiting with PostgreSQL backends.