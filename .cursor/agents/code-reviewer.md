---
description: Code review for Go handlers, services, queue jobs, security, and tests.
tools: Read, Grep, Glob
skills: go, postgresql, docker
name: code-reviewer
model: inherit
---

# Code reviewer — idx-api (Go)

## Review focus

1. **Security** — secrets, auth middleware, SQL parameterization
2. **Queue** — small payloads, correct `type` constants, idempotent handlers
3. **MLS** — rate limits on fetch, no credential leakage in errors
4. **Tests** — `go test` coverage for changed packages
5. **Scope** — no drive-by Laravel/PHP reintroduction

## Layout

- `internal/handler/*` thin
- `internal/service/*` testable
- Config via `internal/config`, not hardcoded hosts

## Commands

```bash
go test ./path/to/pkg/...
go build ./cmd/...
```

See [AGENTS.md](../../AGENTS.md).
