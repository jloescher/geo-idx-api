# Strategy & Monetization Reference

## Contents
- Current business model constraints
- Monetization surfaces in the codebase
- Pricing infrastructure gaps
- Dataset as monetization lever
- Anti-patterns

## Current Business Model Constraints

The platform is invite-only with no billing, no pricing page, no usage tiers, and no public signup. All access is controlled by:

1. **Admin-seeded invitation** — user cannot self-serve
2. **Domain verification** — DNS TXT proves domain ownership
3. **API token** — Bearer token with `idx:full` or `idx:access` abilities
4. **MLS feed allowlist** — `domains.allowed_mls_datasets` restricts which MLS feeds a domain can access

There is no usage metering, no rate limit tiering, and no payment integration.

## Monetization Surfaces in the Codebase

### Token abilities system

```go
// existing — internal/repository/token.go
type APIToken struct {
    Abilities []string `json:"abilities"`
}
```

Current abilities: `idx:full` (all endpoints) and `idx:access` (limited GIS teaser — see `middleware/domain_token.go`). This is a two-tier system ready for expansion.

### Domain-level dataset access

```go
// existing — domains table has allowed_mls_datasets JSONB
// internal/repository/domain.go
func (r *DomainRepo) AllowedDatasets(ctx context.Context, domainID int64) ([]string, error)
```

Domains are restricted to specific MLS datasets (e.g., `stellar`, `beaches`). This is a per-domain access control that maps to per-feed billing.

### Audit logs for usage metering

```go
// existing — internal/service/audit/
// Every authenticated API request writes to audit_logs
```

The `audit_logs` table is a ready-made usage metering source. It records user, endpoint, timestamp, and can be aggregated for billing.

### Comps API as premium feature

```go
// existing — internal/handler/comps/
// POST /api/v1/comps/run with BPO, home value, and investor modes
```

The comps endpoint is a differentiated feature not available from raw MLS feeds. See `docs/comps-api.md`.

## Pricing Infrastructure Gaps

### WARNING: No billing integration

There is no Stripe, Lemon Squeezy, or any payment integration. Monetization would require:

| Need | Implementation surface |
|------|----------------------|
| Usage metering | Aggregate `audit_logs` by user/domain/month |
| Billing tiers | New `billing_plans` table + token ability expansion |
| Payment collection | Stripe Checkout or Billing API integration |
| Invoice generation | Usage aggregation → invoice line items |

### WARNING: No usage limits or quotas

API tokens have abilities but no usage caps. A token with `idx:full` can make unlimited requests. Adding quotas requires:

```sql
-- new code to add — quota tracking
ALTER TABLE personal_access_tokens ADD COLUMN monthly_limit INT;
ALTER TABLE personal_access_tokens ADD COLUMN current_usage INT DEFAULT 0;
```

Plus a middleware check before request processing (before the upstream MLS call, not after).

## Dataset as Monetization Lever

The multi-MLS architecture supports per-feed pricing:

| Dataset | Source | Market |
|---------|--------|--------|
| `stellar` | Bridge Data Output | Stellar MLS region |
| `beaches` | Spark Platform | Beaches MLS region |

Domains request specific datasets via `allowed_mls_datasets`. This naturally maps to:

- **Single-feed plan** — one MLS region
- **Multi-feed plan** — all available datasets
- **Premium datasets** — higher tier for additional feeds as they come online

The `mlsAccess` middleware already enforces feed restrictions. Monetization only needs a billing layer on top.

## Anti-Patterns

### WARNING: Free staging token with no expiration

`CreateStagingToken` generates an `idx:full` token with no expiration date and no domain verification requirement. This is a permanent, unrestricted API key. Add:

```go
// new code to add — staging token with TTL
plain, err := h.tokens.Create(ctx, uid, "Staging", []string{"idx:full"})
// Set expires_at to 7 days from now on the token row
```

### WARNING: No plan-based feature gating

The `idx:full` ability grants access to all endpoints including premium features (comps, GIS). As monetization layers are added, the ability system needs granular permissions:

- `idx:search` — basic listing search
- `idx:comps` — BPO and comp analysis
- `idx:gis` — parcel geometry access
- `idx:full` — everything (premium plan)

Token creation in `VerifyTXT` hardcodes `[]string{"idx:full"}`. This needs to become plan-dependent.

### WARNING: No usage-based cost tracking

MLS API calls to Bridge and Spark incur upstream costs. The platform does not track per-domain upstream request volume. Without this, you cannot correlate revenue with costs. Add upstream request counting per domain to the audit log or a dedicated `api_usage` table.