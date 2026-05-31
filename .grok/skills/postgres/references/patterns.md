# PostgreSQL Patterns Reference

## Contents
- Connection and Driver Selection
- Error Handling
- UPSERT Patterns
- Bulk Coordinate Updates
- PostgreSQL Cache with Gzip
- Advisory Lock for Leader Election
- Dynamic Query Building

---

## Connection and Driver Selection

This project uses **two PostgreSQL drivers** in `repository.DB`:

| Field | Type | Use for |
|-------|------|---------|
| `db.Pool` | `*pgxpool.Pool` | Transactions, bulk ops, PostGIS, queue, advisory locks |
| `db.SQLX` | `*sqlx.DB` | Simple struct-scanning reads (`GetContext`, `SelectContext`) |

### DO: Use sqlx for struct-scanned reads

```go
// internal/repository/domain.go
var d domain.Domain
err := r.db.SQLX.GetContext(ctx, &d, `
    SELECT id, user_id, domain_slug FROM domains
    WHERE is_active = true AND LOWER(domain_slug) = LOWER($1)
    LIMIT 1
`, slug)
```

### DO: Use pgx for writes and transactions

```go
// internal/repository/token.go
_, err := r.db.Pool.Exec(ctx, `
    INSERT INTO personal_access_tokens (tokenable_type, tokenable_id, name, token, abilities, created_at, updated_at)
    VALUES ('App\Models\User', $1, $2, $3, $4, NOW(), NOW())
`, userID, name, HashToken(plain), abil)
```

### WARNING: Don't mix drivers in a transaction

pgx and sqlx use separate connection pools. A sqlx query will **not** see uncommitted pgx transaction state. Always use `pgx.Tx` for transactional work.

---

## Error Handling

### DO: Return nil for not-found with sql.ErrNoRows

```go
// internal/repository/domain.go
if errors.Is(err, sql.ErrNoRows) {
    return nil, nil  // not found = nil value, nil error
}
```

### DO: Use pgx.ErrNoRows for pgx queries

```go
// internal/queue/queue.go
if err == pgx.ErrNoRows {
    return nil, nil  // no jobs available
}
```

### WARNING: Don't swallow errors silently

Cache `Get` may return `nil, false, nil` on miss (errors treated as cache miss), but this is intentional — the caller falls through to live upstream. Everywhere else, surface errors.

---

## UPSERT Patterns

### INSERT ON CONFLICT with COALESCE preservation

Used in `internal/service/sync/listing_mirror.go` for listing upsert. JSONB nav collections (`media`, `unit`, `room`, `open_house`) use `COALESCE(EXCLUDED.col, listings.col)` to preserve existing data when the upstream payload omits them.

```go
// new code to add — upsert pattern
_, err := tx.Exec(ctx, `
    INSERT INTO listings (dataset_slug, listing_key, standard_status, raw_data, media, ...)
    VALUES ($1, $2, $3, $4, $5, ...)
    ON CONFLICT (dataset_slug, listing_key) DO UPDATE SET
        standard_status = EXCLUDED.standard_status,
        raw_data = EXCLUDED.raw_data,
        media = COALESCE(EXCLUDED.media, listings.media),
        updated_at = NOW()
`, rec.DatasetSlug, rec.ListingKey, ...)
```

### Cache UPSERT

`internal/service/cache/proxy_cache.go` — simple key-value cache with gzip:

```go
_, err = p.db.Pool.Exec(ctx, `
    INSERT INTO mls_search_cache (partition_key, fingerprint, compressed_data, last_updated)
    VALUES ($1, $2, $3, NOW())
    ON CONFLICT (partition_key, fingerprint) DO UPDATE
    SET compressed_data = EXCLUDED.compressed_data, last_updated = NOW()
`, partition, fingerprint, compressed)
```

---

## Bulk Coordinate Updates

`internal/service/sync/listing_mirror.go` — PostGIS geography points in batches of 250:

```go
// Build VALUES clause with ST_SetSRID(ST_MakePoint(lng, lat), 4326)::geography
parts = append(parts, fmt.Sprintf(
    "($%d::varchar, $%d::varchar, ST_SetSRID(ST_MakePoint($%d::float8, $%d::float8), 4326)::geography)",
    n, n+1, n+2, n+3),
)
// Update via join
sql := fmt.Sprintf(`
    UPDATE listings AS l SET coordinates = v.geom, updated_at = $1
    FROM (VALUES %s) AS v(ds, k, geom)
    WHERE l.dataset_slug = v.ds AND l.listing_key = v.k
`, strings.Join(parts, ","))
```

Null coordinates use `ANY($2)` with a Go string slice (pgx handles array expansion).

---

## Advisory Lock for Leader Election

`internal/scheduler/leader.go` — session-scoped `pg_try_advisory_lock` on a **dedicated pool connection**:

```go
conn, _ := pool.Acquire(ctx)
var ok bool
conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&ok)
// Lock held until conn.Release() or session ends
```

Key: `913374211` (configurable via `SCHEDULER_LEADER_LOCK_ID`). Two schedulers in multi-DC; one leads, one stands by.

---

## Dynamic Query Building

`internal/service/search/postgis.go` builds WHERE clauses with parameterized `$N` placeholders:

```go
q := `SELECT raw_data, media, unit, room, open_house, custom_fields FROM listings WHERE dataset_slug = $1`
args := []any{dataset}
n := 2
if req.MinPrice != nil {
    q += fmt.Sprintf(" AND list_price >= $%d", n)
    args = append(args, *req.MinPrice)
    n++
}
// PostGIS radius filter
if req.Lat != nil && req.Lng != nil && req.RadiusMiles != nil {
    meters := *req.RadiusMiles * 1609.34
    q += fmt.Sprintf(` AND ST_DWithin(coordinates::geography,
        ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)`, n, n+1, n+2)
    args = append(args, *req.Lng, *req.Lat, meters)
}
```

### WARNING: Never use string interpolation for values

```go
// BAD — SQL injection
q += fmt.Sprintf(" AND city = '%s'", city)

// GOOD — parameterized
q += fmt.Sprintf(" AND city = $%d", n)
args = append(args, city)
```

---

## Related Skills

- **queue-postgresql** — `FOR UPDATE SKIP LOCKED`, fair reservation, NOTIFY/LISTEN
- **cache-postgres** — TTL-based gzip cache with UPSERT
- **geospatial** — PostGIS coordinate storage and spatial queries
- **deploy-patroni** — Multi-DC shared PostgreSQL with advisory locks