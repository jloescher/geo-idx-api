---
name: prioritizing-roadmap-bets
description: Ranks initiatives using impact, effort, and risk signals for the Quantyra IDX API platform
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Prioritizing Roadmap Bets Skill

This skill evaluates potential initiatives for the Quantyra IDX API platform—spanning Bridge MLS proxy, GoHighLevel Marketplace integration, GIS parcel services, and Stripe billing subsystems—by scoring impact against engineering effort and operational risk to guide sequencing decisions.

## Quick Start

1. **Read existing documentation** — Check `docs/INDEX.md` and `docs/superpowers/` for context on current initiatives and approved specs.
2. **Assess initiative scope** — Determine which subsystem(s) are affected: Bridge proxy (`/api/v1/*`), GHL Marketplace (`/leadconnector/*`), GIS proxy (`/api/v1/gis`), or Billing (Stripe/Cashier).
3. **Score the bet** — Rate on three axes:
   - **Impact**: Revenue potential (subscription tiers, metered overage), MLS compliance requirements, or GHL Marketplace visibility
   - **Effort**: Migration complexity (29 migrations in `database/migrations/`), queue job additions, or external API dependencies (Bridge, GHL, ArcGIS)
   - **Risk**: Stellar MLS audit obligations, token refresh reliability, webhook signature verification, or cache invalidation complexity
4. **Compare to active work** — Reference `routes/console.php` for scheduled tasks and `app/Jobs/` for queue workload implications.
5. **Document the decision** — Update planning docs in `docs/superpowers/plans/` with rationale.

## Key Concepts

- **Revenue impact markers**: Look for `SubscriptionCatalog` plan definitions, teaser gating logic, and metered billing hooks in `app/Billing/`
- **Compliance surface**: Stellar MLS audit logging in `bridge_proxy_audit_logs` and `ghl_audit_logs` tables creates hard requirements that override convenience optimizations
- **Subsystem coupling**: GHL Marketplace widgets depend on Bridge proxy for MLS data; GIS proxy is isolated (public ArcGIS only)
- **Cache economics**: 15-minute TTL listings cache (`LISTINGS_CACHE_TTL`) and 3-tier GIS caching define acceptable freshness tradeoffs
- **Token lifecycle**: Hourly `ghl:refresh-tokens` schedule and OAuth token encryption requirements affect reliability scoring

## Common Patterns

- **Bridge proxy enhancements**: High impact for domain-gated features (teaser limits), medium effort for new endpoints in `BridgeProxyController`, low risk if audit logging remains intact
- **GHL Marketplace expansions**: High impact for widget types (`/widget/*` routes), high effort for new OAuth scopes or lead sync mappings, medium risk for webhook handling
- **GIS layer additions**: Medium impact for map engagement (lead capture), medium effort for new ArcGIS source failover chains in `GisProxyService`, low risk (no MLS data)
- **Billing tier changes**: High revenue impact, low-medium effort for `SubscriptionCatalog` updates, high risk if Stripe webhook handling is affected
- **Performance optimizations**: Measure against Octane/FrankenPHP concurrency and PostgreSQL query patterns in `app/Services/Bridge/ListingsCacheService`