---
name: go
description: |
  Manages Go 1.25+ runtime patterns, concurrency, error handling, and performance
  for the idx-api codebase. Use when: writing or reviewing Go code, adding handlers,
  services, repository methods, queue jobs, or scheduler tasks; debugging goroutine
  leaks, nil panics, or error propagation; structuring new internal/ packages;
  or optimizing PostgreSQL query patterns with pgx/sqlx.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Go Skill

Go 1.25+ backend for Quantyra IDX API — Fiber HTTP server, PostgreSQL+PostGIS storage, and a PostgreSQL-native job queue. Three processes (`api`, `worker`, `scheduler`) share one database; multi-DC safety via advisory locks. No ORM — raw SQL through `pgxpool` and `sqlx`.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving go, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.




Required wiring surfaces:
- runtime/infrastructure config: Dockerfile
- nearest typed request/context boundary
- handler/procedure boundary before external side effects

Side-effect barrier:
- Place guards before external APIs, auth mutations, email sends, analytics events, storage writes, and database mutations.


Fallback policy:
- Prefer provider-native/platform-managed primitives by default when no explicit override exists.
- Follow clear user/project overrides, but mention the native alternative and tradeoff.
- Fallbacks must be durable, multi-instance safe, and atomic under concurrency.

Verification rules:
- [error] native-or-explicit-override: Use the provider-native primitive first unless the user/project explicitly overrides it.
- [error] atomic-fallback: Fallback counters must be atomic under concurrency.

## Quick Start

### Verified Existing Pattern — Repository method

```go
// internal/repository/domain.go
func (r *DomainRepo) FindActiveBySlug(ctx context.Context, slug string) (*domain.Domain, error) {
    var d domain.Domain
    err := r.db.SQLX.GetContext(ctx, &d, `
        SELECT id, user_id, domain_slug, is_active
        FROM domains WHERE is_active = true AND LOWER(domain_slug) = LOWER($1) LIMIT 1`, slug)
    if errors.Is(err, sql.ErrNoRows) { return nil, nil }
    if err != nil { return nil, err }
    return &d, nil
}
```

### New Code Pattern — Add a service method

```go
// new code to add
func (s *Service) DoWork(ctx context.Context, id int64) (*Result, error) {
    row, err := s.repo.FindByID(ctx, id)
    if err != nil { return nil, fmt.Errorf("find by id %d: %w", id, err) }
    if row == nil { return nil, nil } // not found is not an error
    return &Result{Data: row}, nil
}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Error wrapping | `fmt.Errorf("…: %w", err)` everywhere | `fmt.Errorf("parse dsn: %w", err)` |
| Context propagation | Every function that does I/O takes `ctx context.Context` | `FindActiveBySlug(ctx, slug)` |
| Constructor DI | `New*` functions wire dependencies; no global state | `NewHandler(cfg, db, logger)` |
| Struct tags | `db:"col"` for sqlx, `json:"field"` for API | `Domain.ID int64 \`db:"id"\`` |
| Sentinel errors | `sql.ErrNoRows` checked with `errors.Is` | Repository returns `nil, nil` for not-found |

## Common Patterns

### Graceful shutdown with signal handling

**When:** Every `cmd/` entry point.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    cancel()
    _ = app.ShutdownWithTimeout(30 * time.Second)
}()
```

## See Also

- [patterns](references/patterns.md)
- [types](references/types.md)
- [modules](references/modules.md)
- [errors](references/errors.md)

## Related Skills

- See the **fiber** skill for routing, middleware, and Fiber-specific patterns
- See the **postgres** and **postgresql** skills for pgx/sqlx query patterns
- See the **queue-postgresql** skill for job enqueue, reserve, and lifecycle
- See the **cache-postgres** skill for proxy cache patterns
- See the **geospatial** skill for PostGIS queries