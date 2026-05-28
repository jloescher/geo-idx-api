# Quantyra IDX API

High-performance MLS proxy and image delivery service for Quantyra IDX, written in **Go 1.25+** with **Fiber**, **PostgreSQL + PostGIS**, and a **PostgreSQL-backed job queue** (no Redis). Bridges multiple MLS feeds including Bridge Data Output (Stellar) and Spark Platform (Beaches) through a unified API.

## Tech Stack

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Runtime | Go | 1.25+ | High-performance HTTP server and background workers |
| Framework | Fiber | v2 | Fast HTTP router with middleware support |
| Database | PostgreSQL + PostGIS | Latest | Primary storage and geospatial queries |
| Queue | PostgreSQL | Latest | Job queue for background processing (no Redis) |
| Logger | slog | stdlib | Structured logging with JSON/text output |
| Build | CGO_ENABLED=0 | - | Single binary deployment with no runtime deps |

## Quick Start

**Prerequisites:** Go 1.25+, PostgreSQL with PostGIS, `.env` from `.env.example`.

```bash
# Clone and setup
git clone [repository]
cd idx-api
cp .env.example .env
# Edit DB_*, BRIDGE_API_KEY, SPARK_ACCESS_TOKEN, etc.

# Database setup
export GOOSE_DBSTRING="postgres://postgres:postgres@127.0.0.1:5432/idx_api?sslmode=disable"
make migrate
make seed-admin   # ADMIN_SEED_EMAIL / ADMIN_SEED_PASSWORD in .env

# Run services (separate terminals)
# API server
make run-api

# Worker (processes jobs from queue)
export WORKER_QUEUES=default,sync-kickoff,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
make run-worker

# Scheduler (kicks off replication and background tasks)
make run-scheduler
```

## Production database (workspace `.env`)

**The checked-in workspace `.env` is configured for the production PostgreSQL database** (`DB_*` / `GOOSE_DBSTRING`). Treat every local `make run-api`, `make run-worker`, `make run-scheduler`, `psql`, Goose, and integration test that loads `.env` as operating against **live production data**.

**Agents and developers MUST NOT add, update, or delete production data** unless the user explicitly requests a specific production change in that turn. That includes:

- No `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE`, or destructive DDL
- No `make migrate`, `make seed-admin`, replication kickoffs, queue workers, schedulers, or `docs/scripts/*` backfill runners unless the user clearly asks to run them on production
- No ad hoc SQL that mutates rows or schema

**Allowed without explicit approval:** read-only investigation (`SELECT`, `EXPLAIN`, schema/catalog queries), code changes, and tests that do not connect to production (unit tests, mocks). Prefer a local or staging DSN when you need to run migrations, seeds, or write tests.

**Do not edit `.env`** to repoint the database unless the user asks you to.

For Patroni backfills and Goose `00006`–`00008`, use a dedicated DSN file (e.g. `docs/scripts/.env.backfill.local`) and the runbooks in `docs/production-data-backfill.md` — not the workspace `.env` — unless the user explicitly directs otherwise.

## Project Structure

```
idx-api/
├── cmd/                  # Entry points
│   ├── api/             # HTTP server (:8000)
│   ├── worker/          # Queue consumer
│   ├── scheduler/       # Cron dispatcher
│   └── seed/           # Database seeding
├── internal/            # Application code
│   ├── api/            # HTTP routes and middleware
│   ├── config/         # Configuration management
│   ├── handler/        # HTTP handlers (bridge, gis, auth, etc.)
│   ├── mlspoxy/        # MLS proxy implementations
│   ├── repository/     # Data access layer
│   ├── service/        # Business logic (audit, cache, search, etc.)
│   ├── scheduler/      # Distributed scheduling with PostgreSQL locks
│   └── web/           # Embedded static assets
├── migrations/         # Goose SQL schema
├── docs/              # Documentation
└── bin/              # Built binaries (make build)
```

## Architecture Overview

The system consists of three main processes:

1. **API Server** (`cmd/api`) - HTTP endpoints for MLS proxy, GIS, search, and dashboard
2. **Worker** (`cmd/worker`) - Processes jobs from PostgreSQL queue (fetch MLS data, persist, etc.)
3. **Scheduler** (`cmd/scheduler`) - Distributed cron with PostgreSQL advisory locks for multi-DC safety

### Key Components

| Module | Location | Purpose |
|--------|----------|---------|
| MLS Proxy | `internal/handler/bridge` | RESO web API proxy for Bridge/Spark MLS |
| GIS Service | `internal/handler/gis` | Parcel data proxy with teaser tiers |
| Image Proxy | `internal/handler/images` | MLS photo proxy with filesystem cache |
| Queue System | `internal/queue` | PostgreSQL job queue with fair work distribution |
| Replication | `internal/service/sync` | MLS data mirroring with chunked persistence |
| Cache Layer | `internal/service/cache` | Multi-level caching (proxy, lookup, etc.) |
| Authentication | `internal/handler/auth` | Domain + API token auth with audit logging |

