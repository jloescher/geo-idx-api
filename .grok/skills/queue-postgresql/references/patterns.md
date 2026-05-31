# Queue PostgreSQL — Patterns Reference

## Contents
- Reservation and Concurrency
- Fair Work Distribution
- Batch Processing
- Error Handling and Retries
- Laravel Compatibility
- Configuration
- Anti-Patterns

## Reservation and Concurrency

Jobs are claimed with `SELECT ... FOR UPDATE SKIP LOCKED` inside a transaction. This is the PostgreSQL-native pattern for safe multi-worker concurrent consumption — no application-level locking needed.

```go
// internal/queue/queue.go:188-244
func (c *Client) reserveFromQueues(ctx context.Context, queues []string) (*ReservedJob, error) {
    tx, err := c.pool.Begin(ctx)
    // …
    err = tx.QueryRow(ctx, `
        SELECT id, queue, payload, attempts FROM jobs
        WHERE queue = ANY($1)
          AND reserved_at IS NULL
          AND available_at <= $2
        ORDER BY id ASC
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    `, queues, now).Scan(&id, &queue, &payloadStr, &attempts)
    // …
    tx.Exec(ctx, `UPDATE jobs SET reserved_at = $1, attempts = attempts + 1 WHERE id = $2`, now, id)
    tx.Commit(ctx)
}
```

**Why SKIP LOCKED:** Two workers hitting the same table simultaneously will each claim different rows. Without `SKIP LOCKED`, the second worker would block until the first transaction commits — serializing all work.

### WARNING: SELECT then UPDATE without SKIP LOCKED

**The Problem:**

```go
// BAD — race condition: two workers can claim the same row
var id int64
tx.QueryRow(ctx, `SELECT id FROM jobs WHERE reserved_at IS NULL ORDER BY id LIMIT 1`).Scan(&id)
tx.Exec(ctx, `UPDATE jobs SET reserved_at = $1 WHERE id = $2`, now, id)
```

**Why This Breaks:** Two workers read the same row before either writes. Both reserve the same job. `FOR UPDATE SKIP LOCKED` is the only correct pattern for multi-consumer queues on PostgreSQL.

**The Fix:** Use `FOR UPDATE SKIP LOCKED` in the SELECT (as shown above).

## Fair Work Distribution

`ReserveFair` rotates a cursor across the configured queues so one feed cannot monopolize processing.

```go
// internal/queue/queue.go:170-186
func (c *Client) ReserveFair(ctx context.Context, queues []string, startIndex int) (*ReservedJob, int, error) {
    n := len(queues)
    for i := 0; i < n; i++ {
        idx := (startIndex + i) % n
        job, err := c.reserveFromQueues(ctx, []string{queues[idx]})
        if job != nil {
            return job, (idx + 1) % n, nil
        }
    }
    return nil, startIndex, nil
}
```

The worker stores the cursor and passes it on the next call:

```go
// internal/queue/worker.go:43-53
func (w *Worker) reserveNext(ctx context.Context) (*ReservedJob, error) {
    if len(w.queues) <= 1 {
        return w.client.Reserve(ctx, w.queues)
    }
    job, next, err := w.client.ReserveFair(ctx, w.queues, w.fairQueueCursor)
    w.fairQueueCursor = next
    return job, nil
}
```

**Queue split recommendation for scale** (from See the **deploy-coolify** skill):

| Worker deployment | `WORKER_QUEUES` | Role |
|-------------------|-----------------|------|
| default-worker (×1) | `default` | kickoff, purge, crypto, GIS |
| fetch-worker (×2) | `bridge-sync-fetch,spark-sync-fetch` | MLS HTTP only |
| persist-worker (×2–4) | `bridge-sync-persist,spark-sync-persist` | Postgres upsert |

### WARNING: Single Queue for All Job Types

**The Problem:** Putting fetch (I/O-heavy, slow) and persist (CPU-heavy, fast) on one queue means a fetch backlog blocks all persist work.

**The Fix:** Separate queues per pipeline stage. `ReserveFair` ensures even rotation when a worker listens on multiple queues.

## Batch Processing

Batches enable parallel chunk processing with a single completion callback (used by replication persist pipeline).

