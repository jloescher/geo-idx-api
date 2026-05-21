# Activation & Onboarding Reference

## Contents
- Onboarding Flow
- Invitation Journey
- Domain Registration Gate
- TXT Verification Step
- API Token Creation
- Friction Points

---

## Onboarding Flow

The full activation journey from admin invite to first successful API call.

### Journey Map

```
1. Admin creates invitation     POST /dashboard/invitations (requireAdmin)
2. Recipient opens invite link  GET /invite/:token
3. Recipient registers          POST /invite/:token (name + password, min 8 chars)
4. Recipient logs in            POST /login (email + password)
5. User adds domain             POST /dashboard/domains (domain_slug + mls_dataset)
6. User verifies domain         POST /dashboard/domains/:id/verify-txt
7. User creates API token       POST /dashboard/api-tokens (name)
8. First API call               Any /api/v1/* with Bearer + X-Domain-Slug
```

### Code Locations

| Step | Handler | File |
|------|---------|------|
| Create invitation | `dashboard.CreateInvitation()` | `internal/handler/dashboard/handler.go` |
| Show invite form | `dashboard.InviteRegisterForm()` | same |
| Accept invite | `dashboard.AcceptInvitation()` | same |
| Login | `dashboard.Login()` | same |
| Add domain | `dashboard.StoreDomain()` | same |
| Verify TXT | `dashboard.VerifyTXT()` | same |
| Create token | `dashboard.CreateToken()` | same |

---

## Invitation Journey

Controlled by `internal/repository/invitation.go`:

- Token hash stored in `user_invitations`
- Expires after `Auth.InvitationTTL` (default 168 hours / 7 days)
- Accept sets `accepted_at`, preventing reuse
- Only admins can create invitations (`requireAdmin` middleware)

### WARNING: No Invite Resend

**The Problem:** If an invitation expires or is lost, there is no resend flow. Admins must create a new invitation.

**Why This Breaks:** No UI indication that an invitation expired vs was never sent. The `FindOpenByHash` query filters `accepted_at IS NULL AND expires_at > NOW()`, returning no rows silently.

**The Fix:** When mapping this journey, note the gap — expired invitations need admin re-creation, not user-facing retry.

---

## Domain Registration Gate

`POST /dashboard/domains` accepts `domain_slug` (text) and `mls_dataset` (default `"stellar"`).

Domain record created with `verification_status` = unverified. The domain must be TXT-verified before API token access works.

Middleware enforcement (`internal/api/middleware/domain_token.go`):

```go
// Gate: DomainToken middleware returns 403 if domain not verified
"Domain must be TXT-verified before API token access is allowed."
```

---

## TXT Verification Step

`POST /dashboard/domains/:id/verify-txt` triggers DNS lookup:

- Looks up `txt_verification_name` (e.g., `_quantyra-verify.example.com`)
- Checks for `txt_verification_value` in DNS TXT records
- On success: sets `txt_verified_at`, enables API token auth
- On failure: returns 422 `"TXT record not found. Publish the verification record at your DNS host, then try again."`
- On DNS error: returns 502 `"DNS lookup failed"`

### Friction Point

This is the highest-friction step. It requires:
1. User to access their DNS hosting provider
2. Create a TXT record at a specific subdomain
3. Wait for DNS propagation
4. Return to dashboard to verify

No polling or automatic retry exists. User must manually re-submit.

---

## API Token Creation

Two token types:

| Type | Endpoint | Scope | Limit |
|------|----------|-------|-------|
| Production | `POST /dashboard/api-tokens` | User's domain | Multiple |
| Staging | `POST /dashboard/api-tokens/staging` | User's domain | One only (409 on duplicate) |

Token is returned once in plain text (`repository/token.go` → `Create()`). SHA-256 hash stored in database. No recovery mechanism — lost tokens must be revoked and recreated.

### WARNING: Staging Token Single-Instance

**The Problem:** `CreateStagingToken()` returns 409 `"Staging token already exists"` if one already exists.

**Why This Breaks:** User can't see the existing staging token value. They must revoke first, losing any integrations using it.

**The Fix:** When mapping this journey, surface the revoke-before-recreate step as a friction point.

---

## Friction Points

| Step | Risk | User Impact |
|------|------|-------------|
| Invitation expiry | No resend, no notification | Blocked users need admin intervention |
| TXT verification | Manual DNS change, no polling | Drop-off during DNS propagation wait |
| Token display-once | No re-display mechanism | Lost tokens require revoke + recreate |
| Staging token 409 | No list/replace flow | Confusing error for new users |
| Domain active check | `is_active` gate | Deactivated domains silently block all API calls |

---

## Onboarding Checklist

Copy this checklist and track progress:

- [ ] Admin account seeded (`make seed-admin`)
- [ ] Invitation created for new user
- [ ] User registered via invite link
- [ ] User logged in to dashboard
- [ ] Domain registered with correct MLS dataset
- [ ] DNS TXT record published
- [ ] TXT verification completed
- [ ] Production API token created and stored securely
- [ ] First API call successful (`GET /api/v1/properties` with Bearer + X-Domain-Slug)