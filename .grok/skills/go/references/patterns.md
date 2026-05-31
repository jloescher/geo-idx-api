# Go Patterns Reference

## Contents
- Error Handling Conventions
- Concurrency Patterns
- Dependency Injection
- Database Query Patterns
- Queue Job Patterns
- Configuration Loading

## Error Handling Conventions

Always wrap errors with context using `%w`. Return `nil, nil` for not-found (not an error). Map to Fiber errors at the handler boundary only.

```go
// GOOD — wrap with context at every return
return nil, fmt.Errorf("find domain %s: %w", slug, err)

// GOOD — not-found is nil, nil
if errors.Is(err, sql.ErrNoRows) { return nil, nil }

// BAD — silent discard
_, _ = s.repo.Save(ctx, item) // error silently ignored
```

### WARNING: Ignoring errors with `_`

**The Problem:** Using `_ = someFunc()` discards the error return. If the operation fails, you'll never know.

**Why This Breaks:** The only acceptable use is `defer res.Body.Close()` or similar cleanup where the error is truly irrelevant. In production, silently discarded errors cause data loss and hours of debugging.

**The Fix:** Handle the error or wrap and return it. If discarding is intentional, add a comment explaining why.

## Concurrency Patterns

### Context cancellation

Every I/O function takes `ctx context.Context` as the first parameter. Check `ctx.Err()` in loops.

```go
// internal/scheduler/scheduler.go — leader loop
for {
    if ctx.Err() != nil { return ctx.Err() }
    leader, ok, err := TryAcquireLeader(ctx, s.db.Pool, lockKey)
    // ...
}
```

### WARNING: Goroutine leaks

**The Problem:** Launching `go func()` without a mechanism to stop it. The goroutine runs forever, holding references to everything it captured.

**Why This Breaks:** Memory leaks. In a long-running worker/scheduler, leaked goroutines accumulate until OOM.

**The Fix:** Always pass a context and select on `ctx.Done()` inside the goroutine.

```go
// BAD
go func() {
    for { doWork() } // never stops
}()

// GOOD
go func() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C: doWork()
        }
    }
}()
```

### Sync.Map for in-process locks

The scheduler uses `sync.Map` for `withoutOverlap` — in-process only. This does NOT protect across multiple scheduler instances. For multi-DC, use PostgreSQL advisory locks. See the **queue-postgresql** skill.

## Dependency Injection

All components use constructor injection via `New*` functions. No global state, no `init()`.

```go
// internal/handler/bridge/handler.go
func NewHandler(cfg config.Config, db *repository.DB, auditor *audit.Logger, logger *slog.Logger) *Handler {
    return &Handler{
        cfg:        cfg,
        factory:    mlspoxy.NewFactory(cfg),
        proxyCache: cache.NewProxyCache(cfg, db),
        search:     search.NewService(cfg, db, cache.NewProxyCache(cfg, db)),
        logger:     logger,
    }
}
```

### WARNING: init() for complex logic

**The Problem:** `init()` functions run before `main()`, cannot be easily tested, and hide dependencies.

**The Fix:** Use explicit constructor functions. This project has zero `init()` functions by design.

## Database Query Patterns

The `DB` struct wraps `pgxpool.Pool` (complex ops, LISTEN/NOTIFY, advisory locks) and `sqlx.DB` (simple scans with struct tags). See the **postgres** skill for PostGIS details.

```go
// sqlx for simple SELECT → struct
err := r.db.SQLX.GetContext(ctx, &d, `SELECT ... FROM domains WHERE ...`, slug)

// pgx for bulk/batch/advanced
_, err := r.db.Pool.Exec(ctx, `UPDATE tokens SET last_used_at = NOW() WHERE id = $1`, id)
```

### WARNING: N+1 queries

**The Problem:** Querying in a loop instead of batching.

**Why This Breaks:** 200 listings × 1 query = 200 round trips. This project uses chunked upserts (`BuildListingRecord`) to batch inserts. Follow that pattern.

## Queue Job Patterns

Jobs are enqueued via `queue.Client.Enqueue` and processed by `queue.Worker.Run`. See the **queue-postgresql** skill for full lifecycle.

```go
// Enqueue
_, _ = c.pool.Exec(ctx, "SELECT pg_notify($1, $2)", c.notifyChannel, queueName)

// Reserve with SKIP LOCKED — safe for multiple workers
job, err := w.reserveNext(ctx)
if job == nil { w.wait(ctx, ticker, notifyCh); continue }
```

## Configuration Loading

`internal/config/config.go` uses helper functions for type-safe env parsing:

```go
env(key, def string) string          // string with default
envBool(key string, def bool) bool   // "true"/"1" parsing
envInt(key string, def int) int      // integer parsing
envDuration(key string, def time.Duration) time.Duration // "30s" or seconds as int
```

## Structured Logging

`slog` throughout — never `fmt.Println` or `log.Printf`.

```go
logger.Info("scheduler leader acquired")
logger.Error("reserve job", "error", err)
logger.Debug("skipped overlapping run", "task", name)
```

## Copyable Checklist — Adding a New Feature

```
- [ ] Define types in internal/domain/ or alongside service
- [ ] Add repository methods in internal/repository/
- [ ] Add service logic in internal/service/
- [ ] Add handler in internal/handler/
- [ ] Register route in internal/api/routes.go
- [ ] If background job: add type constant, handler in internal/job/registry.go
- [ ] If scheduled: add cron in internal/scheduler/scheduler.go
- [ ] Write tests alongside implementation (*_test.go)
- [ ] Run: go test ./...
```