# Strategy & Monetization Reference

## Contents
- Current Monetization State
- Revenue Model Options
- Tier Architecture
- Billing Infrastructure
- Anti-Patterns

## Current Monetization State

The platform has **no billing infrastructure**. No subscription table, no payment integration, no usage metering, no invoice generation. The existing `personal_access_tokens.abilities` field (`["idx:access"]` or `["idx:full"]`) is the only tier indicator.

| Component | Status | Location |
|-----------|--------|----------|
| Tier definition | Two abilities in code | `internal/api/middleware/domain_token.go:35` |
| Feature gating | GIS teaser only | `internal/service/gis/teaser.go:24` |
| Usage tracking | Audit logs (not plan-aware) | `audit_logs` table |
| Billing | None | N/A |
| Pricing page | None | N/A |
| Plan management | None | N/A |

### WARNING: No Subscription Table

**Detected:** No `subscriptions`, `plans`, or `billing` tables in migrations.
**Impact:** Tier assignments are manual (admin sets abilities on token creation). No automated upgrade/downgrade, no billing cycles, no revenue recognition.

## Revenue Model Options

For an MLS data API, three models align with the existing architecture:

### 1. Per-Domain Subscription (Recommended)

Charge per verified domain. Maps to existing `domains` table.

```
Starter: $0/mo per domain — idx:access, teaser GIS
Pro:     $X/mo per domain — idx:full, all features
Enterprise: custom — multi-DC, dedicated workers
```

**Why this fits:** The auth model already resolves to a domain. Each `domains` row with `verification_status = 'verified'` is a billable entity.

### 2. Usage-Based (Future)

Charge per API call, GIS query, or image proxy request.

```
Base:    $0 — 10K requests/mo
Growth:  $X per 1K requests over base
GIS:     $X per 1K full-precision parcel queries
```

**Requires:** Request metering in audit logs, aggregation pipeline, billing cycle job.

### 3. Dataset Access (Add-On)

Charge for access to specific MLS datasets.

```
Stellar (Bridge): included
Beaches (Spark):  add-on
Future MLS feeds: per-feed pricing
```

**Maps to:** `domains.allowed_mls_datasets` JSONB — already supports per-domain dataset restrictions.

## Tier Architecture

### Database Schema for Plans

```sql
-- new code to add — migration for plan support
CREATE TABLE plans (
    id          SERIAL PRIMARY KEY,
    slug        VARCHAR(64) UNIQUE NOT NULL,  -- 'starter', 'pro', 'enterprise'
    name        VARCHAR(128) NOT NULL,
    abilities   JSONB NOT NULL,               -- ["idx:access"] or ["idx:full"]
    limits      JSONB NOT NULL DEFAULT '{}',  -- rate limits, GIS features, etc.
    price_cents INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id          SERIAL PRIMARY KEY,
    domain_id   INT NOT NULL REFERENCES domains(id),
    plan_id     INT NOT NULL REFERENCES plans(id),
    status      VARCHAR(32) NOT NULL DEFAULT 'active', -- active, past_due, canceled
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Plan Resolution in Middleware

```go
// new code to add — extend existing domain_token.go pattern
func resolvePlan(c *fiber.Ctx, subRepo *repository.SubscriptionRepo) Plan {
    domain := c.Locals(ctxkeys.MLSDomain).(*dom.Domain)
    sub, _ := subRepo.ActiveForDomain(c.Context(), domain.ID)
    if sub != nil {
        return sub.Plan
    }
    return DefaultPlan // starter
}
```

## Billing Infrastructure

### Recommended: Webhook-Based

The platform already processes webhooks (scheduler enqueues jobs). Use the same pattern:

```go
// new code to add — billing webhook handler
func (h *Handler) StripeWebhook(c *fiber.Ctx) error {
    event := h.stripe.ConstructEvent(c.Body(), c.Get("Stripe-Signature"))

    switch event.Type {
    case "checkout.session.completed":
        h.activateSubscription(event.Data)
    case "customer.subscription.updated":
        h.updatePlan(event.Data)
    case "invoice.payment_failed":
        h.downgradeToStarter(event.Data)
    }
    return c.SendStatus(200)
}
```

### Schedule with Existing Queue

Use the PostgreSQL job queue for billing tasks:

```go
// new code to add — billing job types
// scheduler enqueues monthly:
//   "billing.usage_snapshot" — capture monthly usage for invoicing
//   "billing.subscription_renewal" — check renewals, send invoices
```

See the **queue-postgresql** skill for job enqueue patterns.
See the **auth-api-token** skill for token and domain model details.

## Anti-Patterns

### WARNING: Storing Payment Data Locally

```go
// BAD — storing card numbers in PostgreSQL
INSERT INTO payments (card_number, cvv, expiry) VALUES ($1, $2, $3)
```

**Why This Breaks:** PCI DSS compliance requires specific security controls. Never store full card data. Use Stripe/Braintree tokens.

### WARNING: Manual Plan Changes

```go
// BAD — admin manually sets abilities to "upgrade"
UPDATE personal_access_tokens SET abilities = '["idx:full"]' WHERE id = $1
```

**Why This Breaks:** No audit trail, no billing record, no automated renewal/downgrade. Plan changes must flow through the subscription system with billing state transitions.

See the **postgres** skill for schema design patterns.
See the **cache-postgres** skill for plan caching to avoid subscription lookups on every request.