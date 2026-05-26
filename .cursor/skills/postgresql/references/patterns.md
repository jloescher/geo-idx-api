# PostgreSQL Patterns Reference

## Contents
- Dual-Driver Architecture
- Transaction Pattern
- Bulk Upsert with ON CONFLICT
- Batch Coordinate Updates (PostGIS)
- Dynamic Query Builder
- Advisory Lock for Leadership
- Job Reservation (SKIP LOCKED)
- Anti-Patterns

## Dual-Driver Architecture

This project uses `pgxpool.Pool` for high-performance operations and `sqlx.DB` for convenient struct scanning. Both connect to the same database via `repository.DB` (`internal/repository/db.go`).

**When to use which:**

| Use `Pool` (pgx) | Use `SQLX` (sqlx) |
|-------------------|-------------------|
| Transactions (`Begin`) | Single-row struct scan (`GetContext`) |
| Bulk operations (`Exec` in loop) | Multi-row struct scan (`SelectContext`) |
| `pg_notify`, advisory locks | Repository reads mapping to domain structs |
| Any `pgx.Tx` operation | |

```go
// pgx — high perf, transactions, notifications
_, err := db.Pool.Exec(ctx, "INSERT INTO ... VALUES ($1)", val)
tx, _ := db.Pool.Begin(ctx)

// sqlx — convenient struct scanning
var d domain.Domain
err := db.SQLX.GetContext(ctx, &d, "SELECT * FROM domains WHERE id = $1", id)
```

## Transaction Pattern

Every transaction follows the same begin/rollback-defer/commit pattern:

```go
// From internal/queue/queue.go and internal/service/sync/listing_mirror.go
tx, err := db.Pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx) // no-op after Commit

// ... operations on tx, not db.Pool ...
_, err = tx.Exec(ctx, "UPDATE ... WHERE id = $1", id)
if err != nil {
    return err // rollback fires via defer
}
return tx.Commit(ctx)
```

**Key rule:** Always operate on `tx`, never on `db.Pool` mid-transaction. A `Pool` call is outside the transaction scope.

## Bulk Upsert with ON CONFLICT

The listing mirror uses `INSERT ... ON CONFLICT (dataset_slug, listing_key) DO UPDATE SET ...` for idempotent upserts (`internal/service/sync/listing_mirror.go`). JSONB columns use `COALESCE(EXCLUDED.media, listings.media)` to preserve existing data when the upstream doesn't include the collection.

```go
// From listing_mirror.go:upsertListing
_, err := tx.Exec(ctx, `
    INSERT INTO listings (dataset_slug, listing_key, ..., media, unit, room, open_house, ...)
    VALUES ($1, $2, ..., $43, $44, $45, $46, ...)
    ON CONFLICT (dataset_slug, listing_key) DO UPDATE SET
        raw_data = EXCLUDED.raw_data,
        media = COALESCE(EXCLUDED.media, listings.media),
        unit = COALESCE(EXCLUDED.unit, listings.unit),
        ...
`, rec.DatasetSlug, rec.ListingKey, ...)
```

**Why COALESCE:** Bridge `/Property/replication` doesn't return expanded nav collections. If `EXCLUDED.media` is NULL (absent from upstream), the existing `media` JSONB is preserved rather than wiped.

### WARNING: Don't upsert row-by-row outside a transaction

**The Problem:**
```go
// BAD — each upsert is its own implicit transaction; slow and not atomic
for _, rec := range records {
    upsertListing(ctx, db.Pool, rec) // individual implicit tx per call
}
```

**The Fix:**
```go
// GOOD — batch in a single transaction, flush at chunk boundary
tx, _ := db.Pool.Begin(ctx)
defer tx.Rollback(ctx)
for i, rec := range records {
    upsertListing(ctx, tx, rec) // use tx, not Pool
    if (i+1) % chunkSize == 0 {
        tx.Commit(ctx)
        tx, _ = db.Pool.Begin(ctx)
        defer tx.Rollback(ctx)
    }
}
tx.Commit(ctx)
```

## Batch Coordinate Updates (PostGIS)

Coordinates are updated in a separate batch pass after row upsert using `ST_SetSRID(ST_MakePoint(...), 4326)::geography`. Segments of 250 pairs are batched into a single `UPDATE ... FROM (VALUES ...)` statement.

```go
// From listing_mirror.go:flushCoordinates
for i := 0; i < len(pairs); i += 250 {
    segment := pairs[i:min(i+250, len(pairs))]
    // Builds: UPDATE listings SET coordinates = v.geom FROM (VALUES ($1, $2, ST_SetSRID(...)), ...) AS v(ds, k, geom) WHERE ...
    _, err := tx.Exec(ctx, sql, args...)
}
```

