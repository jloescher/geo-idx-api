# Auth API Token Workflows

## Contents
- Token Creation Flow
- Token Validation Flow (Per Request)
- Token Revocation
- Domain Verification + Token Auto-Creation
- Adding a New Protected Endpoint

## Token Creation Flow

Triggered from the dashboard when a user creates an API token.

```go
// internal/repository/token.go — Create
// 1. Generate 32 random bytes → hex encode → prefix "idx_"
// 2. Build abilities JSON: `["idx:full"]` or `["idx:access"]`
// 3. Hash plaintext with SHA-256
// 4. INSERT into personal_access_tokens
// 5. Return plaintext (shown ONCE to user, never stored)
```

**Key invariant:** The plaintext token is returned exactly once. There is no recovery mechanism — if the user loses it, they must revoke and re-create.

Copy this checklist for token creation changes:
- [ ] Token uses `crypto/rand.Read` (not `math/rand`)
- [ ] Plaintext is `idx_` + 64 hex chars (32 bytes)
- [ ] Database stores `HashToken(plain)` — never plaintext
- [ ] Abilities are valid JSON array string
- [ ] `tokenable_type` is `'App\\Models\\User'` (Laravel parity)
- [ ] Return value is the plaintext token for one-time display

## Token Validation Flow (Per Request)

Every request to MLS/GIS/image endpoints passes through `DomainToken` middleware.

```
Request → DomainToken middleware
  ├─ Authorization: Bearer present?
  │   ├─ Yes → handleToken()
  │   │   1. Hash plaintext → SELECT from personal_access_tokens
  │   │   2. Expired? → return nil (403)
  │   │   3. Load user from users table
  │   │   4. Check ability (idx:access or idx:full)
  │   │   5. Resolve domain slug (X-Domain-Slug or ?domain=)
  │   │   6. Domain active + owned by user? → 403 if not
  │   │   7. Domain TXT-verified? → 403 if not
  │   │   8. Set Locals → c.Next()
  │   └─ No → handleDomain()
  │       1. Resolve slug (X-Domain-Slug, ?domain=, or Referer hostname)
  │       2. SELECT active domain by slug
  │       3. Set Locals with fullAccess=true → c.Next()
  └─ Handler reads Locals via ctxkeys constants
```

### WARNING: Legacy Sanctum Tokens Are Not Supported

The `FindByPlaintext` method returns `nil, nil, nil` for tokens not matching the SHA-256 format. Legacy Laravel Sanctum tokens (`id|secret` format) cannot be validated. Users must re-issue tokens from `/dashboard` after Go cutover.

## Token Revocation

Deletion from `personal_access_tokens` (hard delete, not soft):

```go
// internal/repository/token.go — Revoke
tag, err := r.db.Pool.Exec(ctx,
    `DELETE FROM personal_access_tokens WHERE id = $1 AND tokenable_id = $2`,
    tokenID, userID)
if tag.RowsAffected() == 0 {
    return fmt.Errorf("token not found")
}
```

Revocation is ownership-scoped: both `tokenID` and `userID` must match. A user cannot revoke another user's token.

## Domain Verification + Token Auto-Creation

When a domain passes TXT verification via the dashboard, a production token is auto-created:

1. Dashboard handler calls domain verification endpoint
2. DNS TXT record validated against `txt_verification_name` / `txt_verification_value`
3. On success, `verification_status` set to `"verified"` or `"verified_ghl"`
4. Production token auto-created with `["idx:full"]` abilities
5. Plaintext shown once in dashboard response

See the **auth-domain** skill for the full domain verification workflow.

## Adding a New Protected Endpoint

Copy this checklist and track progress:
- [ ] Register route under the `DomainToken` middleware group (See the **fiber** skill for router patterns)
- [ ] Extract auth state from Fiber Locals using `ctxkeys` constants
- [ ] Use comma-ok type assertions for all Locals access
- [ ] Call `auditLogger.Log()` after handler logic completes
- [ ] Test both auth paths: Bearer token AND domain-only
- [ ] Verify `idx:access` tokens get correct access level (e.g., GIS teaser for limited tokens)
- [ ] Confirm error responses are generic (no token structure leaks)

### Feedback Loop

1. Add route with `DomainToken` middleware
2. Validate: `go build ./...` compiles
3. Test: `go test ./internal/api/middleware/...`
4. Manual: `curl -H "Authorization: Bearer idx_..." -H "X-Domain-Slug: example.com" localhost:8000/api/v1/your-endpoint`
5. Verify audit log row appears in `mls_proxy_audit_logs`

### WARNING: Don't Register Routes Without DomainToken

**The Problem:**
```go
// BAD — no auth middleware on MLS data endpoint
app.Get("/api/v1/listings", listingHandler)
```

**Why This Breaks:** MLS data requires domain authorization. Unauthenticated endpoints expose licensed MLS listings — a compliance violation that can result in feed termination.

**The Fix:**
```go
// GOOD — wrap in DomainToken middleware group
apiGroup := app.Group("/api/v1", middleware.DomainToken(cfg, domainRepo, tokenRepo))
apiGroup.Get("/listings", listingHandler)
```