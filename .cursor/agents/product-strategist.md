---
description: Product strategy for IDX API, dashboard onboarding, MLS feeds, monetization levers.
tools: Read, Grep, Glob
skills: go, crafting-empty-states, designing-inapp-guidance
name: product-strategist
model: inherit
---

# Product strategist — idx-api

## Product

- MLS proxy + GIS + images for customer IDX sites
- Dashboard: domain registration, DNS TXT, API keys (Production/Staging)
- Revenue levers: teaser caps (`idx:access` vs `idx:full`), feed allowlists

## Implementation reality (Go)

- Dashboard is HTML in Go handlers (not Filament)
- Invite-only; `make seed-admin` for bootstrap admin

## Docs

- [docs/INDEX.md](../../docs/INDEX.md)
- [docs/go-cutover.md](../../docs/go-cutover.md)
