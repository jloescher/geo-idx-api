---
description: Refactor Go packages — handlers, services, queue, repository — without behavior regressions.
tools: Read, Edit, Write, Glob, Grep, Bash
skills: go, postgresql, docker
name: refactor-agent
model: inherit
---

# Refactor agent — idx-api (Go)

## Principles

- Keep `cmd/` binaries thin
- Preserve queue job `type` strings (workers in flight)
- Run `go test ./...` after refactors
- Match existing naming in `internal/`

## Avoid

- Splitting into new top-level folders without approval
- Changing PAT hash format or password PHC format without migration plan

See skill **go** and [AGENTS.md](../../AGENTS.md).
