# Roadmap & Experiments Reference

## Contents
- Current Experiment Infrastructure
- Feature Toggle Patterns
- Rollout Decision Playbook
- DO/DON'T Patterns

## Current Experiment Infrastructure

This project has no A/B testing framework, no feature flag service, and no experiment assignment system. Feature availability is controlled by:

1. **Environment variables** — `MLS_STELLAR_ENABLED`, `MLS_BEACHES_ENABLED`
2. **Database state** — `domains.allowed_datasets` JSON array
3. **User role** — `users.is_admin` boolean
4. **Config toggles** — `internal/config/config.go` reads env vars at startup

### WARNING: Missing Feature Flag System

**Detected:** No feature flag or experiment framework in dependencies.

**Impact:** Cannot roll out dashboard features to subsets of users, run A/B tests, or gate features by plan.

**Mitigation for onboarding changes:** Use database-driven feature visibility rather than adding a feature flag service. The user count is small enough that config-based toggles are sufficient.

```go
// new code to add — lightweight feature visibility via config
type FeatureConfig struct {
    OnboardingChecklist bool // env: FEATURE_ONBOARDING_CHECKLIST (default: true)
    UsageDashboard      bool // env: FEATURE_USAGE_DASHBOARD (default: false)
}
```

## Feature Toggle Patterns

### Pattern: Environment-Based Toggle

For features that apply to all users in an environment:

```go
// new code to add — in internal/config/config.go
func (c *Config) IsOnboardingEnabled() bool {
    return envBool("FEATURE_ONBOARDING_CHECKLIST", true)
}
```

### Pattern: User-Level Toggle

For features targeting specific users (e.g., beta testers):

```go
// new code to add — check user metadata
func (h *Handler) showFeature(user domain.User, feature string) bool {
    if user.IsAdmin {
        return true // admins see all features
    }
    // Query user_features table or user metadata JSONB
    return false
}
```

### Pattern: Dataset-Level Toggle

MLS dataset access is already controlled per-domain. Extend this pattern for feature gating:

```go
// existing pattern — internal/repository/domain.go
func (r *DomainRepo) AllowedDatasets(domainID int64) ([]string, error)
// Returns ["stellar", "beaches"] based on domains.allowed_datasets JSON
```

## Rollout Decision Playbook

Follow this sequence for any dashboard feature change:

| Step | Action | Verification |
|------|--------|-------------|
| 1. Build behind env toggle | `FEATURE_X_ENABLED=false` by default | Feature not visible in staging |
| 2. Enable in staging | Set env var in staging Coolify app | Verify on staging URL |
| 3. Enable in production | Set env var in production Coolify shared env | Monitor `/healthz` and audit logs |
| 4. Remove toggle | Hardcode `true` after confidence | Remove env var from Coolify |

### Copy This Checklist

```
- [ ] Feature built behind FEATURE_*_ENABLED env toggle (default: false)
- [ ] Toggle added to internal/config/config.go
- [ ] Handler reads toggle before rendering feature
- [ ] Tested with toggle off (feature hidden) and on (feature visible)
- [ ] Enabled in staging Coolify environment
- [ ] Verified on staging URL after deploy
- [ ] Enabled in production Coolify shared environment
- [ ] Monitored audit logs for unexpected behavior for 48 hours
- [ ] Toggle removed and hardcoded after confidence
```

## DO/DON'T Patterns

### DO: Use env toggles for gradual rollout

```go
// GOOD — safe, reversible, works across multi-DC
if cfg.FeatureOnboardingChecklist {
    // render checklist
}
```

### DON'T: Deploy half-finished features without a toggle

```go
// BAD — no kill switch if something goes wrong
func (h *Handler) Dashboard() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // directly renders new onboarding UI with no way to disable
    }
}
```

### DO: Test both toggle states

```go
// GOOD — verify feature works when enabled AND hidden when disabled
func TestDashboard_OnboardingHidden(t *testing.T) {
    cfg := &Config{FeatureOnboardingChecklist: false}
    // assert onboarding HTML not in response
}

func TestDashboard_OnboardingShown(t *testing.T) {
    cfg := &Config{FeatureOnboardingChecklist: true}
    // assert onboarding checklist visible
}
```

### DON'T: Create long-lived feature flags

```go
// BAD — feature flags that never get removed add complexity
if cfg.FeatureNewDashboard2024 { // still here in 2026
```

## Integration Points

- **Config**: `internal/config/config.go` — where env toggles are defined.
- **Coolify env**: See `docs/coolify-deployment.md` — shared environment for multi-DC.
- See the **deploy-coolify** skill for environment variable management across apps.
- See the **go** skill for config struct patterns.