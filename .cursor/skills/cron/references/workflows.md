# Cron Workflows Reference

## Contents
- Adding a New Scheduled Job
- Debugging Double-Enqueue
- Scaling Workers for Heavy Replication
- Multi-DC Scheduler Deployment
- Handling Scheduler Failover

---

## Adding a New Scheduled Job

Copy this checklist and track progress:

- [ ] Step 1: Add job type constant to `internal/queue/payload.go`
- [ ] Step 2: Add cron schedule to `runAsLeader` in `internal/scheduler/scheduler.go`
- [ ] Step 3: Implement handler method on `Registry` in `internal/job/handlers.go`
- [ ] Step 4: Register handler in `RegisterAll` in `internal/job/registry.go`
- [ ] Step 5: Update `docs/INDEX.md` scheduled jobs table
- [ ] Step 6: Verify: run scheduler locally, check logs for `enqueued scheduled job`

### Step 1: Job type constant

```go
// internal/queue/payload.go
const TypeMyNewJob = "my.new_job"
```

### Step 2: Cron schedule

```go
// internal/scheduler/scheduler.go — inside runAsLeader
s.addJob(ctx, "my-job-name", "0 0 3 * * *", "default", queue.TypeMyNewJob)
```

Choose the queue name:
- `"default"` — lightweight, non-MLS work (purge, crypto, kickoff).
- A dedicated queue — isolated, high-volume work (e.g., `"my-queue"`).
- See the **queue-postgresql** skill for queue configuration.

### Step 3: Handler implementation

```go
// new code to add — internal/job/handlers.go
func (r *Registry) handleMyNewJob(ctx context.Context, job *queue.ReservedJob) error {
    r.logger.Info("running my new job", "job_id", job.ID)
    // Business logic — return error to trigger retry, nil to delete job
    return r.myService.DoWork(ctx)
}
```

### Step 4: Register in worker

```go
// internal/job/registry.go — add to RegisterAll
w.Register(queue.TypeMyNewJob, r.handleMyNewJob)
```

If the handler needs new service dependencies, add them to the `Registry` struct in `registry.go` and wire them in `deps.go`.

### Step 5: Update docs

Add a row to `docs/INDEX.md` scheduled jobs table with cron expression and purpose.

### Step 6: Verify

1. Run the scheduler: `make run-scheduler`
2. Validate: check logs for `cron schedules registered`
3. Wait for next tick, look for: `enqueued scheduled job` with task name
4. If no log appears, verify cron expression syntax (6 fields, seconds included)
5. If log appears but worker doesn't process, check `WORKER_QUEUES` includes the job's queue

---

## Debugging Double-Enqueue

**Symptoms:** Duplicate jobs in `jobs` table, MLS API rate limiting, `replica_pages` conflicts.

| Cause | Detection | Fix |
|-------|-----------|-----|
| Two schedulers, no advisory lock | Two log lines: `scheduler leader acquired` from different hosts | Set `SCHEDULER_LEADER_LOCK_ID=913374211` on both |
| Same lock ID, lock not held | Standby also logs `leader acquired` | Check `pg_locks` for the advisory lock |
| Job enqueued from API code | Duplicate `enqueued` logs not from scheduler | Search codebase for `queue.Enqueue` calls outside scheduler |

### Verification SQL

```sql
-- Check advisory lock is held
SELECT locktype, database, classid, objid, pid, mode, granted
FROM pg_locks WHERE locktype = 'advisory' AND objid = 913374211;

-- Count duplicate pending jobs by type
SELECT payload->>'type' AS job_type, COUNT(*) AS count
FROM jobs
WHERE reserved_at IS NULL
GROUP BY payload->>'type'
HAVING COUNT(*) > 1;
```

---

## Scaling Workers for Heavy Replication

During initial MLS seed or heavy catch-up, split workers by queue role:

```env
# default-worker (1 instance)
WORKER_QUEUES=default

# fetch-worker (2 instances) — MLS HTTP only
WORKER_QUEUES=bridge-sync-fetch,spark-sync-fetch

# persist-worker (2-4 instances) — Postgres upsert
WORKER_QUEUES=bridge-sync-persist,spark-sync-persist
```

**Why split:** Fetch workers do HTTP I/O (network-bound). Persist workers do `INSERT ... ON CONFLICT` (database-bound). Mixing them means one slow persist blocks the next fetch. See the **deploy-coolify** skill for deployment configuration.

**Tuning env vars (starting points):**

| Variable | Bridge | Spark |
|----------|--------|-------|
| `BRIDGE_SYNC_REPLICATION_TOP` | `2000` | — |
| `SPARK_SYNC_PERSIST_JOB_CHUNK` | — | `50` |

See the **queue-postgresql** skill for batch job configuration.

---

## Multi-DC Scheduler Deployment

Copy this checklist and track progress:

- [ ] Step 1: Set identical `SCHEDULER_LEADER_LOCK_ID` on both DC schedulers
- [ ] Step 2: Set `SCHEDULER_STANDBY_POLL_SECONDS=15` on both
- [ ] Step 3: Deploy both scheduler containers
- [ ] Step 4: Verify one logs `scheduler leader acquired`, other logs `scheduler standby`
- [ ] Step 5: Kill leader container, verify standby acquires lock within 15s
- [ ] Step 6: Check `pg_locks` for exactly one advisory lock row

### Key constraints

- Both schedulers must connect to the **same** PostgreSQL primary (via Patroni/Tailscale).
- Each scheduler holds one dedicated pool connection for the lock. Size the pool accordingly.
- Standby polling interval controls failover detection time. Lower = faster failover, higher = less standby load.
- Do NOT run three schedulers — the third wastes a pool connection in permanent standby.

See the **deploy-coolify** skill for the full 10-app multi-DC matrix.
See the **deploy-patroni** skill for database failover behavior.
See the **hosting-tailscale** skill for cross-DC connectivity.

---

## Handling Scheduler Failover

### Normal failover (leader crash)

1. Leader process terminates → PostgreSQL drops connection → advisory lock released.
2. Standby polls after `SCHEDULER_STANDBY_POLL_SECONDS` → `TryAcquireLeader` succeeds.
3. New leader logs `scheduler leader acquired`, registers cron, enqueues startup kickoff.
4. No duplicate enqueues during transition.

### Patroni failover during leader session

1. PostgreSQL primary switches → all connections reset → advisory lock released.
2. Old leader's `runAsLeader` returns with connection error → `leader.Release()` (no-op, already dropped).
3. Loop re-enters `TryAcquireLeader` → if new primary is ready, lock acquired. If not, temporary standby.
4. Workers polling `jobs` over Tailscale may see brief `readyz` timeouts. Retries handle this.

### WARNING: Do not use pg_advisory_lock (blocking variant)

```go
// BAD — blocks forever if another scheduler holds the lock
conn.QueryRow(ctx, `SELECT pg_advisory_lock($1)`, key)
```

**Why This Breaks:** The blocking variant waits indefinitely. If the leader is healthy, the standby goroutine is stuck forever with no way to log status or respond to shutdown signals.

**The Fix:** Always use `pg_try_advisory_lock` which returns immediately. The standby loop with `time.After(poll)` handles retries with proper context cancellation.