## Platform-Native Production Patterns

Before implementing production behavior, identify the runtime, hosting platform, database, queue, storage, auth, payment, analytics, and email systems involved. Inspect the provider service catalog, official docs, runtime config, and project docs before choosing a fallback implementation.

For changes touching abuse protection, rate limits, background work, scheduled jobs, queues, caching, shared state, secrets, file/object storage, database connectivity, webhooks, payments, auth/session flows, email sending, analytics events, or externally visible side effects:

1. Prefer managed/platform-native primitives over in-process memory, local timers, singleton clients, ad hoc counters, or frontend-only controls.
2. Wire platform capabilities through the repository's infrastructure/config layer, runtime environment, and typed app/context boundary.
3. Place guards before expensive or externally visible side effects such as payment APIs, auth mutations, email sends, analytics events, storage writes, or database mutations.
4. Preserve privacy and anti-enumeration behavior in auth, recovery, invite, checkout, and email flows.
5. Decide and document the failure stance: fail open, fail closed, retry, or degrade gracefully.
6. Check concurrency, retries, serverless/edge isolates, transaction boundaries, and multi-instance behavior before choosing a storage or coordination pattern.
7. Keep the change consistent with the repo's existing deployment/runtime setup rather than introducing a parallel mechanism.

Precedence: follow a clear user instruction first, then explicit project docs, then provider best practices. When a fallback is explicitly required, state the provider-native alternative and make the chosen path durable, multi-instance safe, and atomic under concurrency. Do not present module-scope mutable state, frontend-only checks, detached timers, untyped env access, or non-atomic select-then-update counters as production-ready.

This Go application is designed for cloud deployment with these patterns:

- **Managed primitives**: Uses PostgreSQL advisory locks for scheduler leadership, PostgreSQL-native job queue instead of Redis, platform object storage for images
- **Multi-DC safety**: Scheduler uses PostgreSQL advisory locks (913374211) for leader election across multiple data centers
- **Graceful degradation**: Fallback to cached data when MLS APIs are unavailable
- **State boundaries**: All state persisted to PostgreSQL; no in-process state that would fail in serverless/isolated environments
- **Atomic operations**: Uses transactions for job processing, data persistence, and cache updates

## UI/UX Quality Contract

For frontend, mobile, desktop, CLI, form, dashboard, onboarding, account/settings, or visual polish tasks:

1. Inspect nearby screens/components, the component library, design tokens, and existing density before creating new structure or styles.
2. Choose a surface-appropriate direction: dashboard/tooling should be quiet, dense, and scannable; marketing can be more memorable; CLI/Ink should prioritize stable layout, truncation, and keyboard clarity.
3. Avoid generic AI slop, template-looking screens, random gradient/card stacks, and UI that ignores the product context.
4. For changed interactive flows, define the state matrix before coding: loading, empty, error, disabled, pending, success, retry/recovery, and long-text cases.
5. Verify accessibility basics: labels, focus states, keyboard path, semantic controls, contrast, and non-hover-only guidance.
6. Keep UX distinct from product strategy: UX covers concrete journeys, states, affordances, microcopy, and accessibility; product strategy covers activation, adoption, experiments, and metrics.

## Development Guidelines

### Database safety

When `.env` points at production (see [Production database](#production-database-workspace-env) above), default to **read-only** database access. Do not run mutating commands or scripts against that DSN.

### Code Style
- **File naming**: kebab-case for Go files (`bridge/handler.go`)
- **Code naming**:
  - Structs: PascalCase (`Handler`, `Service`)
  - Functions: camelCase with verb prefix (`NewHandler`, `fetchListings`)
  - Variables: camelCase with context prefix (`ctx`, `cfg`, `db`)
  - Constants: SCREAMING_SNAKE_CASE (`MAX_RETRIES`)
  - Interfaces: PascalCase with -able suffix (`ProxyClient`, `CacheStore`)
- **Import order**: 1) Standard library, 2) Third-party, 3) Internal, 4) Domain, 5) Relative
- **Error handling**: Always return errors with context; use slog.Error for structured logging

### Build & Test
```bash
make build          # Build all binaries
make test           # Run all tests
make fmt            # Format code
make lint           # Run golangci-lint
```

### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `DB_*` | Yes | PostgreSQL connection | `DB_HOST=127.0.0.1` |
| `BRIDGE_API_KEY` | Yes | Bridge MLS API key | `sk-...` |
| `SPARK_ACCESS_TOKEN` | Yes | Spark API access token | `...` |
| `ADMIN_SEED_*` | Yes | Bootstrap admin credentials | `admin@example.com` |
| `WORKER_QUEUES` | Yes | Queue names for worker | `default,sync-kickoff,bridge-sync-fetch,...` |
| `SCHEDULER_LEADER_LOCK_ID` | No | PostgreSQL lock key for scheduler | `913374211` |

