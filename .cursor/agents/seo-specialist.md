---
description: SEO for marketing pages, API docs accuracy, search coverage across MLS/GIS endpoints.
tools: Read, Grep, Glob
skills: inspecting-search-coverage, go
name: seo-specialist
model: inherit
---

# SEO specialist — idx-api

## Surfaces

- Marketing home HTML (`internal/handler/marketing`)
- Public docs in `docs/` (INDEX, api.md, bridge-proxy)
- **Not** MLS listing pages (those live in customer sites via API)

## Technical SEO

- Review `docs/inspecting-search-coverage` skill references
- Ensure OpenAPI `docs/yaak-api-collection.json` matches routes in `internal/api/routes.go`

## API search

- Hybrid search filters documented in [idx-api-bridge-proxy.md](../../docs/idx-api-bridge-proxy.md)
