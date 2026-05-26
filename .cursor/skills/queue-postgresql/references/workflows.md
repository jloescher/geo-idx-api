# Queue PostgreSQL — Workflows Reference

## Contents
- Adding a New Job Type
- Debugging Stuck or Failed Jobs
- Scaling Workers for Heavy Replication
- Migrating from Laravel Queue (Cutover)
- Inspecting Queue State

## Adding a New Job Type

Copy this checklist and track progress:

- [ ] 1. Add type constant in `internal/queue/payload.go`
- [ ] 2. Define args struct in `internal/job/handlers.go`
- [ ] 3. Implement handler function: `func (r *Registry) handleX(ctx context.Context, job *queue.ReservedJob) error`
- [ ] 4. Register in `internal/job/registry.go` `RegisterAll` method
- [ ] 5. If recurring: add cron entry in `internal/scheduler/scheduler.go`
- [ ] 6. Enqueue from service code or scheduler
- [ ] 7. Add the queue to `WORKER_QUEUES` if using a new queue name
- [ ] 8. Test: `go test ./internal/queue/... ./internal/job/...`

### Step-by-step

**1. Define the type** (`internal/queue/payload.go`):

```go
const TypeMyJob = "my.feature_job"
```

**2. Define args and handler** (`internal/job/handlers.go`):

```go
type myJobArgs struct {
    ResourceID string `json:"resource_id"`
}

func (r *Registry) handleMyJob(ctx context.Context, job *queue.ReservedJob) error {
    var args myJobArgs
    if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
        return fmt.Errorf("unmarshal my_job args: %w", err)
    }
    // Do work. Return error to trigger retry/fail.
    return r.myService.DoWork(ctx, args.ResourceID)
}
```

**3. Register** (`internal/job/registry.go`):

```go
w.Register(queue.TypeMyJob, r.handleMyJob)
```

**4. If recurring, add to scheduler** (`internal/scheduler/scheduler.go`):

```go
s.addJob(ctx, "my-feature", "0 */30 * * * *", "default", queue.TypeMyJob)
```

**5. Enqueue** (from any service with queue access):

```go
_, err := q.Enqueue(ctx, "default", queue.TypeMyJob, myJobArgs{ResourceID: "abc"}, 0)
```

**6. Validate:** `go test ./internal/queue/... ./internal/job/...`

Feedback loop:
1. Run `go test ./internal/queue/... ./internal/job/...`
2. If tests fail, fix and repeat
3. Run `go build ./cmd/worker` to confirm compilation
4. Deploy worker with updated `WORKER_QUEUES` if new queue added

## Debugging Stuck or Failed Jobs

### Symptoms and Actions

| Symptom | Diagnostic | Fix |
|---------|-----------|-----|
| Job not picked up | Check `WORKER_QUEUES` includes the job's queue | Add missing queue to env |
| Job picked up repeatedly | `attempts` exceeds `maxAttempts` (3) → `failed_jobs` | Check `failed_jobs.exception` for root cause |
| Workers idle but jobs exist | `reserved_at` set but worker died | `UPDATE jobs SET reserved_at = NULL WHERE reserved_at IS NOT NULL AND available_at < extract(epoch from now()) - 300` |
| Legacy Laravel jobs | Logs show "discarded legacy Laravel queue job" | `DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%'` |
| Batch stuck (no finalize) | `job_batches.pending_jobs > 0` but no child jobs in `jobs` | Check `failed_jobs` for failed children; manually set `pending_jobs = 0` or enqueue `OnComplete` |

### Key Queries

```sql
-- Active jobs by queue
SELECT queue, COUNT(*), MIN(attempts), MAX(attempts) FROM jobs GROUP BY queue;

-- Stalled reservations (reserved > 5 min)
SELECT id, queue, attempts, reserved_at FROM jobs
WHERE reserved_at IS NOT NULL AND available_at < extract(epoch from now()) - 300;

-- Recent failures
SELECT id, queue, exception, failed_at FROM failed_jobs ORDER BY failed_at DESC LIMIT 20;

-- Batch status
SELECT id, name, total_jobs, pending_jobs, failed_jobs, finished_at FROM job_batches WHERE finished_at IS NULL;
```

Feedback loop:
1. Run diagnostic query
2. Identify root cause from `exception` or `queue` mismatch
3. Fix code or config
4. Release stuck jobs: `UPDATE jobs SET reserved_at = NULL WHERE …`
5. Verify workers pick up released jobs

## Scaling Workers for Heavy Replication

During initial MLS seed or heavy catch-up, split workers by pipeline stage:

**Default (single worker, all queues):**
```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

**Scaled (dedicated workers):**

| Worker | `WORKER_QUEUES` | Replicas |
|--------|-----------------|----------|
| default | `default` | 1 |
| fetch | `bridge-sync-fetch,spark-sync-fetch` | 2 |
| persist | `bridge-sync-persist,spark-sync-persist` | 2–4 |

**Env tuning** (See the **deploy-coolify** skill for full table):

| Variable | Bridge | Spark |
|----------|--------|-------|
| `*_SYNC_REPLICATION_TOP` | 2000 | 1000 |
| `*_SYNC_PERSIST_JOB_CHUNK` | 50 | 50 |
| `*_SYNC_UPSERT_CHUNK` | 250 | 250 |

**Smoke after scale-up:**
```sql
-- At most one pending/processing replica_pages per provider+dataset
SELECT provider, dataset_slug, status, COUNT(*) FROM replica_pages
WHERE status IN ('pending','processing') GROUP BY 1,2,3;
```

Feedback loop:
1. Deploy scaled workers
2. Watch logs for interleaved `enqueued fetch` for `stellar` and `beaches`
3. Monitor `jobs` table depth: `SELECT queue, COUNT(*) FROM jobs GROUP BY queue;`
4. Adjust replica count if queue depth stays above chunk size

## Migrating from Laravel Queue (Cutover)

See the **deploy-coolify** skill and `docs/go-cutover.md` for full runbook.

Copy this checklist and track progress:

- [ ] 1. Deploy Go `api`, `worker`, `scheduler` against same PostgreSQL
- [ ] 2. Run `goose -dir migrations up` (idempotent)
- [ ] 3. Start workers → scheduler → API
- [ ] 4. Purge legacy jobs: `DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';`
- [ ] 5. Verify: `/healthz`, `/readyz`, `POST /api/v1/search`
- [ ] 6. Monitor: `GET /api/v1/bridge/stats` for replication status

**Important:** Go workers automatically discard legacy Laravel payloads — no manual filter needed. But purging them avoids polluting logs and queue scans.

## Inspecting Queue State

### Real-time monitoring

```sql
-- Current queue depth
SELECT queue, COUNT(*) AS pending,
       COUNT(*) FILTER (WHERE reserved_at IS NOT NULL) AS in_progress
FROM jobs GROUP BY queue;

-- Worker NOTIFY channel activity
SELECT * FROM pg_stat_notification_queue; -- if pg_notification_queue extension available

-- Advisory lock holder (scheduler leader)
SELECT locktype, database, pid, mode, granted
FROM pg_locks WHERE locktype = 'advisory' AND classid = 0 AND objid = 913374211;
```

### Health verification

```bash
# API health
curl -sf http://localhost:8000/healthz && echo "ok"

# Replication stats
curl -sf http://localhost:8000/api/v1/bridge/stats | jq .
```

Feedback loop:
1. Check queue depth query
2. If depth > 0 and not decreasing, check worker logs for errors
3. If workers are idle with depth > 0, check `reserved_at` for stalled reservations
4. Release stalled jobs and verify workers consume them