## Available Commands

| Command | Description |
|---------|-------------|
| `make run-api` | Start HTTP server on port 8000 |
| `make run-worker` | Start queue worker |
| `make run-scheduler` | Start scheduler process |
| `make migrate` | Run database migrations (**not** against production `.env` unless explicitly requested) |
| `make seed-admin` | Create admin user from env vars (**not** against production `.env` unless explicitly requested) |
| `make build` | Build all binaries to bin/ |
| `make test` | Run all tests with coverage |

## Database Schema

The application uses PostgreSQL with PostGIS. Key tables:

- `domains` - Authorized domains and API keys
- `tokens` - Active API tokens with scopes
- `jobs` - PostgreSQL job queue (Laravel parity)
- `replica_pages` - Staged MLS data before persistence
- `listings` - Final mirrored listings with geospatial data
- `audit_logs` - API access and change tracking

## Multi-MLS Support

The system supports multiple MLS feeds through a unified interface:

- **Bridge Data Output** (Stellar) - RESO OData feed
- **Spark Platform** (Beaches) - RESO OData feed with custom extensions
- **Dataset routing** - `?dataset=stellar|beaches` parameter support

## GIS Integration

Built-in GIS parcel proxy with:

- PostGIS spatial queries
- Teaser tiering for authenticated vs public access
- County/primary data source separation
- Bounding box limits for performance

## Authentication & Authorization

- **Domain-based auth**: Validates request hostname against allowed domains
- **API tokens**: Token-based authentication with scopes
- **Audit logging**: All authenticated requests logged
- **MLS access control**: Per-feed authorization checks

## Testing

- Unit tests: Handler functions and business logic
- Integration tests: Database operations and HTTP endpoints
- E2E tests: Full replication workflows (when available)
- Test pattern: `*_test.go` files co-located with implementation

## Deployment

Docker multi-target build for separate services:

- `Dockerfile.api` - HTTP server with health checks
- `Dockerfile.worker` - Queue consumer
- `Dockerfile.scheduler` - Cron dispatcher
- Supports Coolify/Dokploy deployment with multi-DC PostgreSQL (Patroni + Tailscale)

## Additional Resources

- @README.md - Full project overview and setup
- @docs/INDEX.md - Complete documentation index
- @docs/listings-mirror.md - MLS replication details
- @docs/go-cutover.md - Migration from Laravel
- @docs/coolify-deployment.md - Multi-DC deployment guide


## Skill Usage Guide

When working on tasks involving these technologies, invoke the corresponding skill:

