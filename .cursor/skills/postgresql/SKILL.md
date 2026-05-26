---
name: postgresql
description: |
  Manages PostgreSQL database operations with PostGIS extensions for the Quantyra IDX API.
  Use when: writing SQL queries, creating migrations, implementing repository methods,
  configuring connection pools, using PostGIS spatial queries, implementing transactions,
  working with the PostgreSQL job queue (FOR UPDATE SKIP LOCKED, pg_notify),
  advisory locks for distributed coordination, bulk upsert/batch operations,
  or any database schema changes.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# PostgreSQL Skill

This project uses a **dual-driver PostgreSQL architecture** (`pgx/v5` pool + `sqlx` wrapper) with **PostGIS** for geospatial queries, a **PostgreSQL-native job queue** (no Redis), and **advisory locks** for multi-DC scheduler leadership. All state is PostgreSQL-backed — no in-process mutable state survives restarts.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving postgresql, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.

Risk signals this skill can participate in:
- cache/shared state: Avoid module-level mutable state for serverless or multi-instance code. Use a provider or database primitive with clear concurrency behavior.
- database/concurrency: Prefer atomic statements, unique constraints, transactions, or provider primitives for coordination. Avoid select-then-insert/update counters unless protected by a lock or constraint. For state flips, use conditional writes such as UPDATE ... WHERE field IS NULL RETURNING instead of read-then-update. For relation creation such as organization membership, add a database uniqueness invariant and an idempotent insert/upsert path.



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
- [warning] relational-uniqueness-invariant: Membership/link/ownership creation should use a database uniqueness invariant plus idempotent insert/upsert behavior.

## Quick Start

### Existing Pattern — Repository with dual drivers

```go
// pgx for high-performance operations (transactions, bulk, notifications)
_, err := r.db.Pool.Exec(ctx, "INSERT INTO ... VALUES ($1, $2)", a, b)

// sqlx for convenient struct scanning
err := r.db.SQLX.GetContext(ctx, &result, "SELECT ... WHERE id = $1", id)
```

### Existing Pattern — Transaction with rollback

```go
tx, err := w.db.Pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)
// ... operations on tx ...
return tx.Commit(ctx)
```

## Key Concepts

| Concept | Usage | Location |
|---------|-------|----------|
| `pgxpool.Pool` | Primary connection for transactions, bulk ops | `internal/repository/db.go` |
| `sqlx.DB` | Struct scanning via `GetContext`/`SelectContext` | `internal/repository/` |
| `ON CONFLICT DO UPDATE` | Upsert with COALESCE for nullable JSONB | `listing_mirror.go` |
| `FOR UPDATE SKIP LOCKED` | Concurrent-safe job claiming | `internal/queue/queue.go` |
| `pg_notify` | Worker wakeup on enqueue | `internal/queue/queue.go` |
| `pg_try_advisory_lock` | Multi-DC scheduler leader election | `internal/scheduler/leader.go` |
| PostGIS `geography` | Distance calculations in meters | `internal/service/search/postgis.go` |
| Goose migrations | `+goose Up/Down` SQL files | `migrations/` |

## Common Patterns

### Dynamic query builder with parameterized placeholders

```go
// From internal/service/search/postgis.go — parameterized, injection-safe
q := "SELECT ... FROM listings WHERE dataset_slug = $1"
args := []any{dataset}
n := 2
if req.MinPrice != nil {
    q += fmt.Sprintf(" AND list_price >= $%d", n)
    args = append(args, *req.MinPrice)
    n++
}
```

### WARNING: Never use string interpolation for values

**The Problem:**
```go
// BAD — SQL injection
q := fmt.Sprintf("SELECT * FROM listings WHERE city = '%s'", city)
```

**Why This Breaks:** SQL injection allows arbitrary query execution. Even "safe" inputs break on apostrophes (e.g., `Coeur d'Alene`).

**The Fix:**
```go
// GOOD — parameterized
q := "SELECT * FROM listings WHERE city = $1"
rows, err := db.Pool.Query(ctx, q, city)
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- **queue-postgresql** — job queue, FOR UPDATE SKIP LOCKED, pg_notify
- **geospatial** — PostGIS spatial queries and coordinate indexing
- **cache-postgres** — PostgreSQL-backed caching layer
- **deploy-patroni** — multi-DC PostgreSQL with Patroni over Tailscale
- **go** — Go-specific database patterns (pgx, sqlx)