# idx-api HTTP API overview

This document summarizes the currently registered HTTP surface in the Go Fiber app (`internal/api/routes.go` + dashboard/static mounts).

## Canonical machine-readable API doc

- OpenAPI source file: `docs/yaak-api-collection.json`
- This repository does **not** currently register `/openapi.json` or `/swagger` routes in `internal/api/routes.go`.

## Route groups

- Infrastructure (public): `/healthz`, `/readyz`, `/health/replicas`, `/metrics`
- Marketing/static (public): `/`, `/static/*`
- API auth: `/api/auth/token`, `/api/auth/user`
- Versioned API: `/api/v1/*` (MLS proxy, GIS, search, comps, operations)
- Admin API (session-cookie protected): `/api/v1/admin/*`
- Image proxy: `/images/:listingKey/:photoId`
- Dashboard HTML/session flows: `/login`, `/logout`, `/dashboard/*`, `/invite/:token`

## Authentication model (code-aligned)

- `DomainToken` middleware protects `/api/v1/*` and `/images/*`.
- Token mode: `Authorization: Bearer <token>` with `idx:access` or `idx:full`, plus `X-Domain-Slug` header or `?domain=` query.
- Domain mode (no bearer): domain resolved from `X-Domain-Slug`, `?domain=`, or `Referer` host.
- `MLSAccess` middleware resolves allowed dataset/feed for `/api/v1/*` except GIS bypass paths.
- Admin API routes (`/api/v1/admin/*`) require dashboard `session_id` cookie via `SessionAuthMiddleware`.

## Token issuance

`POST /api/auth/token` accepts:

- `email` (string)
- `password` (string)

It returns a token payload:

- `token` (plaintext PAT)
- `abilities` (currently includes `idx:full`)

## Detailed endpoint reference

For a full route-by-route table (methods, auth, handlers, and behavior notes), see `docs/routes-reference.md`.
