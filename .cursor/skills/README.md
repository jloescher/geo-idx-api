# Quantyra GeoIDX — Cursor skills index

In-repo Cursor agent skills (`SKILL.md` per folder). **Primary runtime is Go** (see [README.md](../../README.md)).

## Invoke when

| Area | Skill folder |
|------|----------------|
| **Go API, workers, queue, handlers** | **`go`** |
| PostgreSQL / goose migrations | `postgresql` |
| Docker / Compose / Coolify | `docker` |
| Nginx (idx-images edge) | `nginx` |
| Bridge MLS + GIS search surfaces | `inspecting-search-coverage` |
| Dashboard HTML / embedded static UI | `frontend-design`, `crafting-empty-states`, `designing-inapp-guidance` |

### Legacy (pre–Go cutover — archived, do not use for new backend work)

Moved to **`_legacy/`** (Laravel, PHP, Fortify, Livewire, Pulse, Vite, Tailwind build, SQLite test stack, Stripe/Cashier). Use **`go`** and **`postgresql`** instead.

## Product surfaces (Go)

- **Marketing home**: `internal/handler/marketing` → `/`
- **Dashboard**: `internal/handler/dashboard` → `/login`, `/dashboard` (domains, TXT verify, API keys)
- **MLS / GIS API**: `internal/api/routes.go` — Bridge + Spark proxy, GIS, images, search
- **Auth middleware**: `internal/api/middleware/domain_token.go`, `mls_access.go`

See [AGENTS.md](../../AGENTS.md) and [docs/INDEX.md](../../docs/INDEX.md).
