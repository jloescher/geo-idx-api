# MCP API Parity Design (2026-06-01)

Design spec for expanding idx-api-mcp to match customer `/api/v1` capabilities while supporting OAuth-native Grok Web connectors.

## Goals

1. **OAuth without `mcp_key` in tool args** — Grok passes `Authorization: Bearer <oauth_access_token>` only.
2. **API parity** — Search, listing detail, RESO proxy, GIS, stats via MCP tools.
3. **Safety** — Public listing JSON via `BuildPublicListingJSON*` pipeline; live MLS through idx-api-web cache.

## Architecture

```
Grok/Cursor → idx-api-mcp → PostGIS embed (Active/Pending search)
                         → HTTP apiclient → idx-api-web /api/v1/* (live MLS, RESO, GIS)
```

## Auth

- `internal/mcp/auth` resolves OAuth token scopes + granted MCP key union, or direct `mcp_` bearer / stdio param.
- HTTP middleware and `httpContextFunc` both call `auth.Injector.InjectFromHTTP`.

## Scopes

| Scope | Tools |
|-------|-------|
| `monitor` | snapshot, summary, queue, GIS health, inspect |
| `comps` | run_comps + helpers |
| `content` | search_listings_for_content |
| `api` | search_listings, get_listing, RESO/GIS proxy tools |

## Env (idx-api-mcp)

| Variable | Purpose |
|----------|---------|
| `MCP_API_INTERNAL_URL` | idx-api-web base URL for HTTP delegation |
| `MCP_API_DOMAIN_SLUG` | `X-Domain-Slug` for service PAT calls |
| `MCP_API_SERVICE_TOKEN` | Long-lived PAT with `idx:full` |

## Rate limits

`mcp_tool_usage` table + `internal/mcp/ratelimit` — rolling per-minute windows by MCP key or OAuth client.

## Phases shipped

- **Phase 0:** OAuth-native auth
- **Phase 1:** search_listings, get_listing, live delegation, comps domainSlug fix
- **Phase 2:** RESO/GIS/stats/pub proxy tools
- **Phase 3:** rate limits, optional `mcp_keys.domain_id`, get_monitoring_summary