**Why separate from upsert:** PostGIS geography inserts are expensive. Isolating them allows the upsert to succeed even if coordinate geometry fails, and batches geometry work efficiently.

## Dynamic Query Builder

The PostGIS search builder (`internal/service/search/postgis.go`) constructs SQL with incremental `$N` parameter placeholders. This avoids string interpolation of values while keeping dynamic filter composition clean.

```go
q := "SELECT raw_data, media, ... FROM listings WHERE dataset_slug = $1"
args := []any{dataset}
n := 2 // next placeholder index

if req.MinPrice != nil {
    q += fmt.Sprintf(" AND list_price >= $%d", n)
    args = append(args, *req.MinPrice)
    n++
}
if req.Lat != nil && req.Lng != nil && req.RadiusMiles != nil {
    meters := *req.RadiusMiles * 1609.34
    q += fmt.Sprintf(` AND coordinates IS NOT NULL AND ST_DWithin(
        coordinates::geography,
        ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)`, n, n+1, n+2)
    args = append(args, *req.Lng, *req.Lat, meters)
    n += 3
}
q += fmt.Sprintf(" ORDER BY modification_timestamp DESC NULLS LAST LIMIT $%d OFFSET $%d", n, n+1)
args = append(args, limit+1, skip) // limit+1 for has-more detection
```

## Advisory Lock for Leadership

The scheduler uses `pg_try_advisory_lock` on a **dedicated, held-open connection** (`internal/scheduler/leader.go`). Session-scoped locks release automatically when the connection closes.

```go
// From internal/scheduler/leader.go
func TryAcquireLeader(ctx context.Context, pool *pgxpool.Pool, key int64) (*LeaderSession, bool, error) {
    conn, _ := pool.Acquire(ctx) // dedicated connection
    var ok bool
    conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&ok)
    if !ok {
        conn.Release()
        return nil, false, nil
    }
    return &LeaderSession{conn: conn, key: key}, true, nil
}
```

### WARNING: Don't use advisory locks on pooled connections without holding

**The Problem:**
```go
// BAD — lock is released when connection returns to pool
_, err := db.Pool.Exec(ctx, "SELECT pg_advisory_lock($1)", key)
// connection returns to pool → lock released → other schedulers grab it
```

**The Fix:** Hold a dedicated `pgxpool.Conn` for the lock lifetime. See the `LeaderSession` pattern above — the connection is only released in `Release()`.

## Job Reservation (SKIP LOCKED)

Workers claim jobs atomically with `FOR UPDATE SKIP LOCKED` inside a transaction. See the **queue-postgresql** skill for full queue patterns.

```go
// From internal/queue/queue.go:reserveFromQueues
tx, _ := c.pool.Begin(ctx)
defer tx.Rollback(ctx)
err = tx.QueryRow(ctx, `
    SELECT id, queue, payload, attempts FROM jobs
    WHERE queue = ANY($1) AND reserved_at IS NULL AND available_at <= $2
    ORDER BY id ASC FOR UPDATE SKIP LOCKED LIMIT 1
`, queues, now).Scan(&id, &queue, &payloadStr, &attempts)
```

## Anti-Patterns

### WARNING: Select-then-update without locks

**The Problem:**
```go
// BAD — race condition between SELECT and UPDATE
var count int
db.Pool.QueryRow(ctx, "SELECT pending_jobs FROM job_batches WHERE id = $1", id).Scan(&count)
if count == 1 {
    // Another worker could decrement between our SELECT and UPDATE
    db.Pool.Exec(ctx, "UPDATE job_batches SET pending_jobs = 0 WHERE id = $1", id)
}
```

**The Fix:**
```go
// GOOD — atomic decrement with condition
var pending int
db.Pool.QueryRow(ctx, `
    UPDATE job_batches SET pending_jobs = pending_jobs - 1
    WHERE id = $1 AND pending_jobs > 0
    RETURNING pending_jobs
`, id).Scan(&pending)
if pending == 0 { /* finalize */ }
```

### WARNING: Missing context on database calls

**The Problem:**
```go
// BAD — no timeout, no cancellation propagation
rows, err := db.SQLX.Queryx("SELECT ...", args...)
```

**The Fix:**
```go
// GOOD — context enables timeout and cancellation
rows, err := db.SQLX.QueryxContext(ctx, "SELECT ...", args...)
```