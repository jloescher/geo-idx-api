---
name: building-acquisition-tools
description: Designs lead magnets or free tools for acquisition
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Building Acquisition Tools Skill

This skill designs self-serve acquisition tools and lead magnets that drive traffic, capture leads, and convert users to paid subscriptions. Tools are embedded via widgets, hosted as public utilities, or gated behind registration flows.

## Quick Start

1. **Identify the distribution surface** — GHL marketplace widgets (`/widget/*`), standalone public pages (`/marketing`), or embedded iframe components
2. **Set the teaser boundary** — Use `idx:access` for limited listings (3 items), full MLS requires subscription upgrade
3. **Enable lead capture** — Widget lead forms post to `/widget/api/leads` which creates `QuantyraLead` and syncs to GHL CRM
4. **Gate strategically** — Apply OTP after `gate_after_views` hits; use GIS parcels for map dwell time before gating
5. **Track conversion** — `ghl_installed_locations.lead_count` and `subscription_status` measure tool effectiveness

## Key Concepts

**Widget Architecture** — Three-phase middleware chain (key validate → origin validate → CORS append) lets external sites embed IDX search, lead forms, and listing showcases. Each widget type maps to a route in `routes/ghl-widget.php` with its own Blade template.

**Teaser Mode** — Non-`idx:full` requests get capped listings (3 items via `BridgeTeaser`) and simplified GIS coordinates (~11m precision). Cached full data stays canonical; teaser applied after decompression so upgrade path is instant.

**Lead Ingest Pipeline** — `POST /widget/api/leads` → `QuantyraLead` → `SyncLeadToGhlJob` → GHL contact/opportunity via `GhlLeadMapping` rules. Tags pipeline and stage from seeder config.

**Geography as Acquisition** — GIS `/api/v1/gis` parcels (public Florida cadastral) overlay on Leaflet maps without MLS compliance burden. Keeps users engaged pre-registration; chain with `/listings` call for full property data.

**Billing Integration** — Free tools check `subscription_status` (none/trial/active) via `SubscriptionCatalog`. Upgrade CTAs use Stripe Checkout sessions with trial days from config.

## Common Patterns

**Embed Widget** — Create widget view in `resources/views/widget/`, add route in `ghl-widget.php`, enforce `GHL_WIDGET_RATE_LIMIT` per API key, validate Origin against `ghl_registered_urls`.

**Gated Public Tool** — Build Livewire component in `app/Livewire/Marketing/`, use `DomainOrTokenAuth` middleware, apply teaser logic via `BridgeProxyController`, redirect to `billing.checkout` when limits hit.

**API-First Utility** — Expose standalone endpoint (e.g., GIS proxy), use Sanctum for server-to-server or domain registration for browser traffic, cache aggressively at edge (`GIS_EDGE_CACHE_TTL`), log to `bridge_proxy_audit_logs`.

**GHL Marketplace Distribution** — OAuth install flow → URL registration → widget key issuance (`qh_*` prefix). Agency tokens exchange to location tokens via `LocationTokenService`; webhooks handle `INSTALL`/`UNINSTALL` lifecycle.

**Degradation Strategy** — When upstream fails (ArcGIS, Bridge), return `meta.degraded=true` with `leaflet_fallback` URL for OSM tiles. Empty feature collections preserve UX while protecting data contracts.