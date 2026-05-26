# Roadmap & Experiments Reference

## Contents
- Config-Driven Feature Flags
- Experiment Architecture Constraints
- Rollout Patterns
- Verification Commands
- Anti-Patterns

## Config-Driven Feature Flags

The platform uses environment-variable-driven flags through `internal/config/config.go`. There is no feature flag service (LaunchDarkly, Unleash, etc.).

### Existing Flag Pattern

```go
// existing pattern — config.go
type MLSConfig struct {
    StellarEnabled  bool  `env:"MLS_STELLAR_ENABLED" envDefault:"true"`
    BeachesEnabled  bool  `env:"MLS_BEACHES_ENABLED" envDefault:"true"`
}

type GISConfig struct {
    TeaserMaxFeatures    int           `env:"GIS_TEASER_MAX_FEATURES" envDefault:"40"`
    TeaserCoordDecimals  int           `env:"GIS_TEASER_COORD_DECIMALS" envDefault:"4"`
    MaxBboxSpanDeg       float64       `env:"GIS_MAX_BBOX_SPAN_DEG" envDefault:"0.35"`
}
```

### Flag Categories

| Category | Pattern | Example | Rollback |
|----------|---------|---------|----------|
| Kill switch | `bool` with `envDefault:"true"` | `StellarEnabled` | Set env to `false`, restart |
| Tier limit | `int` with default | `TeaserMaxFeatures` | Change env, restart |
| Size tuning | `int` with default | `StellarPersistChunk` | Change env, restart |
| Duration | `time.Duration` with default | `EdgeCacheTTL` | Change env, restart |

### Adding a New Flag

1. Add field to the appropriate config struct with `env` and `envDefault` tags
2. Check in handler or middleware — not in service layer
3. Add to `.env.example`
4. Document in relevant `docs/` file
5. Deploy with env change — no code deploy needed for flag toggle

## Experiment Architecture Constraints

### No A/B Testing Infrastructure

The platform has no A/B testing framework. The architecture constrains experiments:

| Constraint | Impact |
|-----------|--------|
| Server-rendered dashboard | No client-side variant assignment |
| No session-based feature flags | Cannot vary features per session |
| Config loaded at startup | Flag changes require process restart |
| Multi-DC deployment | Flags must be consistent across regions |

### What IS Possible

| Experiment Type | Mechanism | Effort |
|----------------|-----------|--------|
| Feature on/off | Config flag + env var | Low |
| Tier limit change | Config int + env var | Low |
| Default behavior | `envDefault` tag | Low |
| Per-domain enable | `domains` table column | Medium |
| Gradual rollout | Deploy to one DC first | Medium |

### Per-Domain Experiment Pattern

```go
// new code to add — per-domain feature flag
type Domain struct {
    // ... existing fields
    Features []string // e.g., ["comps", "gis-full"]
}

// new code to add — check in middleware
func hasFeature(domain Domain, feature string) bool {
    for _, f := range domain.Features {
        if f == feature { return true }
    }
    return false
}
```

## Rollout Patterns

### Safe Rollout Checklist

Copy this checklist for feature releases:
- [ ] Add config flag with `envDefault:"false"` (off by default)
- [ ] Deploy to staging with flag enabled
- [ ] Verify via `GET /readyz` and `GET /api/v1/bridge/stats`
- [ ] Deploy to production with flag disabled
- [ ] Enable flag on one DC (NYC or ATL)
- [ ] Monitor audit logs for errors and usage
- [ ] Enable flag on second DC
- [ ] Monitor for 24 hours
- [ ] Remove flag (optional) after stability confirmed

### Multi-DC Rollout Consideration

The platform runs two DCs (NYC + ATL) with shared Patroni PostgreSQL. Feature flags in config affect both DCs simultaneously. For gradual rollout:

1. Deploy new image to one DC with flag enabled
2. Other DC runs old image (flag doesn't exist → uses default)
3. Once validated, deploy to second DC

This requires images with different env vars per DC, which Coolify supports.

## Verification Commands

| Verification | Command |
|-------------|---------|
| Service health | `curl /healthz` |
| DB + PostGIS | `curl /readyz` |
| Replication status | `curl /api/v1/bridge/stats` |
| Type check (if frontend) | `bun run check-types` |
| Go build | `GOFLAGS=-mod=mod go build ./cmd/...` |
| Go test | `GOFLAGS=-mod=mod go test ./...` |

## Anti-Patterns

### WARNING: Feature Flags as Code Branches

**The Problem:**

```go
// BAD — deeply nested flag logic
if flag1 {
    if flag2 {
        // path A
    } else {
        if flag3 { /* path B */ }
    }
}
```

**Why This Breaks:** Combinatorial explosion. Each flag doubles the test matrix. Debugging requires knowing all flag states.

**The Fix:** One flag, one early return. Keep paths independent. If you need combinations, use a single enum/config value instead of multiple booleans.

### WARNING: Database-Driven Feature Flags Without Migration

**The Problem:** Adding a `features JSONB` column to `domains` without a migration.

**Why This Breaks:** Schema drift between environments. Staging and production have different column states.

**The Fix:** Use Goose migrations in `migrations/`. See the **postgres** skill for migration patterns.

See the **deploy-coolify** skill for multi-DC deployment and environment configuration.
See the **cache-postgres** skill for edge cache TTL configuration.