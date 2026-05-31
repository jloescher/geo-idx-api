# Strategy & Monetization Reference

## Contents
- Current Monetization Model
- Tier Architecture
- GIS Data as Revenue Lever
- Expansion Revenue Paths
- Anti-Patterns

## Current Monetization Model

Quantyra IDX is an **invite-only B2B API** with no visible pricing, no billing integration, and no subscription management. Revenue mechanics are implicit:

| Tier | Access method | Scope | GIS access |
|------|--------------|-------|------------|
| Staging | Auto-created token | `idx:full` | Full (testing) |
| Production | Domain-verified token | `idx:full` | Full |
| Teaser | `idx:access`-only token | Limited | 40 features, 4-decimal coords |

There is no payment gateway, no usage metering for billing, and no tier upgrade flow beyond DNS verification. Monetization is currently handled outside the application (presumably through direct sales contracts).

## Tier Architecture

### Token Scope Model

```go
// Tokens are created with explicit ability arrays:
// Production:  []string{"idx:full"}
// Staging:     []string{"idx:full"}
// Future tiers could add: []string{"idx:access", "idx:search", "idx:images"}
```

The `abilities` field on `personal_access_tokens` supports granular scope control but currently only uses `idx:full`.

### GIS Teaser Configuration

```go
// internal/service/gis/teaser.go — configurable tier boundaries
maxFeatures := cfg.TeaserMaxFeatures    // GIS_TEASER_MAX_FEATURES env
decimals := cfg.TeaserCoordDecimals     // GIS_TEASER_COORD_DECIMALS env
```

These environment variables control the teaser tier without code changes — useful for experimenting with tier boundaries.

### Domain Verification as Paywall

DNS TXT verification (`VerifyTXT()`) functions as an identity confirmation step. The verified domain is required for production API access:

```go
// internal/handler/dashboard/handler.go — verification gate
_, err = h.db.Pool.Exec(c.Context(), `
    UPDATE domains SET verification_status = 'verified', txt_verified_at = NOW()
    WHERE id = $1 AND user_id = $2`, id, uid)
// Only after this does a production token get issued
```

## GIS Data as Revenue Lever

The GIS teaser tier is the most sophisticated monetization mechanism in the codebase. It creates a data-fidelity gradient:

| Metric | Full access | Teaser (`idx:access`) |
|--------|------------|----------------------|
| Feature count | Unlimited | 40 (configurable) |
| Coordinate precision | Full | 4 decimal places (~11m) |
| Upgrade path | — | Verify a domain |

### Monetization Insight

The teaser creates upgrade desire through the **endowment effect**: users see real parcel data (they "own" the preview) but at reduced fidelity. The gap between what they see and what they could have drives the upgrade decision.

## Expansion Revenue Paths

Based on the existing architecture, these expansion paths are available without structural changes:

### 1. Tiered Token Scopes

The `abilities` field already supports multiple scopes:

```go
// new code to add — hypothetical tiered scopes
[]string{"idx:search"}           // Search only
[]string{"idx:search", "idx:images"}  // Search + images
[]string{"idx:full"}             // Everything
[]string{"idx:full", "gis:full"} // Everything + full GIS
```

### 2. Usage Metering via Audit Logs

```sql
-- Existing audit_logs table can drive usage-based billing
SELECT domain_id, DATE(created_at) AS day, COUNT(*) AS requests
FROM audit_logs
GROUP BY domain_id, DATE(created_at)
ORDER BY day DESC;
```

### 3. Multi-MLS Upsell

The `allowed_mls_datasets` JSONB column on `domains` currently stores a single dataset. Supporting multiple datasets per domain is an expansion lever:

```go
// Current: `["stellar"]`
// Upsell: `["stellar", "beaches"]` — add Beaches MLS as a paid add-on
```

## Anti-Patterns

### WARNING: Adding Billing Without Token Scope Enforcement

**The Problem:** Adding payment collection without enforcing scope boundaries at the API middleware level.

**Why This Breaks:** If all tokens get `idx:full` regardless of payment status, billing becomes optional. Users will pay once, then use the API indefinitely.

**The Fix:** Before adding billing, ensure the middleware in `internal/api/middleware/` checks token abilities on every request. See the **auth-api-token** skill for scope enforcement patterns.

### WARNING: Hardcoding Tier Limits

**The Problem:**

```go
// BAD — hardcoded tier values in business logic
if featureCount > 40 {
    // truncate
}
```

**Why This Breaks:** Tier limits change. Hardcoded values require code changes and deployments to adjust pricing.

**The Fix:** The existing `config.GISConfig` pattern (env-driven) is correct. Extend this pattern to any new tier boundaries.

### WARNING: Free Tier Without Engagement Tracking

**The Problem:** Offering a free tier without measuring whether users are deriving value from it.

**Why This Breaks:** Free users who don't engage never convert. Without tracking, you can't identify which users need outreach.

**The Fix:** Before adding a free tier, ensure the audit log captures enough signal to measure engagement. The existing `audit_logs` table provides request-level data; ensure it's queryable by token scope.

See the **auth-api-token** skill for token scope and middleware patterns.
See the **geospatial** skill for GIS configuration.
See the **cache-postgres** skill for audit log storage.
See the **queue-postgresql** skill for background job patterns that could support usage metering.