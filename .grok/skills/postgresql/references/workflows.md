# PostgreSQL Workflows Reference

## Contents
- Adding a New Migration
- Adding a New Repository Method
- Implementing a Bulk Data Operation
- Adding a PostGIS Spatial Query
- Debugging Queue Issues

## Adding a New Migration

Migrations use Goose SQL format in `migrations/`. The existing schema is a single file (`00001_initial.sql`).

### Checklist

Copy this checklist and track progress:
- [ ] Create new migration file: `migrations/NNNNN_descriptive_name.sql`
- [ ] Add `+goose Up` and `+goose Down` sections
- [ ] Test up: `GOOSE_DBSTRING="..." goose -dir migrations up`
- [ ] Test down: `GOOSE_DBSTRING="..." goose -dir migrations down`
- [ ] Verify schema with `\d table_name` in psql
- [ ] Update domain structs if adding columns (`internal/domain/`)

```sql
-- migrations/00002_add_new_column.sql

-- +goose Up
ALTER TABLE listings ADD COLUMN IF NOT EXISTS new_field VARCHAR(255);

-- Create index if column is queried in PostGIS search
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_listings_new_field
    ON listings (new_field) WHERE new_field IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_listings_new_field;
ALTER TABLE listings DROP COLUMN IF EXISTS new_field;
```

**Migration rules:**
- Use `IF NOT EXISTS` / `IF EXISTS` for idempotency
- Use `CREATE INDEX CONCURRENTLY` for large tables (non-blocking)
- Partial indexes with `WHERE` clause match query patterns (e.g., `WHERE standard_status IN ('Active', 'Pending')`)
- Run migrations against the **Patroni primary** only, not replicas

### WARNING: Don't modify existing migration files

**The Problem:** Goose tracks applied migrations by timestamp. Modifying an already-applied file causes schema drift between environments.

**The Fix:** Always create a new migration file. If in dev before first deploy, `goose down` then modify, but never after staging/production has run it.

## Adding a New Repository Method

Repositories live in `internal/repository/` and use the `repository.DB` wrapper. Follow the established pattern:

```go
// internal/repository/my_repo.go
type MyRepo struct {
    db *DB
}

func NewMyRepo(db *DB) *MyRepo {
    return &MyRepo{db: db}
}

// Use SQLX for single/multi-row struct scans
func (r *MyRepo) FindByID(ctx context.Context, id int64) (*domain.Thing, error) {
    var d domain.Thing
    err := r.db.SQLX.GetContext(ctx, &d, `
        SELECT id, name, status, created_at
        FROM things
        WHERE id = $1
    `, id)
    if err != nil {
        return nil, err
    }
    return &d, nil
}

// Use Pool for writes, transactions, or when not mapping to a struct
func (r *MyRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
    _, err := r.db.Pool.Exec(ctx, `
        UPDATE things SET status = $1, updated_at = NOW() WHERE id = $2
    `, status, id)
    return err
}
```

### DO/DON'T

| DO | DON'T |
|----|-------|
| Pass `context.Context` as first param | Use `db.SQLX.Query()` without `Context` |
| Return typed domain structs | Return raw `*sql.Rows` to callers |
| Use `$1, $2` parameterized placeholders | Interpolate values with `fmt.Sprintf` |
| Handle `pgx.ErrNoRows` / `sql.ErrNoRows` | Panic on query errors |

## Implementing a Bulk Data Operation

For operations processing hundreds or thousands of rows (MLS replication, batch deletes), use the chunked-transaction pattern from `listing_mirror.go`:

1. **Buffer records** in a slice
2. **Flush** when buffer hits chunk size (250 default)
3. Each flush is a **single transaction**
4. After all flushes, handle deletes separately

