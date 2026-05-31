# Roadmap & Experiments Reference

How release notes reflect phased rollouts and experimental features for Quantyra IDX API.

## Phased Rollout in Release Notes

Quantyra uses phased infrastructure (Phase 1: primary-only, Phase 2: read replicas). Release notes should reflect the current phase and any phase transitions.

### Phase Markers in Notes

| Phase | What to Document | Example |
|-------|-----------------|---------|
| Phase 1 (primary only) | All containers point to Patroni primary | `docs/coolify-deployment.md` §8 |
| Phase 2 (read replicas) | `DB_READ_HOST` routing for API reads | `docs/coolify-deployment.md` §9 |
| Experimental | Feature flags, env-var-gated behavior | `BRIDGE_SYNC_FULL_PROPERTY` |

### DO: Mark experimental features clearly

```markdown
## v2.4.0

### Experimental
- **Bridge**: Nav hydration after replication (`BRIDGE_SYNC_NAV_HYDRATE_AFTER_REPLICATION`, default `true`) — paginates `/Property` with nav `$expand` to backfill `unit`/`room`/`open_house` JSONB. Report issues with `dataset=stellar` listings missing expanded collections.
```

### DON'T: Ship experimental features without disclosure

```markdown
<!-- BAD — looks like a stable feature -->
### Features
- **Bridge**: Nav hydration now populates unit, room, and open house data after replication
```

### Feature Flags and Env Vars

When a release introduces env-var-gated behavior, document:

1. The env var name and default value
2. What changes when the flag is enabled/disabled
3. Which `dataset_slug` values are affected

Known env-var gates from the codebase:

| Env Var | Default | Controls |
|---------|---------|----------|
| `BRIDGE_SYNC_FULL_PROPERTY` | `true` | Full Property payload vs `$select` mode |
| `BRIDGE_SYNC_NAV_HYDRATE_AFTER_REPLICATION` | `true` | Post-replication nav expand |
| `MLS_LOCAL_MIRROR_ROLLING_MONTHS` | `12` (local), `3` (staging), `0` (prod) | Rolling window for mirror data |
| `MLS_REPLICATION_FRESHNESS_MINUTES` | `15` | How often incremental sync runs |

### Roadmap Transition Notes

When a feature graduates from experimental to stable:

```markdown
### Graduated from Experimental
- **Nav hydration** (`BRIDGE_SYNC_NAV_HYDRATE_AFTER_REPLICATION`) is now stable — enabled by default for all Bridge datasets. The env var remains for opt-out.
```

### Roadmap Release Checklist

Copy this checklist for releases with phased or experimental changes:
- [ ] Experimental features are marked with env var + default
- [ ] Phase transitions (1→2) are called out explicitly
- [ ] Graduated features are noted as no longer experimental
- [ ] Dataset-specific behavior is documented per `dataset_slug`

## Related References

- See the **deploy-coolify** skill for multi-DC phase documentation
- See the **deploy-patroni** skill for read replica phase notes
- See `docs/coolify-deployment.md` §9 for Phase 2 read replica details