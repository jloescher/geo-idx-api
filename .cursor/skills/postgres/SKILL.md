---
name: postgres
description: |
  Manages PostgreSQL database connections, queries, migrations, and advanced patterns
  (pgx pool, sqlx, PostGIS, advisory locks, job queue, bulk upsert).
  Use when: writing repository methods, SQL queries, migration files, transaction blocks,
  bulk upsert patterns, PostGIS spatial queries, job queue operations, advisory lock logic,
  connection pool configuration, or any database interaction in idx-api.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# PostgreSQL Skill

Dual-driver PostgreSQL access via `pgxpool.Pool` (transactions, bulk ops, PostGIS) and `sqlx.DB` (simple reads). PostgreSQL-native job queue, advisory locks for multi-DC scheduler leadership, PostGIS spatial search, and gzip-compressed proxy cache — no Redis.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving postgres, verify against current docs FIRST:



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
- webhook/side-effect flow: Verify signatures before side effects and fail closed if the webhook secret is missing or verification fails. Claim provider event ids atomically, and avoid compensating-delete stranded-event failure modes. Add business-level idempotency for semantically duplicate events such as created/paid pairs for the same purchase. Use transactions, conditional updates, unique constraints, or status-based claims so retries can safely reprocess after rollback. Keep response behavior explicit for duplicate, ignored, verified, and failed events.



Required wiring surfaces:
- runtime/infrastructure config: Dockerfile
- nearest typed request/context boundary
- handler/procedure boundary before external side effects

Side-effect barrier:
- Place guards before external APIs, auth mutations, email sends, analytics events, storage writes, and database mutations.

Webhook idempotency contract:
- Verify provider signatures before any side effect and fail closed when the secret is missing or verification fails.
- Make the provider event claim atomic and retry-safe; prefer a transaction or explicit pending/processed/failed status over a fallible compensating delete.
- Add business-level idempotency for distinct provider events that represent the same purchase, subscription, membership, or entitlement transition.
- Use conditional writes (for example `WHERE field IS NULL RETURNING`), unique constraints, unique indexes, or upserts instead of read-then-update gates.
- If memberships, links, or ownership rows are created, enforce duplicate prevention at the database level, not only in application code.


Fallback policy:
- Prefer provider-native/platform-managed primitives by default when no explicit override exists.
- Follow clear user/project overrides, but mention the native alternative and tradeoff.
- Fallbacks must be durable, multi-instance safe, and atomic under concurrency.

Verification rules:
- [error] native-or-explicit-override: Use the provider-native primitive first unless the user/project explicitly overrides it.
- [error] atomic-fallback: Fallback counters must be atomic under concurrency.
- [error] webhook-signature-fail-closed: Webhook handlers must verify signatures before side effects and fail closed when the secret is missing.
- [error] webhook-atomic-claim: Webhook event claims must be atomic and retry-safe; avoid fallible compensating delete flows.
- [error] webhook-business-idempotency: Distinct provider events for one purchase/member change need a business-level idempotency gate such as a conditional update or unique constraint.
- [warning] relational-uniqueness-invariant: Membership/link/ownership creation should use a database uniqueness invariant plus idempotent insert/upsert behavior.

## Quick Start

### Verified Existing Pattern — Dual Driver Setup

```go
// internal/repository/db.go — opens both pgx pool and sqlx wrapper
func New(ctx context.Context, cfg config.DBConfig) (*DB, error) {
    poolCfg, _ := pgxpool.ParseConfig(cfg.DSN())
    pool, _ := pgxpool.NewWithConfig(ctx, poolCfg)
    sqlxDB, _ := sqlx.Connect("pgx", cfg.DSN())
    return &DB{Pool: pool, SQLX: sqlxDB}, nil
}
```

### New Code Pattern — Repository Method

```go
// new code to add — use sqlx for single-row reads
func (r *MyRepo) FindByID(ctx context.Context, id int64) (*MyType, error) {
    var m MyType
    err := r.db.SQLX.GetContext(ctx, &m, `SELECT * FROM my_table WHERE id = $1`, id)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    return &m, err
}
```

## Key Concepts

| Concept | Driver | When to use |
|---------|--------|-------------|
| `db.SQLX.GetContext` | sqlx | Single-row reads (find by ID, slug) |
| `db.SQLX.SelectContext` | sqlx | Multi-row reads (list all) |
| `db.Pool.Exec` | pgx | Writes, simple updates |
| `db.Pool.Query` | pgx | Multi-row reads with scan loop |
| `db.Pool.QueryRow` | pgx | Single-row reads returning scalars |
| `tx.Exec` (pgx.Tx) | pgx | Transactional writes, bulk upsert |

## Common Patterns

### Transaction with Chunked Flush

**When:** Bulk persist with upsert + coordinate updates in one atomic unit.

```go
tx, err := w.db.Pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx)
for _, rec := range pending {
    if err := upsertListing(ctx, tx, rec); err != nil { return err }
}
return tx.Commit(ctx)
```

### UPSERT with COALESCE Preservation

**When:** Upsert that keeps existing JSONB when new value is null.

```sql
ON CONFLICT (dataset_slug, listing_key) DO UPDATE SET
    media = COALESCE(EXCLUDED.media, listings.media)
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- **go** — Go language patterns used throughout the codebase
- **fiber** — HTTP framework that routes to repository/handler code
- **postgresql** — PostgreSQL administration and configuration
- **queue-postgresql** — Job queue built on `jobs` table with `FOR UPDATE SKIP LOCKED`
- **cache-postgres** — `mls_search_cache` gzip TTL caching
- **geospatial** — PostGIS spatial queries and coordinate handling
- **deploy-patroni** — Multi-DC Patroni cluster for shared PostgreSQL