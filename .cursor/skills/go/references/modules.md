# Go Modules Reference

## Contents
- Package Layout
- cmd/ Entry Points
- internal/ Layers
- Dependency Flow
- Adding a New Package

## Package Layout

```
idx-api/
├── cmd/api/main.go          # HTTP server
├── cmd/worker/main.go       # Queue consumer
├── cmd/scheduler/main.go    # Cron dispatcher
├── cmd/seed/main.go         # DB seeding
├── internal/
│   ├── api/routes.go        # Route registration
│   ├── config/config.go     # Env loading, Config struct
│   ├── domain/domain.go     # Shared types (Domain, User, APIToken)
│   ├── repository/          # DB access (one file per entity)
│   ├── handler/             # HTTP handlers (bridge/, gis/, auth/, images/)
│   ├── service/             # Business logic (search/, sync/, cache/, comps/, audit/)
│   ├── queue/               # Job queue client and worker
│   ├── job/registry.go      # Job type → handler mapping
│   ├── scheduler/           # Cron + advisory lock
│   ├── mlspoxy/             # MLS proxy client factory
│   └── web/static/          # Embedded dashboard assets
└── migrations/               # Goose SQL
```

## cmd/ Entry Points

Each binary is thin: load config, connect DB, wire dependencies, run.

```go
// cmd/api/main.go (simplified)
func main() {
    cfg := config.MustLoad()
    logger := newLogger(cfg)
    db, err := repository.New(ctx, cfg.DB)
    // ...
    app := fiber.New()
    api.RegisterRoutes(app, cfg, db, logger)
    logger.Info("api started", "port", cfg.App.Port)
    log.Fatal(app.Listen(fmt.Sprintf(":%d", cfg.App.Port)))
}
```

Three binaries, three `Dockerfile` targets: `api`, `worker`, `scheduler`. See the **docker** skill.

## internal/ Layers

Dependency direction is strictly one-way: `handler → service → repository`. Handlers never import `repository` directly except through `service` constructors. See the **fiber** skill for handler patterns.

| Layer | Owns | Never does |
|-------|------|-----------|
| `handler/` | HTTP parsing, Fiber error responses | SQL queries, business logic |
| `service/` | Business logic, orchestration | Direct HTTP access |
| `repository/` | SQL queries, scan to domain types | HTTP responses, business decisions |
| `domain/` | Pure type definitions | Any logic or I/O |
| `queue/` | Job lifecycle (enqueue, reserve, release) | Business logic |
| `job/` | Maps job types to service calls | SQL queries |

## Dependency Flow

```
cmd/api/main.go
  └→ api.RegisterRoutes(app, cfg, db, logger)
       └→ handler.New*(cfg, db, logger)
            └→ service.New*(cfg, db)
                 └→ repository.New*Repo(db)

cmd/worker/main.go
  └→ job.NewRegistry(cfg, db, logger)
       └→ queue.Worker.Register(type, handler)
            └→ handler calls service methods
```

### WARNING: Circular imports

**The Problem:** `handler` importing `repository` directly, or `service` importing `handler`.

**Why This Breaks:** Go compiler rejects circular imports. Even if it didn't, it creates tangled coupling that makes changes cascade unpredictably.

**The Fix:** Follow the layering strictly. If a handler needs repo data, go through a service. If two services need shared logic, extract to a third service in the same package or a shared sub-package.

## Repository Package

One file per entity. Constructor takes `*repository.DB`, returns a typed repo struct.

```go
// internal/repository/domain.go
type DomainRepo struct { db *DB }

func NewDomainRepo(db *DB) *DomainRepo { return &DomainRepo{db: db} }

func (r *DomainRepo) FindActiveBySlug(ctx context.Context, slug string) (*domain.Domain, error) { ... }
```

See the **postgres** skill for query patterns.

## Service Package

Services are organized by domain concern in subdirectories:

| Package | Purpose |
|---------|---------|
| `service/search/` | PostGIS + live MLS hybrid search |
| `service/sync/` | Bridge/Spark replication and persist |
| `service/cache/` | Proxy cache and lookup cache |
| `service/comps/` | BPO/home-value comparable analysis |
| `service/audit/` | Request audit logging |

## Handler Package

Organized by feature in subdirectories matching routes:

| Package | Routes | Notes |
|---------|--------|-------|
| `handler/bridge/` | `/api/v1/properties`, `/api/v1/search` | MLS proxy + mirror |
| `handler/gis/` | `/api/v1/gis` | Parcel geometry |
| `handler/auth/` | `/api/v1/auth/*` | Token issuance |
| `handler/images/` | `/images/*` | Photo proxy with NVMe cache |

See the **fiber** skill for middleware and routing.

## Queue Package

`internal/queue/` provides `Client` (enqueue) and `Worker` (consume). Job type registration happens in `internal/job/registry.go`.

```go
// internal/job/registry.go
func (r *Registry) RegisterAll(w *queue.Worker) {
    w.Register(queue.TypeNoop, r.handleNoop)
    w.Register(queue.TypeMLSReplicationKickoff, r.handleReplicationKickoff)
    w.Register(queue.TypeBridgeFetchPage, r.handleBridgeFetch)
    w.Register(queue.TypeBridgePersistChunk, r.handleBridgePersist)
}
```

See the **queue-postgresql** skill for job lifecycle details.

## Adding a New Package Checklist

```
- [ ] Create internal/<package>/ directory
- [ ] Define types (domain models or interfaces)
- [ ] Implement constructor: func New*(...) *Type
- [ ] Keep dependency direction: handler → service → repository
- [ ] Add route in internal/api/routes.go if HTTP-facing
- [ ] Add job type in internal/queue/ and handler in internal/job/ if background
- [ ] Run go test ./... to verify no import cycles
```