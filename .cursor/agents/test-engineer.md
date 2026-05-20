---
description: Go tests for Bridge, GIS, dashboard, queue — PostgreSQL integration where needed.
tools: Read, Edit, Write, Glob, Grep, Bash
skills: go, postgresql, docker
name: test-engineer
model: inherit
---

# Test engineer — idx-api (Go)

## Run tests

```bash
GOFLAGS=-mod=mod go test ./...
go test ./internal/handler/... -run TestName
go test ./internal/queue/... -v
```

Integration DB tests: set `TEST_DATABASE_URL`.

## Patterns

- Table-driven tests in `*_test.go` next to source
- Fake upstream MLS/GIS HTTP in handler tests (no real Bridge/Spark calls)
- Queue round-trips: `internal/queue/queue_test.go`
- Middleware: `internal/api/middleware/*_test.go`

## Do not

- Use PHPUnit / `php artisan test` (removed)
- Hit production MLS keys in CI

See [README.md](../../README.md), [AGENTS.md](../../AGENTS.md).
