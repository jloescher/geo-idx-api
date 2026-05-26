# Activation & Onboarding Reference

How users go from "no account" to "making authenticated API calls" in Quantyra IDX.

## Contents
- Onboarding Flow
- Activation Milestones
- Scoping New Onboarding Steps
- Anti-Patterns

## Onboarding Flow

The activation path is linear and invite-gated:

```
Admin seed → Invite link → Register → Login → Add domain → DNS TXT verify → Receive API token → First API call
```

Each step maps to code in the dashboard handler (`internal/handler/dashboard/handler.go`):

| Step | Handler method | Storage |
|------|---------------|---------|
| Admin seed | `cmd/seed/main.go` | `users` table |
| Create invite | `CreateInvitation()` | `invitations` table (SHA256 token hash) |
| Accept invite | `AcceptInvitation()` | Creates `users` row |
| Login | `Login()` | Session store (`fiber/v2/middleware/session`) |
| Add domain | `StoreDomain()` | `domains` row (`verification_status: pending`) |
| DNS verify | `VerifyTXT()` | `dns.VerifyTXT()` → updates status to `verified` |
| Token issued | `tokens.Create()` inside `VerifyTXT()` | `personal_access_tokens` (SHA256 hash) |

### Activation milestone: first API call

The production token is **only** issued after DNS TXT verification succeeds (`dashboard/handler.go:224`). This is the activation gate — a verified domain proves the customer controls the hostname that will make API calls.

```go
// internal/handler/dashboard/handler.go:224
// Token is issued AFTER successful DNS verification
plain, _ := h.tokens.Create(c.Context(), uid, "Production", []string{"idx:full"})
```

## Activation Milestones

When scoping features, define which milestone the work advances:

| Milestone | Signal | Table |
|-----------|--------|-------|
| Account created | `users` row exists | `users` |
| Domain added | `domains` row with `pending` status | `domains` |
| Domain verified | `verification_status = 'verified'` | `domains` |
| Token issued | `personal_access_tokens` row | `personal_access_tokens` |
| First API call | `mls_proxy_audit_logs` row | `mls_proxy_audit_logs` |
| First search | Audit log with `request_type = 'search'` | `mls_proxy_audit_logs` |

## Scoping New Onboarding Steps

### WARNING: Adding steps before verification

**The Problem:** Inserting onboarding steps between "add domain" and "DNS verify" drops activation rate. Each extra step is a drop-off point.

**The Fix:** New onboarding steps should go **after** the first API call (post-activation). Pre-verification, the flow must stay minimal: add domain → verify → token.

### Acceptance criteria template for onboarding changes

```
Given [user state from milestone table]
When [user takes action]
Then [observable outcome in table/API]
And [no regression in existing activation path]
```

Example for "add MLS dataset selector to domain form":

```
Given user is logged in with no domains
When user submits domain form with dataset=beaches
Then domains row has mls_dataset='beaches' and allowed_mls_datasets='["beaches"]'
And DNS verification flow still works unchanged
```

## Anti-Patterns

### WARNING: Auto-issuing tokens without verification

Tokens must require domain verification. Issuing tokens for unverified domains allows unauthenticated API access from any hostname — this defeats the domain-based auth model.

### WARNING: Invitation tokens stored in plaintext

`internal/service/auth/invitations.go` stores SHA256 hashes. Invitation links use the raw token in the URL (`/invite/:token`). Never store raw invitation tokens in the database.

## See Also

- See the **auth-api-token** skill for token creation and verification
- See the **fiber** skill for session middleware patterns