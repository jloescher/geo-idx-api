# Growth Engineering Reference

## Contents
- Growth Levers in API Products
- Existing Growth Infrastructure
- Referral and Expansion Patterns
- Anti-Patterns

## Growth Levers in API Products

For an MLS data API, growth engineering focuses on:

1. **Developer experience** — docs, API key provisioning, response quality
2. **Usage-based expansion** — starter tier hits limits → upgrade nudge
3. **Domain virality** — each IDX site using the API is visible to end users
4. **Data network effects** — more MLS datasets → more value per domain

## Existing Growth Infrastructure

| Asset | Location | Growth potential |
|-------|----------|-----------------|
| DNS verification | `internal/handler/dashboard/handler.go:52` | Each verified domain is a committed user |
| Multi-MLS datasets | `bridge_stellar`, `spark_beaches` | More datasets = broader market reach |
| GIS teaser | `internal/service/gis/teaser.go` | Built-in upgrade trigger |
| OpenAPI spec | `GET /openapi.json` | Developer onboarding surface |
| Embedded docs | `docs/` directory | SEO and developer education |

### Domain-as-Growth-Unit Pattern

Each verified domain is a growth unit. The current `domains` table tracks:

```sql
-- from migrations — domain lifecycle
verification_status VARCHAR(32) DEFAULT 'pending'  -- pending → verified
is_active BOOLEAN DEFAULT TRUE                      -- active flag
allowed_mls_datasets JSONB                          -- per-domain data access
```

**Growth metric:** Domains with `verification_status = 'verified'` AND `is_active = true` that make API calls in the last 30 days.

## Referral and Expansion Patterns

### Domain-Level Referrals

The existing domain model naturally supports referral tracking:

```go
// new code to add — referral tracking on domain creation
func (h *Handler) StoreDomain(c *fiber.Ctx) error {
    referrer := c.Query("ref") // referral code from existing domain
    slug := c.FormValue("domain")

    // ... existing domain creation ...

    if referrer != "" {
        // track referral relationship
        h.db.Pool.Exec(c.Context(),
            `INSERT INTO domain_referrals (referrer_slug, referred_slug, created_at)
             VALUES ($1, $2, NOW())`, referrer, slug)
    }
}
```

### Expansion via Dataset Access

The `allowed_mls_datasets` JSONB field on domains controls which MLS feeds a domain can access. This is a natural expansion lever:

```go
// new code to add — dataset gating creates upsell opportunities
func (h *Service) checkDatasetAccess(domain *dom.Domain, dataset string) error {
    if len(domain.AllowedMLSDatasets) == 0 {
        return nil // no restriction = all datasets
    }
    for _, allowed := range domain.AllowedMLSDatasets {
        if allowed == dataset {
            return nil
        }
    }
    return ErrDatasetNotOnPlan // triggers upgrade conversation
}
```

### API Response Growth Hooks

```go
// new code to add — embed growth signals in API responses
func addUsageHeaders(c *fiber.Ctx, usage PlanUsage) {
    c.Set("X-Usage-Requests-Month", strconv.Itoa(usage.RequestsMonth))
    c.Set("X-Usage-Requests-Limit", strconv.Itoa(usage.RequestLimit))
    if usage.RequestsMonth > usage.RequestLimit*8/10 {
        c.Set("X-Usage-Warning", "Approaching plan limit. Consider upgrading.")
    }
}
```

## Anti-Patterns

### WARNING: Growth Hacks That Degrade API Quality

```go
// BAD — injecting ads or marketing into API responses
func (h *Handler) Search(c *fiber.Ctx) error {
    results := h.service.Search(params)
    results["sponsored"] = h.getAds() // ads in data API
    return c.JSON(results)
}
```

**Why This Breaks:** API consumers build parsing logic around response shapes. Injecting marketing content into data payloads breaks integrations, loses trust, and violates the API contract.

**Growth for API products belongs in:**
- Response headers (non-breaking)
- Dashboard UI (separate from API responses)
- Email lifecycle (transactional + marketing)
- Developer experience (docs, examples, SDKs)

### WARNING: Viral Loops That Require API Consumers to Act

```go
// BAD — requiring IDX site owners to add Quantyra branding
func (h *Handler) Images(c *fiber.Ctx) error {
    // overlay watermark with "Powered by Quantyra" on every image
    return c.Type("jpeg").Send(watermarkedImage)
}
```

**Why This Breaks:** MLS images have licensing requirements. Watermarking proxy images may violate MLS terms of service and creates legal risk. Growth must respect data licensing.

See the **auth-api-token** skill for domain verification and token management.
See the **frontend-design** skill for dashboard growth surfaces.