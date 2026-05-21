# PostgreSQL Workflows Reference

## Contents
- Adding a New Repository
- Writing a Migration
- Bulk Data Import with Chunked Upsert
- Job Queue Reserve-and-Process
- Scheduler Leadership Acquisition
- Migration Verification Checklist

---

## Adding a New Repository

Copy this checklist and track progress:

- [ ] Define struct in `internal/domain/` with db tags
- [ ] Create `internal/repository/my_thing.go` with `MyThingRepo` struct holding `*DB`
- [ ] Use `db.SQLX.GetContext` / `SelectContext` for reads
- [ ] Use `db.Pool.Exec` for writes
- [ ] Return `(nil, nil)` on `sql.ErrNoRows`
- [ ] Add constructor `NewMyThingRepo(db *DB) *MyThingRepo`

### Template

```go
// new code to add
package repository

type MyThingRepo struct {
    db *DB
}

func NewMyThingRepo(db *DB) *MyThingRepo {
    return &MyThingRepo{db: db}
}

func (r *MyThingRepo) FindByID(ctx context.Context, id int64) (*mything.Thing, error) {
    var t mything.Thing
    err := r.db.SQLX.GetContext(ctx, &t, `
        SELECT id, name, created_at FROM my_things WHERE id = $1
    `, id)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    return &t, err
}
```

---

## Writing a Migration

Migrations live in `migrations/` using Goose SQL format. Single schema file: `00001_initial.sql`.

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS my_things (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX my_things_name_idx ON my_things (name);

-- +goose Down
DROP TABLE IF EXISTS my_things;
```

Run: `export GOOSE_DBSTRING="postgres://..." && make migrate`

### DO: Use IF NOT EXISTS for idempotent migrations

The project runs against shared databases (multi-DC Patroni). Migrations must be re-runnable.

---

## Bulk Data Import with Chunked Upsert

This is the primary pattern for MLS replication. See `internal/service/sync/listing_mirror.go`.

### Workflow

1. Accumulate records in a slice
2. When slice reaches `upsertChunk` (default 250), flush:
   - `db.Pool.Begin(ctx)` → transaction
   - Loop: `upsertListing(ctx, tx, rec)` per record
   - `flushCoordinates(ctx, tx, coords)` — batch PostGIS update
   - `flushNullCoordinates(ctx, tx, nullCoordKeys)` — null out stale coords
   - `tx.Commit(ctx)`
3. Final flush for remaining records
4. Batch deletes for non-Active/Pending rows

```go
// new code to add — chunked flush skeleton
flush := func() error {
    if len(pending) == 0 { return nil }
    tx, err := db.Pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)
    for _, rec := range pending {
        if err := upsertRow(ctx, tx, rec); err != nil { return err }
    }
    return tx.Commit(ctx)
}
for _, item := range items {
    pending = append(pending, item)
    if len(pending) >= chunkSize {
        if err := flush(); err != nil { return err }
    }
}
if err := flush(); err != nil { return err }
```

### WARNING: Don't flush one row at a time

Each `Begin`/`Commit` round-trips to PostgreSQL. Chunking 250 rows per transaction keeps latency bounded and avoids connection pool exhaustion.

---

## Job Queue Reserve-and-Process

See **queue-postgresql** skill for full details. Core pattern from `internal/queue/queue.go`:

1. `ReserveFair(ctx, queues, startIndex)` — rotates across queues, uses `FOR UPDATE SKIP LOCKED`
2. Process job
3. On success: `Delete(ctx, job.ID)`
4. On failure: `Release(ctx, job, maxAttempts, jobErr)` — returns to queue with delay
5. After max attempts: `Fail(ctx, job, jobErr)` — moves to `failed_jobs`

### DO: Use ReserveFair for multi-queue workers

```go
// Prevents Bridge backlog from starving Spark fetch
job, nextIdx, err := client.ReserveFair(ctx, queues, startIdx)
```

---

## Scheduler Leadership Acquisition

`internal/scheduler/leader.go` — PostgreSQL advisory lock pattern:

1. `TryAcquireLeader(ctx, pool, key)` — `pg_try_advisory_lock` on a dedicated `pgxpool.Conn`
2. If acquired: run cron jobs
3. If not: `WaitForLeader` polls every `SCHEDULER_STANDBY_POLL_SECONDS`
4. On shutdown: `leader.Release(ctx)` — unlocks and returns conn to pool

### WARNING: Lock is session-scoped

If the leader process dies, the connection closes and PostgreSQL releases the lock automatically. Do NOT store lock state in application memory — it must be PostgreSQL-managed.

---

## Migration Verification Checklist

After running migrations, verify indexed columns are populated:

```sql
-- From README.md — verify mirror data quality
SELECT COUNT(*) AS total,
       COUNT(list_price) AS with_price,
       COUNT(coordinates) AS with_geom
FROM listings WHERE dataset_slug = 'stellar';
```

Copy this checklist and track progress:
- [ ] `make migrate` completes without error
- [ ] `PostGIS_Version()` returns non-empty string
- [ ] Indexed columns populated (run verification query)
- [ ] `listings` GIST index exists on `coordinates`
- [ ] `jobs` table accessible by worker
- [ ] `GET /readyz` returns 200 (PostGIS check)

Validate: `make migrate && curl -s http://localhost:8000/readyz`
If validation fails, check `GOOSE_DBSTRING` matches `.env` `DB_*` values.

---

## Related Skills

- **queue-postgresql** — Job reservation, fair dispatch, NOTIFY/LISTEN
- **cache-postgres** — TTL gzip cache operations
- **geospatial** — PostGIS coordinate storage and spatial queries
- **go** — Go patterns for context propagation and error handling
- **deploy-patroni** — Multi-DC shared PostgreSQL topology