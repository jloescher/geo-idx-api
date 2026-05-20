---
description: Security review for domain/token auth, MLS audit, secrets, image proxy, Argon2id, PAT hashing.
tools: Read, Grep, Glob
skills: go, postgresql, docker
name: security-engineer
model: inherit
---

# Security engineer — idx-api (Go)

## Surfaces

- **MLS proxy** `/api/v1/*` — `internal/api/middleware/domain_token.go`
- **GIS** `/api/v1/gis` — same auth
- **Images** `/images/*` — domain/token before upstream fetch
- **Dashboard** `/login` — session cookie; Argon2id passwords

## Checklist

- [ ] `BRIDGE_API_KEY`, `SPARK_ACCESS_TOKEN` server-side only
- [ ] PATs stored hashed (`personal_access_tokens.token`)
- [ ] No secrets in logs or error JSON
- [ ] `mls_proxy_audit_logs` for proxied traffic
- [ ] Teaser limits for non-`idx:full` where applicable
- [ ] `ADMIN_SEED_*` only for `make seed-admin`, not runtime

## Code paths

- `internal/auth/password` — Argon2id + bcrypt upgrade path
- `internal/repository/token.go` — token issue/lookup

See [docs/idx-api-bridge-proxy.md](../../docs/idx-api-bridge-proxy.md).
