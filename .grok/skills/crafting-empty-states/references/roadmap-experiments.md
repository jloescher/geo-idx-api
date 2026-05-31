# Roadmap & Experiments Reference

## Contents
- Feature Rollout Patterns
- Experiment Constraints
- Rollout Checklist
- Anti-Patterns

## Feature Rollout Patterns

This project has no feature flag system, no A/B testing framework, and no gradual rollout mechanism. Features are shipped by deploying code changes to Coolify applications.

### Rollout Mechanisms Available

| Mechanism | When to Use | Example |
|-----------|------------|---------|
| Code deploy | All users get the change | New empty state HTML |
| `is_admin` check | Admin-only preview of new features | New dashboard section |
| Database migration | Schema-backed feature gating | New column on `users` or `domains` |
| Environment variable | Config toggle without code change | `ENABLE_ONBOARDING_CHECKLIST` |

### Admin-Only Preview Pattern

```go
// existing — internal/handler/dashboard/handler.go:170-176
var isAdmin bool
_ = h.db.Pool.QueryRow(c.Context(), `SELECT is_admin FROM users WHERE id = $1`, uid).Scan(&isAdmin)
if isAdmin {
    b.WriteString(`<div class="card"><h2>Invite user</h2>...`)
}
```

Use this same pattern for feature previews:

```go
// new code to add — admin-only feature preview
if isAdmin {
    b.WriteString(`<div class="card"><h2>Setup checklist (preview)</h2>...`)
}
```

### Environment Variable Toggle

```go
// new code to add — in internal/config/config.go, add field
OnboardingChecklist bool `env:"ONBOARDING_CHECKLIST" envDefault:"true"`

// new code to add — in dashboard handler
if h.cfg.OnboardingChecklist {
    b.WriteString(`<div class="setup-progress">...`)
}
```

## Experiment Constraints

### No A/B Framework

The invite-only dashboard serves a small number of users (MLS administrators). Building an A/B framework is over-engineering. Instead:

1. Ship the change to all users
2. Measure via SQL queries (see **product-analytics** reference)
3. Iterate based on feedback

### Multi-DC Considerations

Both NYC and ATL instances serve the same `Dockerfile` targets from the same git commit. You cannot roll out a feature to one DC but not the other without:

- Separate Coolify app configurations (different env vars)
- Or a database-backed toggle that both DCs read

The existing scheduler advisory lock pattern (see the **deploy-patroni** skill) ensures only one scheduler runs. API instances are stateless and read the same database.

## Rollout Checklist

Copy this checklist and track progress:

- [ ] Add feature behind `is_admin` check for internal testing
- [ ] Verify on staging (`APP_ENV=staging`) with staging database
- [ ] Write SQL query to measure the feature's impact (see **product-analytics**)
- [ ] Remove admin guard or add env toggle for general availability
- [ ] Deploy to both DCs (or let Coolify pull the same image)
- [ ] Monitor `audit_logs` and error rates for 24 hours
- [ ] Run `make test` and verify `go build ./cmd/...` succeeds

## Anti-Patterns

- **NEVER** add a third-party feature flag service (LaunchDarkly, Unleash). The user base is small and the deployment is self-hosted. Use database columns or env vars.
- **AVOID** creating a `feature_flags` table with string keys and JSON values. This becomes unmaintainable quickly. Use explicit config fields.
- **AVOID** client-side feature detection (checking `window.featureX`). The dashboard is server-rendered; feature checks belong in Go handlers.
- **NEVER** deploy different code to NYC and ATL Coolify apps. Both should use the same Docker image. Use env vars for configuration differences.

## Related Skills

- See the **deploy-coolify** skill for deployment workflows
- See the **deploy-patroni** skill for multi-DC coordination
- See the **cache-postgres** skill for config-backed caching