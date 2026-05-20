---
description: Queue throughput, replication fetch/persist scaling, PostGIS search, image cache, Fiber concurrency.
tools: Read, Grep, Glob, Bash
skills: go, postgresql, docker
name: performance-engineer
model: inherit
---

# Performance engineer — idx-api (Go)

## Hot paths

- MLS proxy JSON (Bridge/Spark HTTP)
- `POST /api/v1/search` — PostGIS + live merge
- Replication: `bridge-sync-fetch` / `spark-sync-fetch` (rate-limited upstream)
- Persist: `*-sync-persist` (parallel chunks)
- Image proxy + NVMe `IMAGE_CACHE_PATH`

## Scaling

- Multiple worker replicas; split fetch vs persist `WORKER_QUEUES`
- PostgreSQL `SKIP LOCKED` job reservation
- LISTEN/NOTIFY wakeup on enqueue

## Tuning env

- `BRIDGE_SYNC_PERSIST_JOB_CHUNK`, `SPARK_SYNC_PERSIST_JOB_CHUNK`
- `MLS_*` mirror window and page retention
- `GIS_EDGE_CACHE_TTL`, `LISTINGS_CACHE_TTL`

## Scheduler

`cmd/scheduler` enqueues only — workers must run to execute.

See [docs/coolify-deployment.md](../../docs/coolify-deployment.md).
