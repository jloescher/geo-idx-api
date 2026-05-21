# Strategy and Monetization Reference

## Contents
- Current business model
- Monetization signals in the codebase
- Tier design from existing data
- Anti-patterns

## Current Business Model

Quantyra IDX is an **invite-only B2B MLS proxy**. The platform:
- Proxies Bridge (Stellar) and Spark (Beaches) MLS data
- Delivers listing images via NVMe-cached proxy
- Provides PostGIS-backed search and GIS parcel data
- Charges per domain with MLS dataset access

### Revenue surfaces in the codebase

| Surface | Table | Monetization signal |
|---|---|---|
| Domain registration | `domains` | One domain = one customer site |
| API token scopes | `personal_access_tokens` | `idx:full`, `idx:access` tiers |
| MLS dataset routing | `domains.allowed_mls_datasets` | JSONB array — per-dataset access control |
| API usage | `audit_logs` | Per-domain, per-endpoint call volume |
| GIS teaser tiers | `internal/handler/gis` | Free teaser vs authenticated full access |

The `allowed_mls_datasets` field is the existing access control for monetization:

```go
// internal/handler/dashboard/handler.go:188 — domain creation with dataset access
_, err := h.db.Pool.Exec(c.Context(), `
    INSERT INTO domains (..., allowed_mls_datasets, ...)
    VALUES ($1, $2, $3, $4::jsonb, ...)
`, uid, slug, mls, `["`+mls+`"]`, ...)
```

## Tier Design from Existing Data

The codebase already supports two scope levels:

| Scope | Access | Potential tier |
|---|---|---|
| `idx:full` | All endpoints, full listing data | Professional |
| `idx:access` | Limited (GIS teaser only) | Starter |

### Usage-based pricing signals

```sql
-- Per-domain API call volume (last 30 days)
SELECT
  d.domain_slug,
  COUNT(*) AS total_calls,
  COUNT(*) FILTER (WHERE a.path LIKE '/api/v1/search%') AS search_calls,
  COUNT(*) FILTER (WHERE a.path LIKE '/images/%') AS image_calls,
  COUNT(*) FILTER (WHERE a.path LIKE '/api/v1/gis%') AS gis_calls
FROM domains d
JOIN audit_logs a ON a.domain_id = d.id
WHERE a.created_at > NOW() - INTERVAL '30 days'
GROUP BY d.id
ORDER BY total_calls DESC;
```

This query drives pricing tier decisions: if image proxy calls dominate, price on image volume. If search calls dominate, price on search queries.

### Feature gating points

Existing code that can support tiered access:

1. **Search endpoint** (`POST /api/v1/search`) — gate by scope or usage quota
2. **Image proxy** (`/images/*`) — gate by cache tier or monthly image limit
3. **GIS teaser** — already has a free tier pattern in `internal/handler/gis`
4. **Comps API** (`POST /api/v1/comps/run`) — gate BPO mode by scope
5. **Dataset access** — `allowed_mls_datasets` already controls which MLS feeds a domain can use

## Lifecycle messaging for monetization

| Trigger | Message | Goal |
|---|---|---|
| Domain verified, first token created | "Welcome to Quantyra IDX — here's your first API call" | Activation |
| 80% of monthly search quota reached | "You're approaching your plan limit — upgrade for uninterrupted access" | Upsell |
| New MLS dataset available | "Stellar + Beaches data now available — add to your domain" | Cross-sell |
| 30 days inactive | "Your domains haven't made API calls recently — need help?" | Retention |
| Staging token used in production | "We noticed staging traffic on your production domain — switch to a production token" | Quality/upsell |

## Anti-patterns

### WARNING: Hardcoding pricing or limits in handler code

Never embed plan limits, pricing, or quota values in handler string literals or service logic. These must come from configuration (env vars or database) so they can change without redeployment. Follow the existing pattern in `internal/config/config.go` where all tunable values are centralized.

### WARNING: Breaking MLS access control for monetization

The `allowed_mls_datasets` JSONB field controls which MLS feeds a domain can access. Any tier-based gating must respect this field — do not add a separate access control layer that could conflict with MLS data licensing requirements. See `docs/listings-mirror.md` for MLS data scope rules.

## Related Skills

- **auth-api-token** — token scopes and access tiers
- **geospatial** — GIS teaser tier pattern
- **queue-postgresql** — enqueue usage-based notification emails
- **go** — config patterns for pricing variables