# Auth API Token Patterns

## Contents
- Token Storage and Hashing
- Dual-Mode Authentication
- Ability Checking
- Audit Logging
- Anti-Patterns

## Token Storage and Hashing

Tokens use SHA-256 hashing. The plaintext (`idx_` + 64 hex chars) is never stored.

```go
// internal/repository/token.go — HashToken
func HashToken(plain string) string {
    sum := sha256.Sum256([]byte(plain))
    return hex.EncodeToString(sum[:])
}
```

**Token creation** generates 32 random bytes, prefixes `idx_`, hashes for storage:

```go
// internal/repository/token.go — Create
b := make([]byte, 32)
rand.Read(b)
plain = "idx_" + hex.EncodeToString(b)  // shown ONCE to user
abil := `["` + strings.Join(abilities, `","`) + `"]`
// store HashToken(plain), not plain
```

### WARNING: Never Store Plaintext Tokens

**The Problem:**
```go
// BAD — stores raw token in database
_, err := db.Exec(ctx, `INSERT INTO tokens (token) VALUES ($1)`, plainToken)
```

**Why This Breaks:** Database compromise leaks all active tokens. Anyone with DB access can impersonate every API consumer.

**The Fix:**
```go
// GOOD — store SHA-256 hash only
_, err := db.Exec(ctx, `INSERT INTO personal_access_tokens (..., token) VALUES (..., $1)`, HashToken(plain))
```

**When You Might Be Tempted:** Debugging auth issues and wanting to "see" the token in the database.

## Dual-Mode Authentication

The `DomainToken` middleware supports two auth paths:

**Bearer token** (`Authorization: Bearer idx_...`):
1. Hash token → lookup in `personal_access_tokens`
2. Check ability (`idx:access` or `idx:full`)
3. Resolve domain slug from `X-Domain-Slug` header or `?domain=` query param
4. Verify domain ownership + TXT verification

**Domain-only** (no token):
1. Resolve slug from `X-Domain-Slug`, `?domain=`, or `Referer` hostname
2. Lookup active domain by slug
3. Grant full access (`MLSFullAccess=true`)

```go
// internal/api/middleware/domain_token.go — dispatch
auth := c.Get("Authorization")
if strings.HasPrefix(auth, "Bearer ") {
    return handleToken(c, domains, tokens, plain)
}
return handleDomain(c, domains)
```

### WARNING: Don't Bypass Domain Verification

**The Problem:**
```go
// BAD — skip TXT verification for "convenience"
d, _ := domains.FindActiveForUser(ctx, userID, slug)
setMLSLocals(c, "token", d, ...)  // no verification check
```

**Why This Breaks:** Unverified domains can belong to anyone. Skipping verification allows unauthorized MLS data access — a compliance violation with MLS providers.

**The Fix:**
```go
// GOOD — existing pattern from handleToken
if !d.IsVerified() {
    return fiber.NewError(fiber.StatusForbidden,
        "Domain must be TXT-verified before API token access is allowed.")
}
```

## Ability Checking

Abilities are stored as a JSON string in the `abilities` column. Two valid abilities:

| Ability | Access Level |
|---------|-------------|
| `idx:full` | Full MLS data access |
| `idx:access` | Limited access (GIS teaser applies) |

```go
// internal/repository/token.go — HasAbility
func (r *TokenRepo) HasAbility(tok *domain.APIToken, ability string) bool {
    if tok.Abilities == nil {
        return false
    }
    s := *tok.Abilities
    return strings.Contains(s, ability)
}
```

### WARNING: Don't Add New Abilities Without Middleware Updates

**The Problem:** Adding a new ability string (e.g., `idx:admin`) without updating the `handleToken` guard means tokens with that ability alone get rejected:

```go
// BAD — only checks these two abilities
if !tokens.HasAbility(tok, "idx:access") && !tokens.HasAbility(tok, "idx:full") {
    return fiber.NewError(fiber.StatusForbidden, "Token is missing required IDX abilities.")
}
```

**The Fix:** Update the ability gate in `handleToken` when introducing new scopes, or refactor to a centralized ability validator.

## Audit Logging

Audit entries are fire-and-forget. The `Log` method is nil-safe and errors are ignored (intentional for request-path logging):

```go
// internal/service/audit/logger.go
func (l *Logger) Log(c *fiber.Ctx, requestType string, listingCount *int, cacheHit *string) {
    if l == nil || l.db == nil { return }
    // extract from Locals, INSERT into mls_proxy_audit_logs
    _, _ = l.db.Pool.Exec(context.Background(), `INSERT INTO ...`, ...)
}
```

Logged fields: `domain_slug`, `token_name`, `request_type`, `listing_count`, `ip_address`, `user_id`, `cache_hit`.

**Never log the plaintext token.** Use `token_name` (the human-readable label like "Production") instead.

## Context Locals Contract

After `DomainToken` middleware, these Fiber Locals are guaranteed set:

| Key | Type | Set By |
|-----|------|--------|
| `mls.auth` | `string` | `"token"` or `"domain"` |
| `mls.domain` | `*domain.Domain` | Both paths |
| `mls.domain_slug` | `string` | Both paths |
| `mls.token_name` | `*string` | Token path only (nil for domain) |
| `mls.user_id` | `*int64` | Token path only (nil for domain) |
| `mls.full_access` | `bool` | `true` for domain/`idx:full`, `false` for `idx:access` |

### WARNING: Don't Access Locals Without Type Assertion

**The Problem:**
```go
// BAD — panics if middleware didn't set the value
slug := c.Locals("mls.domain_slug").(string)
```

**Why This Breaks:** If a route is registered without the `DomainToken` middleware, `Locals` returns nil and the type assertion panics.

**The Fix:**
```go
// GOOD — safe type assertion with comma-ok
slug, ok := c.Locals(ctxkeys.MLSDomainSlug).(string)
if !ok || slug == "" {
    return fiber.NewError(fiber.StatusUnauthorized, "Missing domain context.")
}
```