```go
// Based on internal/service/sync/listing_mirror.go:HydrateReplicaBatch
func ProcessBulk(ctx context.Context, db *repository.DB, records []Record) error {
    const chunkSize = 250
    var pending []Record

    flush := func() error {
        if len(pending) == 0 {
            return nil
        }
        tx, err := db.Pool.Begin(ctx)
        if err != nil {
            return err
        }
        defer tx.Rollback(ctx)
        for _, rec := range pending {
            if err := upsertRecord(ctx, tx, rec); err != nil {
                return err
            }
        }
        return tx.Commit(ctx)
    }

    for _, rec := range records {
        pending = append(pending, rec)
        if len(pending) >= chunkSize {
            if err := flush(); err != nil {
                return err
            }
            pending = nil
        }
    }
    return flush() // remaining records
}
```

**Why chunk:** A single transaction holding 50K rows causes lock contention and WAL bloat. 250-row transactions balance throughput with lock hold time.

### Feedback loop:

1. Make changes
2. Validate: `go test ./internal/service/sync/...`
3. If tests fail, fix issues and repeat step 2
4. Run against staging: verify `replica_pages` drains and `listings` populates
5. Check row counts match upstream: `SELECT COUNT(*) FROM listings WHERE dataset_slug = 'stellar'`

## Adding a PostGIS Spatial Query

PostGIS queries must use `geography` type (SRID 4326) for distance calculations. The `coordinates` column stores `geography(Point, 4326)`.

```go
// Radius search — from internal/service/search/postgis.go
meters := radiusMiles * 1609.34
query := `
    SELECT ... FROM listings
    WHERE dataset_slug = $1
      AND coordinates IS NOT NULL
      AND ST_DWithin(coordinates::geography, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, $4)
`
rows, err := db.Pool.Query(ctx, query, dataset, lng, lat, meters)
```

**Key points:**
- Always check `coordinates IS NOT NULL` before spatial functions — NULL geography crashes `ST_DWithin`
- Use `::geography` cast for meter-based distance (not `::geometry` which uses degrees)
- `ST_MakePoint(lng, lat)` — **longitude first**
- Existing GIST index on `coordinates` supports `ST_DWithin` efficiently

### WARNING: Don't use bounding-box math in Go

**The Problem:**
```go
// BAD — approximate, ignores curvature, no index usage
minLat := lat - radius/69.0
maxLat := lat + radius/69.0
db.Pool.Query(ctx, "SELECT * FROM listings WHERE latitude BETWEEN $1 AND $2", minLat, maxLat)
```

**The Fix:** Use PostGIS `ST_DWithin` on the `geography` column. It uses the GIST index and handles great-circle distance correctly.

## Debugging Queue Issues

The PostgreSQL job queue uses `jobs`, `job_batches`, and `failed_jobs` tables. Workers poll via `FOR UPDATE SKIP LOCKED`.

### Diagnostic queries

```sql
-- Pending/processing jobs by queue
SELECT queue, COUNT(*),
       COUNT(*) FILTER (WHERE reserved_at IS NOT NULL) AS reserved,
       COUNT(*) FILTER (WHERE reserved_at IS NULL) AS available
FROM jobs GROUP BY queue;

-- Stuck jobs (reserved but worker died)
SELECT id, queue, attempts, reserved_at,
       EXTRACT(EPOCH FROM NOW() - TO_TIMESTAMP(reserved_at))::int AS seconds_reserved
FROM jobs WHERE reserved_at IS NOT NULL
ORDER BY reserved_at ASC;

-- Recent failures
SELECT queue, exception, failed_at FROM failed_jobs ORDER BY failed_at DESC LIMIT 20;

-- Batch progress
SELECT id, name, total_jobs, pending_jobs, total_jobs - pending_jobs AS completed,
       finished_at IS NOT NULL AS is_done
FROM job_batches ORDER BY created_at DESC LIMIT 10;
```

### Purge legacy Laravel jobs (post-cutover only)

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
```

### Worker not picking up jobs

1. Verify `WORKER_QUEUES` env matches the queue names in `jobs`
2. Check `pg_notify` is working: `LISTEN idx_jobs_wakeup;` in psql, then trigger enqueue
3. Verify no stuck reservations (see diagnostic query above)
4. Worker logs should show `polling queues` or `reserved job`