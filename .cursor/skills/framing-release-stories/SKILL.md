---
name: framing-release-stories
description: Builds launch narratives, assets, and rollout checklists
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
---

# Framing Release Stories Skill

A structured approach to preparing software releases for the Quantyra IDX API. This skill helps craft clear narratives, identify affected subsystems, generate deployment checklists, and coordinate cross-surface rollouts across Bridge MLS proxy, GHL Marketplace, GIS services, and billing components.

## Quick Start

1. **Identify scope** — Read recent commits (`git log --oneline -20`) and open PRs to understand what's shipping
2. **Map to subsystems** — Check which of the four primary systems are touched:
   - Bridge MLS Proxy (`/api/v1/*`, image proxy)
   - GHL Marketplace (OAuth, webhooks, widgets, lead sync)
   - GIS Parcel Proxy (`/api/v1/gis`, ArcGIS failover chain)
   - Billing (Stripe/Cashier, subscription tiers, checkout)
3. **Draft narrative** — Write release summary in three layers: user-facing (what changed), operator-facing (what to monitor), and compliance-facing (MLS/Stellar implications if any)
4. **Generate checklist** — Create environment-specific rollout steps for staging → production
5. **Tag assets** — Note which Docker images (`Dockerfile.idx-api`, `Dockerfile.idx-images`), migrations, and scheduled jobs require attention

## Key Concepts

**Subsystem boundaries** — The service has four independent subsystems (Bridge proxy, GHL Marketplace, GIS proxy, Billing). Releases should explicitly note which subsystems are modified to assess blast radius.

**Multi-surface deployment** — Production runs three public surfaces (idx.quantyralabs.cc, idx-api.quantyralabs.cc, idx-images.quantyralabs.cc). Releases may require coordinated deploys or sequential rollouts depending on contract changes between surfaces.

**Teaser vs full gating** — Any change to access control (DomainOrTokenAuth middleware, Sanctum abilities) must account for teaser-mode behavior affecting non-full-access clients.

**Cache invalidation strategy** — GIS uses generation-based invalidation; Bridge listings use 15-minute TTL cache. Releases modifying data shapes must include cache clearing steps or TTL coordination.

**Webhook compatibility** — GHL and Stripe webhooks have signature verification. Releases must note if webhook handlers changed in ways that could fail in-flight requests during deploy.

**Scheduled job alignment** — `routes/console.php` schedules hourly GHL token refresh, 15-minute Bridge cache refresh, and weekly GIS metadata probes. Releases should flag if job behavior changes and whether queues need draining.

## Common Patterns

**Migrations + Seeds** — When adding GHL tables or billing features, releases typically require:
```bash
php artisan migrate
php artisan db:seed --class=GhlConfigSeeder  # if lead mappings changed
```

**Docker image promotion** — Standard pattern: build from project root with build context `.`, tag with commit SHA and `latest`, deploy `idx-api` before `idx-images` if URL rewriting logic changed.

**Feature flag via subscription tier** — New capabilities are often gated by `SubscriptionCatalog` plan definitions. Rollout checklist includes verifying Stripe price IDs match config and tier capabilities are documented.

**Environment variable synchronization** — Cross-subsystem releases often need env vars synced across:
- `.env.example` (developer template)
- Root `.env` (Docker Compose)
- Production secrets management

**Rollback indicators** — Monitor these for rollback decisions:
- `bridge_proxy_audit_logs` error rate (Bridge proxy health)
- `ghl_sync_logs` failed status (lead sync health)
- GIS `degraded=true` responses (ArcGIS failover frequency)

**Documentation drift check** — Releases touching public APIs should verify `docs/` consistency:
- `docs/idx-api-bridge-proxy.md` for Bridge changes
- `docs/ghl-api-routes-reference.md` for GHL route changes
- `docs/gis-api.md` for GIS parameter/response changes