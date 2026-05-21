---
name: queue-postgresql
description: |
  Manages a PostgreSQL-backed job queue with fair work distribution, batch processing,
  and NOTIFY-based wakeup. Built on pgx/v5 with FOR UPDATE SKIP LOCKED reservation.
  Use when: adding new job types, modifying queue behavior, debugging worker issues,
  implementing batch jobs, tuning WORKER_QUEUES, or touching internal/queue/ code.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Queue PostgreSQL Skill

PostgreSQL-native job queue (no Redis) using `FOR UPDATE SKIP LOCKED` for atomic reservation, `pg_notify` for instant worker wakeup, and round-robin `ReserveFair` to prevent one MLS feed from starving others. Laravel `jobs` table compatible for cutover parity.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving queue-postgresql, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.

Risk signals this skill can participate in:
- cache/shared state: Avoid module-level mutable state for serverless or multi-instance code. Use a provider or database primitive with clear concurrency behavior.
- database/concurrency: Prefer atomic statements, unique constraints, transactions, or provider primitives for coordination. Avoid select-then-insert/update counters unless protected by a lock or constraint. For state flips, use conditional writes such as UPDATE ... WHERE field IS NULL RETURNING instead of read-then-update. For relation creation such as organization membership, add a database uniqueness invariant and an idempotent insert/upsert path.



Required wiring surfaces:
- runtime/infrastructure config: Dockerfile
- nearest typed request/context boundary
- handler/procedure boundary before external side effects

Side-effect barrier:
- Place guards before external APIs, auth mutations, email sends, analytics events, storage writes, and database mutations.


Fallback policy:
- Prefer provider-native/platform-managed primitives by default when no explicit override exists.
- Follow clear user/project overrides, but mention the native alternative and tradeoff.
- Fallbacks must be durable, multi-instance safe, and atomic under concurrency.

Verification rules:
- [error] native-or-explicit-override: Use the provider-native primitive first unless the user/project explicitly overrides it.
- [error] atomic-fallback: Fallback counters must be atomic under concurrency.
- [warning] relational-uniqueness-invariant: Membership/link/ownership creation should use a database uniqueness invariant plus idempotent insert/upsert behavior.

## Quick Start

### Verified Existing Pattern — Enqueue

```go
// internal/queue/queue.go — single job
id, err := q.Enqueue(ctx, "bridge-sync-fetch", queue.TypeBridgeFetchPage, args, 0)
```

### Verified Existing Pattern — Register Handler

```go
// internal/job/registry.go
func (r *Registry) RegisterAll(w *queue.Worker) {
    w.Register(queue.TypeBridgeFetchPage, r.handleBridgeFetchPage)
}
```

### New Code Pattern — Add a Job Type

```go
// 1. Add constant in internal/queue/payload.go
const TypeMyNewJob = "my.new_job"

// 2. Add handler in internal/job/handlers.go
func (r *Registry) handleMyNewJob(ctx context.Context, job *queue.ReservedJob) error { … }

// 3. Register in internal/job/registry.go RegisterAll
w.Register(queue.TypeMyNewJob, r.handleMyNewJob)

// 4. Enqueue from service or scheduler
_, err := q.Enqueue(ctx, "default", queue.TypeMyNewJob, args, 0)
```

## Key Concepts

| Concept | Detail |
|---------|--------|
| `Client` | `pgxpool.Pool`-backed queue client (`internal/queue/queue.go`) |
| `Worker` | Poll + NOTIFY loop with handler registry (`internal/queue/worker.go`) |
| `Payload` | JSON envelope: `{"type":"bridge.fetch_page","args":{…}}` |
| `ReserveFair` | Round-robin across queues so no single feed monopolizes lowest IDs |
| `EnqueueBatch` | Atomic batch insert in one tx; `OnComplete` callback when all finish |
| `FOR UPDATE SKIP LOCKED` | Row lock that skips already-locked rows — safe for N workers on same table |
| `pg_notify` | Wakes workers instantly on enqueue; falls back to polling |
| Laravel compat | Legacy `CallQueuedHandler` payloads silently discarded (`internal/queue/laravel.go`) |

## Common Patterns

### Batch with Finalize Callback

```go
// internal/queue/queue.go — used by replication pipeline
batchID, err := q.EnqueueBatch(ctx, queue.BatchSpec{
    Name:  "bridge-persist-chunks",
    Queue: "bridge-sync-persist",
    Jobs:  chunkJobs,
    OnComplete: queue.BatchJob{
        Type: queue.TypeBridgePersistFinalize,
        Args: finalizeArgs,
    },
})
```

### Worker Bootstrap

```go
// cmd/worker/main.go pattern
q := queue.NewClient(db.Pool, cfg.Queue.Table, cfg.Queue.NotifyChannel, cfg.Queue.RetryAfter)
registry := job.NewRegistry(cfg, db, logger)
registry.InitServices(q)
worker := queue.NewWorker(q, cfg.Queue.WorkerQueues, cfg.Queue.PollInterval, logger)
registry.RegisterAll(worker)
worker.Run(ctx)
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **go** skill for Go concurrency and error handling conventions
- See the **postgresql** skill for PostgreSQL/PostGIS schema patterns
- See the **cache-postgres** skill for the proxy cache layer that interacts with this queue
- See the **deploy-coolify** skill for multi-DC worker/scheduler deployment