---
description: Debug Bridge proxy auth, queue workers, GIS cache, Spark replication, DB connectivity.
tools: Read, Grep, Glob, Bash
skills: go, postgresql, docker
name: debugger
model: inherit
---

# Debugger — idx-api (Go)

## Common issues

| Symptom | Check |
|---------|--------|
| `unknown job type type=""` | Laravel rows in `jobs` — purge per go-cutover |
| Spark jobs idle | `WORKER_QUEUES` includes `spark-sync-*` |
| Login 401 | `make seed-admin`; Argon2id hash in `users.password` |
| PAT 401 | Re-issue token from dashboard (SHA-256 storage) |
| DB SSL error | `DB_SSLMODE=require` for remote host |
| Static CSS missing | `/static/css/app.css` from embedded `internal/web` |

## Logs

- API/worker/scheduler: structured slog to stdout
- Run locally: `make run-api`, `make run-worker`

## Verify

```bash
curl -s http://127.0.0.1:8000/healthz
curl -s http://127.0.0.1:8000/readyz
```

See [docs/deployment-operations.md](../../docs/deployment-operations.md).
