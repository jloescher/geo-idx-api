# Go Error Handling Reference

## Contents
- Error Wrapping Convention
- Not-Found Pattern
- Handler Error Mapping
- Queue Error Handling
- Sync/Replication Errors
- Common Pitfalls

## Error Wrapping Convention

Every error that crosses a package boundary is wrapped with `%w` and annotated with the operation that failed.

```go
// internal/repository/db.go
return nil, fmt.Errorf("parse dsn: %w", err)
return nil, fmt.Errorf("connect: %w", err)
return nil, fmt.Errorf("ping: %w", err)
return nil, fmt.Errorf("sqlx connect: %w", err)
```

### WARNING: Losing error context with `fmt.Errorf("…: %v", err)`

**The Problem:** Using `%v` instead of `%w` prints the error but destroys the chain. `errors.Is` and `errors.As` won't work on the result.

**Why This Breaks:** Callers further up the stack cannot inspect or match the root cause. Debugging requires reading log output instead of writing programmatic error checks.

**The Fix:** Always use `%w` when wrapping.

```go
// BAD — loses error chain
return fmt.Errorf("save domain: %v", err)

// GOOD — preserves error chain
return fmt.Errorf("save domain: %w", err)
```

## Not-Found Pattern

Repository methods return `nil, nil` when a row doesn't exist. The caller distinguishes "not found" from "error" by checking both values.

```go
// internal/repository/domain.go
func (r *DomainRepo) FindActiveBySlug(ctx context.Context, slug string) (*domain.Domain, error) {
    var d domain.Domain
    err := r.db.SQLX.GetContext(ctx, &d, `SELECT ... FROM domains WHERE ... LIMIT 1`, slug)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil // not found — not an error
    }
    if err != nil {
        return nil, err // actual error
    }
    return &d, nil
}
```

### WARNING: Returning an error for not-found

**The Problem:** Returning `fmt.Errorf("not found")` forces every caller to string-match the error message to distinguish not-found from real failures.

**Why This Breaks:** String matching is fragile. If the message changes, callers silently break. `errors.Is(sql.ErrNoRows, err)` is also wrong — it leaks SQL internals to the service layer.

**The Fix:** Return `nil, nil`. The service layer decides whether not-found is an error (404) or acceptable (optional relation).

## Handler Error Mapping

Handlers are the **only** place where errors become HTTP status codes. Services return plain errors; handlers translate.

```go
// internal/service/comps/service.go
func (s *Service) Run(c *fiber.Ctx) error {
    resp, err := s.engine.Run(c.Context(), feed, req)
    if err != nil {
        if isValidationErr(err) {
            return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
        }
        return fiber.NewError(fiber.StatusBadGateway, err.Error())
    }
    return c.JSON(resp)
}
```

### WARNING: Leaking internal errors to clients

**The Problem:** Returning `err.Error()` to the client when `err` contains SQL queries, file paths, or stack traces.

**Why This Breaks:** Information disclosure vulnerability. Attackers learn about your DB schema, file layout, and internal service names.

**The Fix:** Map errors to generic messages at the handler boundary. Include details in structured logs, not in HTTP responses.

```go
// BAD — leaks SQL query
return fiber.NewError(500, err.Error()) // "pq: duplicate key value violates unique constraint..."

// GOOD — generic message + structured log
s.logger.Error("comps run failed", "error", err)
return fiber.NewError(fiber.StatusBadGateway, "upstream service error")
```

## Queue Error Handling

Workers handle job failures via `Release` (retry) or `Fail` (dead-letter after max attempts).

```go
// internal/queue/worker.go
if err := w.process(ctx, job); err != nil {
    _ = w.client.Release(ctx, job, w.maxAttempts, err) // retry or fail
} else {
    _ = w.client.Delete(ctx, job.ID) // success — remove from queue
}
```

Jobs are retried with exponential backoff. After `maxAttempts`, the job moves to `failed_jobs`. See the **queue-postgresql** skill.

### WARNING: Panics in job handlers

**The Problem:** A nil pointer dereference or index out of range inside a job handler will crash the worker process.

**The Fix:** The worker should recover from panics per job:

```go
// new code to add — defensive panic recovery in worker loop
func (w *Worker) process(ctx context.Context, job *ReservedJob) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic in job %d: %v", job.ID, r)
        }
    }()
    // ... call registered handler
}
```

## Sync/Replication Errors

Bridge/Spark sync errors are logged and the cursor is NOT advanced, so the next kickoff retries from the last successful position.

```go
// internal/service/sync/bridge_sync.go — fetch page
result, err := s.fetchPage(ctx, fetchURL, query, dataset, true)
if err != nil {
    return PageResult{}, fmt.Errorf("fetch replication page: %w", err)
}
```

Key behaviors:
- HTTP 400 on Bridge `/replication` with timestamp filter → fallback to status-only filter
- Cursor stores `last_modification_timestamp` per dataset for incremental resume
- `replication_in_progress` flag prevents double kickoff

## Common Pitfalls

| Pitfall | Symptom | Fix |
|---------|---------|-----|
| `%v` instead of `%w` | `errors.Is` returns false | Always use `%w` |
| Non-pointer nullable scan | `sql: Scan error` on NULL column | Use `*string`, `*time.Time` |
| Returning error for not-found | Callers string-match | Return `nil, nil` |
| `err != nil` before `errors.Is` | Misses wrapped sentinel errors | Always use `errors.Is` first |
| Unhandled `sql.ErrNoRows` in `QueryRow` | Silent nil struct returned | Check for `sql.ErrNoRows` explicitly |
| Logging at wrong level | Important failures missed in prod noise | `Error` for failures, `Info` for lifecycle, `Debug` for skipping |

## Error Decision Flowchart

```
Error occurs
  ├─ Is it sql.ErrNoRows?
  │    └─ Return nil, nil (not found is not an error)
  ├─ Is it a validation error?
  │    └─ Return fiber.NewError(422, message)
  ├─ Is it an upstream API error?
  │    └─ Log details, return fiber.NewError(502, generic message)
  ├─ Is it in a job handler?
  │    └─ Return error → worker releases or fails the job
  └─ Is it an internal error?
       └─ Wrap with %w, return up the call stack
```