| Skill | Invoke When |
|-------|-------------|
| go | Manages Go 1.25+ runtime patterns and performance optimizations |
| fiber | Configures Fiber v2 HTTP router with middleware support |
| postgresql | Manages PostgreSQL database operations with PostGIS extensions |
| postgres | Manages PostgreSQL database connections and queries |
| docker | Configures Docker multi-target builds for API, worker, and scheduler services |
| frontend-design | Applies UI design with Tailwind CSS and component styling patterns |
| deploy-coolify | Manages Coolify deployments with multi-DC PostgreSQL replication |
| ux | Improves dashboard flows, authentication paths, and API error states |
| deploy-docker | Configures Docker builds and container orchestration patterns |
| deploy-patroni | Manages PostgreSQL Patroni cluster configuration and failover |
| hosting-tailscale | Configures Tailscale networking for multi-DC connectivity |
| hosting-coolify | Configures Coolify server deployments and resource allocation |
| storage-s3 | Configures S3-compatible object storage for image caching |
| queue-postgresql | Manages PostgreSQL job queue with fair work distribution |
| auth-api-token | Manages API token authentication with audit logging |
| cache-postgres | Manages PostgreSQL-based cache for MLS proxy responses |
| proxy-web | Configures web proxy for MLS API integration with caching and rate limiting |
| geospatial | Manages PostGIS spatial queries and geographic data processing |
| auth-domain | Manages domain-based authentication and access control |
| cron | Configures distributed cron scheduling with PostgreSQL advisory locks |
| scoping-feature-work | Breaks features into MVP slices and acceptance criteria |
| prioritizing-roadmap-bets | Ranks initiatives using impact, effort, and risk signals |
| designing-onboarding-paths | Designs onboarding paths, checklists, and first-run UI |
| mapping-user-journeys | Maps in-app journeys and identifies friction points in code |
| improving-activation-flow | Optimizes activation steps and time-to-value milestones |
| crafting-empty-states | Creates empty states and onboarding affordances |
| orchestrating-feature-adoption | Plans feature discovery, nudges, and adoption flows |
| designing-inapp-guidance | Builds tooltips, tours, and contextual guidance |
| running-product-experiments | Sets up product experiments and rollout checks |
| instrumenting-product-metrics | Defines product events, funnels, and activation metrics |
| triaging-user-feedback | Routes feedback into backlog and quick wins |
| writing-release-notes | Drafts release notes tied to shipped features |
| structuring-offer-ladders | Frames plan tiers, value ladders, and upgrade logic |
| clarifying-market-fit | Aligns ICP, positioning, and value narrative for on-page messaging |
| framing-release-stories | Builds launch narratives, assets, and rollout checklists |
| embedding-decision-cues | Applies behavioral cues that improve conversion decisions |
| crafting-page-messaging | Writes conversion-focused messaging for pages and key CTAs |
| generating-growth-hypotheses | Generates channel experiments and growth loops |
| designing-lifecycle-messages | Designs onboarding and lifecycle email sequences |
| tightening-brand-voice | Refines copy for clarity, tone, and consistency |
| planning-editorial-arcs | Defines content themes, briefs, and editorial cadence |
| orchestrating-social-rhythm | Plans social content beats and distribution rhythm |
| tuning-landing-journeys | Improves landing page flow, hierarchy, and conversion paths |
| streamlining-signup-steps | Reduces friction in signup and trial activation |
| accelerating-first-run | Improves onboarding sequence and time-to-value |
| reducing-form-falloff | Improves lead capture forms to reduce drop-off |
| refining-prompt-surfaces | Optimizes banners, modals, and in-app prompts |
| strengthening-upgrade-moments | Improves upgrade prompts and paywall messaging |
| mapping-conversion-events | Defines funnel events, tracking, and success signals |
| designing-variation-tests | Plans A/B experiments and measurement plans |
| calibrating-paid-campaigns | Aligns paid acquisition with landing pages and pixels |
| building-acquisition-tools | Designs lead magnets or free tools for acquisition |
| engineering-referral-loops | Designs referral or partner loop mechanics |
| inspecting-search-coverage | Audits technical and on-page search coverage |
| scaling-template-pages | Builds scalable, template-driven search pages |
| adding-structured-signals | Adds structured data for rich results |
| building-compare-hubs | Creates comparison and alternative pages for discovery |

## Prompt-Aware Production Contract

Before coding, scan the user's prompt for relevant skills and production-risk signals.

- Load or inspect the relevant skill when the task matches a skill name, its Use when description, or nearby technology terms.
- Production-risk signals include: abuse/rate-limit guard, background/lifecycle work, scheduled/recurring work, cache/shared state, secrets/env wiring, database/concurrency, webhook/side-effect flow, email/external side effect, API/auth flow.
- For those signals, create a short task contract before coding: likely skills, provider docs to inspect, preferred native service, wiring surfaces, side-effect barriers, fallback policy, and verification criteria.
- Infer provider/runtime from repository evidence even when the user does not name it. If a repo uses Cloudflare Workers via Alchemy, an abuse/rate-limit prompt should consider Cloudflare runtime capabilities before a DB counter.
- Inspect provider service catalogs, best-practice docs, and runtime/database/config surfaces before choosing code.
- If the user's prompt clearly asks for a different mechanism, follow the user and mention the provider-recommended alternative plus the tradeoff.
- If project docs clearly mandate a different mechanism, follow project docs and preserve their constraints.
- Otherwise prefer the platform-recommended/native primitive before in-memory, frontend-only, detached async, or ad hoc counter solutions.
- Place guards before external side effects and document failure behavior.
- If a fallback is used, make it durable, multi-instance safe, and atomic under concurrency; non-atomic select-then-update counters are not production-safe.

## UI/UX Quality Contract

For frontend, mobile, desktop, CLI, form, dashboard, onboarding, account/settings, or visual polish tasks:

- UI/UX signals include: UI/interface change, form/flow UX, state coverage, accessibility/interaction quality, responsive layout, conversion/onboarding flow.
- Load or inspect frontend-design for visual/interface craft and ux for journeys, state coverage, microcopy, and interaction quality when those skills exist.
- Inspect nearby screens/components, the component library, design tokens, and current density before creating a new visual direction.
- Choose a surface-appropriate direction: dashboard/tooling should be quiet, dense, and scannable; marketing can be more memorable; CLI/Ink should prioritize stable layout, truncation, and keyboard clarity.
- Avoid generic AI slop, template-looking screens, random gradient/card stacks, and UI that ignores the product context.
- For changed interactive flows, define the state matrix before coding: loading, empty, error, disabled, pending, success, retry/recovery, and long-text cases.
- Verify accessibility basics: labels, focus states, keyboard path, semantic controls, contrast, and non-hover-only guidance.
- The final hook check is advisory and may warn about missing UI states, responsive constraints, or accessibility cues without blocking completion.
