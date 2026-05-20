---
description: Docker, Coolify, PostgreSQL queue workers, goose migrations, idx-images Nginx edge.
tools: Read, Edit, Write, Bash, Glob, Grep
skills: go, docker, postgresql
name: devops-engineer
model: inherit
---

# DevOps — idx-api (Go)

## Images

| Service | Dockerfile | Target | Port |
|---------|------------|--------|------|
| API | `Dockerfile` | `api` | 8000 |
| Worker | `Dockerfile` | `worker` | — |
| Scheduler | `Dockerfile` | `scheduler` | — |
| idx-images | `Dockerfile.idx-images` | — | 8080 |

## Env (all replicas)

- `DB_*`, `DB_SSLMODE=require` for remote Postgres
- `WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist`
- `BRIDGE_API_KEY`, `SPARK_ACCESS_TOKEN`
- `IDX_*_URL` public URLs

## Deploy steps

```bash
export GOOSE_DBSTRING="postgres://..."
make migrate
make seed-admin   # ADMIN_SEED_* in env
```

Purge legacy Laravel queue rows after cutover:

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
```

## Health

- API: `GET /healthz`, `GET /readyz`
- idx-images: proxies to api:8000

## Docs

- [docs/coolify-deployment.md](../../docs/coolify-deployment.md)
- [docs/deployment-operations.md](../../docs/deployment-operations.md)
- [docs/go-cutover.md](../../docs/go-cutover.md)

**Legacy:** FrankenPHP/Octane Dockerfiles are obsolete; use root `Dockerfile` Go targets.
