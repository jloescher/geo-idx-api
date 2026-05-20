---
description: PostgreSQL schema, goose migrations, PostGIS listings mirror, replica_pages, queue tables.
tools: Read, Edit, Write, Grep, Glob
skills: go, postgresql
name: data-engineer
model: inherit
---

# Data engineer — idx-api

## Schema

- **migrations/00001_initial.sql** — consolidated goose migration
- PostGIS `listings`, `replica_pages`, `listing_sync_cursors`
- Queue: `jobs`, `job_batches`, `failed_jobs`

## Commands

```bash
export GOOSE_DBSTRING="postgres://..."
make migrate
```

## Go data access

- `internal/repository/` — pgx + sqlx
- `internal/service/sync` — mirror persist
- `internal/service/search/postgis.go` — hybrid search

## Docs

- [docs/database-migrations.md](../../docs/database-migrations.md)

Do not reference `database/migrations/*.php` (removed).
