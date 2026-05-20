# Coolify — production and staging (Go)

Run **Quantyra IDX API** on [Coolify](https://coolify.io/) using the **[Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)** with [`Dockerfile`](../Dockerfile) targets **`api`**, **`worker`**, and **`scheduler`**, plus [`Dockerfile.idx-images`](../Dockerfile.idx-images).

Use **separate Coolify projects** for staging and production, each with its own PostgreSQL database.

**Queues:** PostgreSQL `jobs` table (no Redis). Deploy **web**, **worker(s)**, **scheduler**, and **idx-images**.

**Related:** [README.md](../README.md), [deployment-operations.md](deployment-operations.md), [go-cutover.md](go-cutover.md).

---

## 1. Applications per environment

| App | Dockerfile | Build target | Port / health |
|-----|------------|--------------|---------------|
| **idx-api-web** | `Dockerfile` | `api` | **8000** — `GET /healthz` |
| **idx-api-worker** | `Dockerfile` | `worker` | No HTTP — process health optional |
| **idx-api-scheduler** | `Dockerfile` | `scheduler` | No HTTP |
| **idx-images** | `Dockerfile.idx-images` | default | **8080** |

**Build context:** repository root (`.`).

**Runtime env:** Same `DB_*`, `BRIDGE_*`, `SPARK_*`, `WORKER_QUEUES`, and public URLs on web, worker, and scheduler.

---

## 2. Worker configuration

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

Optional split during heavy replication:

| Replica | `WORKER_QUEUES` |
|---------|-----------------|
| Fetch | `default,bridge-sync-fetch,spark-sync-fetch` |
| Persist | `bridge-sync-persist,spark-sync-persist` |

**Post-cutover:** purge legacy Laravel jobs once:

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
```

---

## 3. Post-deploy

```bash
# In a one-off container or locally with production DSN:
export GOOSE_DBSTRING="postgres://..."
goose -dir migrations postgres "$GOOSE_DBSTRING" up
# Or: make migrate

# Admin login (from ADMIN_SEED_* env on seed job / local .env):
make seed-admin
```

Notify customers to **re-issue API keys** from `/dashboard` after Go cutover.

---

## 4. idx-images

[`Dockerfile.idx-images`](../Dockerfile.idx-images), port **8080**. Upstream **`idx-api:8000`** on the shared Docker network.

---

## 5. Resources (starting points)

| Service | CPU | RAM |
|---------|-----|-----|
| Web (`api`) | 0.5–1.0 | 512–1024 MB |
| Worker | 0.25–0.5 each | 512–1024 MB |
| Scheduler | 0.1–0.25 | 256–384 MB |

Reserve host memory for PostgreSQL if co-located.

---

## 6. Spark / Bridge outbound

Workers and web need HTTPS to Bridge and Spark hosts (`BRIDGE_HOST`, `SPARK_REPLICATION_HOST`, `SPARK_API_HOST`). See [spark/idx-api-integration.md](spark/idx-api-integration.md).

---

## 7. Local smoke build

```bash
docker build -f Dockerfile --target api -t idx-api:local .
docker build -f Dockerfile --target worker -t idx-api-worker:local .
docker run --rm -p 8000:8000 --env-file .env idx-api:local
```

---

## Legacy note

Older docs referenced FrankenPHP/Octane (`Dockerfile.production`, `php artisan queue:work`). The **current** stack is **Go binaries** in [`Dockerfile`](../Dockerfile). Remove FrankenPHP base image variables from Coolify if migrating an existing project.
