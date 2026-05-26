# Activation & Onboarding Reference

## Contents
- Activation milestones
- Domain creation flow
- Token issuance flow
- First-run verification
- Common errors

## Activation Milestones

The activation funnel for a new Quantyra IDX customer:

| Step | Action | Measured by |
|------|--------|-------------|
| 1. Domain authorized | Admin adds domain via `/dashboard` | `domains` row exists |
| 2. Token created | Customer creates API token | `tokens` row exists with `active = true` |
| 3. First API call | Customer hits `/api/v1/properties` or `/api/v1/search` | `audit_logs` entry with action `api.request` |
| 4. Data returned | Customer receives listing data (200 response with results) | Response contains `value` array with items |
| 5. Replication verified | Customer checks sync status | `GET /api/v1/bridge/stats` returns `last_sync_finished_at` |

### DO: Track milestones via audit logs

```go
// new code to add — record activation milestone in audit log
func (r *Repository) RecordMilestone(ctx context.Context, domainID, action string) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO audit_logs (action, subject_type, subject_id, metadata, created_at)
         VALUES ($1, 'domain', $2, '{}', NOW())`,
        action, domainID,
    )
    return err
}
// Call with: "domain.created", "token.created", "api.first_call", "data.first_listing"
```

### DON'T: Use in-memory counters for milestone tracking

```go
// BAD — lost on restart, not shared across instances, breaks multi-DC
var activationCount map[string]int // shared mutable state
```

In-memory state fails in multi-DC (NYC + ATL). Both API instances must see the same activation state. See the **cache-postgres** skill.

## Domain Creation Flow

The `/dashboard` is invite-only. Admin creates domains and customers manage tokens.

### DO: Validate domain uniqueness before insert

```go
// new code to add — idempotent domain creation
func (r *Repository) CreateDomain(ctx context.Context, hostname string) (string, error) {
    var id string
    err := r.db.QueryRowContext(ctx,
        `INSERT INTO domains (hostname, active, created_at)
         VALUES ($1, true, NOW())
         ON CONFLICT (hostname) DO UPDATE SET updated_at = NOW()
         RETURNING id`,
        hostname,
    ).Scan(&id)
    return id, err
}
```

### DON'T: Skip hostname normalization

```go
// BAD — "Example.COM" and "example.com" create duplicate domains
db.ExecContext(ctx, `INSERT INTO domains (hostname) VALUES ($1)`, rawHostname)

// GOOD — normalize before insert
hostname = strings.ToLower(strings.TrimPrefix(rawHostname, "www."))
```

## Token Issuance Flow

Customers create tokens from the dashboard. Tokens use SHA-256 hashes (not Laravel Sanctum format).

### DO: Hash tokens with SHA-256 at creation time

```go
// new code to add — consistent with existing auth pattern
import "crypto/sha256"

func HashToken(plaintext string) string {
    h := sha256.Sum256([]byte(plaintext))
    return hex.EncodeToString(h[:])
}
```

### WARNING: Legacy Sanctum tokens are NOT accepted

The Go API rejects `id|secret` format tokens. Customers must re-issue tokens from `/dashboard` after cutover. See `docs/go-cutover.md`.

## First-Run Verification

After activation, verify the customer can reach data:

```
Copy this checklist and track progress:
- [ ] Domain exists in `domains` table with `active = true`
- [ ] Token exists in `tokens` table linked to domain
- [ ] `GET /api/v1/properties?dataset=stellar` returns 200 with `value` array
- [ ] `GET /api/v1/bridge/stats` shows `last_sync_finished_at` not null
- [ ] `POST /api/v1/search` returns listings within the domain's authorized area
```

### Validation loop

1. Make changes to activation flow
2. Verify: `GOFLAGS=-mod=mod go test ./internal/handler/auth/... ./internal/handler/bridge/...`
3. If tests fail, fix and repeat step 2
4. Manual smoke: create domain → create token → `curl -H "Authorization: Bearer <token>" http://localhost:8000/api/v1/properties?dataset=stellar`

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| 403 on `/api/v1/*` | Domain not in `domains` table or token revoked | Verify domain exists and token `active = true` |
| Empty `value` array | Replication not yet complete | Check `GET /api/v1/bridge/stats` and wait for scheduler kickoff |
| 401 on dashboard | Invite-only; admin must seed domain | Use `make seed-admin` then add domain from dashboard |
| Legacy token rejected | Sanctum `id\|secret` format | Re-issue from `/dashboard` (see **auth-api-token** skill) |