```go
// internal/queue/queue.go:91-153
batchID, err := q.EnqueueBatch(ctx, queue.BatchSpec{
    Name:  "bridge-persist-chunks",
    Queue: cfg.Bridge.SyncPersistQueue,
    Jobs:  []queue.BatchJob{
        {Type: queue.TypeBridgePersistChunk, Args: chunk1Args},
        {Type: queue.TypeBridgePersistChunk, Args: chunk2Args},
    },
    OnComplete: queue.BatchJob{
        Type: queue.TypeBridgePersistFinalize,
        Args: finalizeArgs,
    },
})
```

Completion is tracked atomically: `UPDATE job_batches SET pending_jobs = pending_jobs - 1 … RETURNING pending_jobs, options`. When `pending_jobs` hits zero, the `OnComplete` job is enqueued.

**Why one transaction:** Batch header and all child jobs are inserted atomically. If any insert fails, the whole batch rolls back — no orphaned partial batches.

## Error Handling and Retries

```go
// internal/queue/worker.go:82-88
if err := w.process(ctx, job); err != nil {
    w.logger.Error("job failed", "id", job.ID, "type", job.Payload.Type, "error", err)
    _ = w.client.Release(ctx, job, w.maxAttempts, err)
} else {
    _ = w.client.Delete(ctx, job.ID)
    w.handleBatchComplete(ctx, job)
}
```

| Outcome | Behavior |
|---------|----------|
| Success | `DELETE FROM jobs` (completed jobs are not retained) |
| Failure (below max) | `Release`: clear `reserved_at`, set `available_at` to now + `retryAfter` |
| Failure (at max) | `Fail`: insert into `failed_jobs`, then `DELETE FROM jobs` |
| Unknown type | Log warning, return error, trigger release/fail path |

Default `maxAttempts`: **3**. Default `retryAfter`: **120s** (`DB_QUEUE_RETRY_AFTER`).

### WARNING: Swallowing Errors in Handlers

**The Problem:**

```go
// BAD — worker thinks job succeeded, deletes it
func (r *Registry) handleMyJob(ctx context.Context, job *queue.ReservedJob) error {
    doWork() // error discarded
    return nil
}
```

**Why This Breaks:** Failed work is silently lost. No retry, no `failed_jobs` record. Always return errors from handlers.

**The Fix:**

```go
func (r *Registry) handleMyJob(ctx context.Context, job *queue.ReservedJob) error {
    if err := doWork(); err != nil {
        return fmt.Errorf("handleMyJob: %w", err)
    }
    return nil
}
```

## Laravel Compatibility

Legacy Laravel jobs (`CallQueuedHandler` payloads) are detected and silently discarded:

```go
// internal/queue/worker.go:92-100
if job.Payload.Type == "" && IsLegacyLaravelPayload(job.Raw) {
    w.logger.Info("discarded legacy Laravel queue job …")
    return nil // treated as success, deleted
}
```

Post-cutover cleanup SQL:
```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
```

## Configuration

| Env Variable | Default | Purpose |
|-------------|---------|---------|
| `DB_QUEUE_TABLE` | `jobs` | Table name |
| `WORKER_QUEUES` | `default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist` | Queues this worker polls |
| `QUEUE_POLL_INTERVAL_MS` | `1000` | Fallback poll when NOTIFY is silent |
| `DB_QUEUE_RETRY_AFTER` | `120` | Seconds before released job becomes available |
| `QUEUE_NOTIFY_CHANNEL` | `idx_jobs_wakeup` | `pg_notify` channel name |
| `SCHEDULER_LEADER_LOCK_ID` | `913374211` | Advisory lock key for multi-DC scheduler |

## Anti-Patterns

### WARNING: Module-Level Mutable State for Queue Coordination

**The Problem:**

```go
// BAD — breaks with multiple worker processes
var lastProcessedID int64
```

**Why This Breaks:** Multiple workers (separate processes) cannot share process memory. Use database primitives (`FOR UPDATE SKIP LOCKED`, advisory locks) for coordination.

**The Fix:** All coordination state lives in PostgreSQL (`jobs.reserved_at`, `jobs.attempts`, `job_batches.pending_jobs`, advisory locks).

### WARNING: Enqueueing Inside a Handler Without Queue Isolation

**The Problem:** A handler enqueues a new job to the same queue it's consuming — potential infinite loop if the new job triggers the same handler.

**The Fix:** Use dedicated queues per pipeline stage (fetch vs persist vs finalize) and set `WORKER_QUEUES` to only the stages that worker should consume.