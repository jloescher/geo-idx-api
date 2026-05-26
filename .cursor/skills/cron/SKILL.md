---
name: cron
description: |
  Configures distributed cron scheduling with PostgreSQL advisory locks for
  multi-DC leader election. Uses robfig/cron/v3 with session-level
  pg_try_advisory_lock to ensure exactly one scheduler enqueues jobs across
  datacenters.
  Use when: adding scheduled jobs, modifying cron expressions, configuring
  scheduler leader election, adjusting replication kickoff timing, or debugging
  double-enqueue issues.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Cron Skill

Distributed cron scheduling backed by PostgreSQL advisory locks. One scheduler per datacenter holds a session-level `pg_try_advisory_lock`; the other stays standby. The leader registers cron jobs via `robfig/cron/v3` (6-field with seconds), each enqueuing typed jobs into the PostgreSQL `jobs` table for workers to process. No Redis, no external coordination service.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving cron, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.

Risk signals this skill can participate in:
- scheduled/recurring work: Use provider-managed schedules instead of request-time intervals. Make retries and idempotency explicit.



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

## Quick Start

### Verified Existing Pattern

```go
// internal/scheduler/scheduler.go — registering cron jobs inside runAsLeader
s.addJob(ctx, "mls-kickoff", "0 * * * * *", "default", queue.TypeMLSReplicationKickoff)
s.addJob(ctx, "purge-closed", "0 5 3 * * *", "default", queue.TypeMLSPurgeClosed)
s.cron.Start()
<-ctx.Done()
```

### New Code Pattern

```go
// new code to add — 3-file change for a new scheduled job
// 1. internal/queue/payload.go — add type constant
const TypeMyNewJob = "my.new_job"

// 2. internal/scheduler/scheduler.go — add to runAsLeader
s.addJob(ctx, "my-job", "0 0 2 * * *", "default", queue.TypeMyNewJob)

// 3. internal/job/registry.go — register handler in RegisterAll
w.Register(queue.TypeMyNewJob, r.handleMyNewJob)
```

## Key Concepts

| Concept | Usage | Location |
|---------|-------|----------|
| `robfig/cron/v3` | 6-field cron with seconds (`s m h dom mon dow`) | `scheduler.go:31` |
| `pg_try_advisory_lock` | Non-blocking session-scoped lock for leader election | `leader.go:31` |
| `withoutOverlap` | In-process `sync.Map` guard per job name | `scheduler.go:120` |
| `FOR UPDATE SKIP LOCKED` | Safe multi-worker job reservation | See **queue-postgresql** skill |
| `pg_notify` | Instant worker wakeup on enqueue | See **queue-postgresql** skill |
| `ReserveFair` | Round-robin across queues to prevent starvation | See **queue-postgresql** skill |

## Common Patterns

### Adding a new scheduled job

**When:** Product needs a periodic background task.

```go
// new code to add — full registration
// 1. internal/queue/payload.go
const TypeNightlyCleanup = "my.nightly_cleanup"

// 2. internal/scheduler/scheduler.go — inside runAsLeader
s.addJob(ctx, "nightly-cleanup", "0 0 3 * * *", "default", queue.TypeNightlyCleanup)

// 3. internal/job/registry.go — add handler method and register
w.Register(queue.TypeNightlyCleanup, r.handleNightlyCleanup)
```

### Multi-DC scheduler configuration

**When:** Deploying schedulers in multiple datacenters.

```env
SCHEDULER_LEADER_LOCK_ID=913374211
SCHEDULER_STANDBY_POLL_SECONDS=15
```

One scheduler logs `scheduler leader acquired`; the other logs `scheduler standby, waiting for leader lock`. On leader crash, standby acquires the lock within the poll interval.

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **go** skill for Go-specific patterns and conventions
- See the **queue-postgresql** skill for job queue implementation details
- See the **postgres** skill for PostgreSQL advisory lock internals
- See the **deploy-coolify** skill for multi-DC scheduler deployment topology
- See the **deploy-patroni** skill for database failover behavior with advisory locks