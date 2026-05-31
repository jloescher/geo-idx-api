# Cron Patterns Reference

## Contents
- Job Registration
- Leader Election with Advisory Locks
- Overlap Protection
- Fair Queue Distribution
- Job Lifecycle
- Anti-Patterns

## Job Registration

All cron jobs are registered inside `runAsLeader` (`internal/scheduler/scheduler.go`). This ensures only the leader enqueues work.

**Current job schedule:**

| Name | Cron (6-field) | Frequency | Queue | Type |
|------|----------------|-----------|-------|------|
| `coingecko` | `0 */10 * * * *` | Every 10 min | `Coingecko.Queue` | `crypto.refresh_pricing` |
| `mls-proxy-cache-purge` | `0 */15 * * * *` | Every 15 min | `default` | `mls.proxy_cache_purge` |
| `mls-kickoff` | `0 * * * * *` | Every minute | `default` | `mls.replication_kickoff` |
| `purge-replica` | `0 15 4 * * *` | Daily 04:15 | `default` | `mls.purge_replica_pages` |
| `purge-closed` | `0 5 3 * * *` | Daily 03:05 | `default` | `mls.purge_closed_listings` |
| `gis-probe` | `0 30 6 * * 1` | Monday 06:30 | `GIS.Queue` | `gis.probe_sources` |

**Cron format:** `robfig/cron/v3` with `cron.WithSeconds()` — 6 fields: `seconds minutes hours day-of-month month day-of-week`.

```go
// internal/scheduler/scheduler.go — job registration pattern
s.addJob(ctx, "mls-kickoff", "0 * * * * *", "default", queue.TypeMLSReplicationKickoff)
```

The `addJob` method wraps each job with `withoutOverlap` and logs errors on registration failure.

### Startup Kickoff

```go
// internal/scheduler/scheduler.go:90 — immediate first tick so workers show activity
s.enqueue(ctx, "mls-kickoff-startup", "default", queue.TypeMLSReplicationKickoff, nil)
```

Fires once at startup so the first replication cycle doesn't wait up to 60 seconds.

---

## Leader Election with Advisory Locks

### WARNING: Never run two schedulers without the advisory lock

**The Problem:** Two scheduler processes both register cron jobs and enqueue work, causing double-enqueue of every scheduled job.

**Why This Breaks:**
1. Replication kickoff fires twice per minute — duplicate HTTP requests to MLS APIs.
2. Cache purge runs concurrently — wasted database writes.
3. Purge jobs delete overlapping rows — potential data inconsistency.

**The Fix:**

```go
// internal/scheduler/leader.go — session-scoped advisory lock
conn, err := pool.Acquire(ctx)
err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&ok)
```

Key properties:
- **Session-scoped**: Lock held until the connection closes or `pg_advisory_unlock` is called.
- **Non-blocking**: `pg_try_advisory_lock` returns immediately with `false` if another session holds it.
- **Crash-safe**: If the leader process dies, PostgreSQL drops the connection and releases the lock automatically.
- **Dedicated connection**: The leader holds one pool connection for the lock lifetime. Do not release it until leadership ends.

### Leader Session Lifecycle

```go
// internal/scheduler/scheduler.go:Run
for {
    leader, ok, err := TryAcquireLeader(ctx, s.db.Pool, lockKey)
    if !ok {
        // Standby: poll every SCHEDULER_STANDBY_POLL_SECONDS
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(poll):
            continue
        }
    }
    // Leader: register cron, block until context cancelled
    runErr := s.runAsLeader(ctx)
    leader.Release(ctx) // unlock + return connection to pool
}
```

### Configuration

| Env var | Default | Purpose |
|---------|---------|---------|
| `SCHEDULER_LEADER_LOCK_ID` | `913374211` | Advisory lock key (int64) |
| `SCHEDULER_STANDBY_POLL_SECONDS` | `15` | Standby retry interval |

---

## Overlap Protection

### In-process (`withoutOverlap`)

```go
// internal/scheduler/scheduler.go:120
func (s *Scheduler) withoutOverlap(name string, fn func()) func() {
    return func() {
        if _, loaded := s.locks.LoadOrStore(name, true); loaded {
            s.logger.Debug("skipped overlapping run", "task", name)
            return
        }
        defer s.locks.Delete(name)
        fn()
    }
}
```

Prevents the same cron job from running twice within a single scheduler process (e.g., if an enqueue takes longer than the interval). Uses `sync.Map` — safe for concurrent cron callbacks.

**Limitation:** In-process only. Two separate scheduler instances without the advisory lock will both execute the same job concurrently.

### Database-level (replication guard)

The replication kickoff checks for active `replica_pages` rows before starting a new fetch. Combined with the advisory lock, this provides defense-in-depth against overlapping replication cycles. See the **queue-postgresql** skill for batch job patterns.

---

## Fair Queue Distribution

Workers use `ReserveFair` to round-robin across queues. Without it, `ORDER BY id ASC` always picks from the lowest-ID queue first — during heavy Bridge replication, `bridge-sync-fetch` would starve `spark-sync-fetch`. See the **queue-postgresql** skill for implementation details.

---

## Job Lifecycle

```
Scheduler → Enqueue (INSERT into jobs + pg_notify)
                ↓
Worker ← Reserve (FOR UPDATE SKIP LOCKED)
                ↓
Worker → process job
     ├── success → Delete from jobs + handleBatchComplete
     └── failure → Release (reset reserved_at, retry) or Fail (move to failed_jobs)
```

**Batch jobs:** `EnqueueBatch` creates a `job_batches` row + N child jobs. When each child completes, `CompleteBatchJob` decrements `pending_jobs`. At zero, the batch's `on_complete` job is enqueued automatically. This is how replication chains `fetch_page → persist_chunk → persist_finalize`. See the **queue-postgresql** skill.

---

## Anti-Patterns

### WARNING: Registering jobs outside runAsLeader

**The Problem:** Adding cron jobs before leader acquisition means standby schedulers also enqueue.

```go
// BAD — both schedulers register this, both enqueue
func New(cfg config.Config, ...) *Scheduler {
    s.cron.AddFunc("0 * * * * *", func() { s.enqueue(...) })
    return s
}
```

**Why This Breaks:** Standby schedulers that never acquire the lock still fire cron callbacks and enqueue duplicate jobs.

**The Fix:** Only register jobs inside `runAsLeader`, which runs after advisory lock acquisition.

### WARNING: Using time.Ticker for production scheduling

```go
// BAD — no overlap protection, no leader coordination, lost on crash
go func() {
    for range time.Tick(time.Minute) {
        doWork()
    }
}()
```

**Why This Breaks:** No overlap protection, no leader election, not durable across process restarts, no observability.

**The Fix:** Use `s.addJob()` to register a cron job that enqueues a typed queue job.

### WARNING: Ignoring job handler errors

```go
// BAD — job deleted, never retried
worker.Register(queue.TypeMyJob, func(ctx context.Context, job *queue.ReservedJob) error {
    if err := criticalWork(); err != nil {
        log.Println("oops") // logged but swallowed
        return nil          // job deleted, never retried
    }
    return nil
})
```

**The Fix:** Return the error so the worker releases the job for retry or moves it to `failed_jobs` after `maxAttempts`.