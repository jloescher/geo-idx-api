# Growth Engineering Reference

## Contents
- Growth Loops in This Codebase
- API Key Re-Issuance as Growth Event
- Multi-MLS as Expansion Lever
- Anti-Patterns

## Growth Loops in This Codebase

Growth in a developer API follows a specific loop:

```
Developer reads docs → Creates API key → Makes first call → Builds on platform → Needs more datasets/expenses → Upgrades
```

Each step maps to a code artifact:

| Loop step | Code surface | Content lever |
|-----------|-------------|---------------|
| Reads docs | `docs/*.md` | Working examples, clear auth flow |
| Creates key | `/dashboard`, `internal/handler/auth` | Copy-paste token generation |
| First call | `POST /api/v1/search` | Low-friction dataset routing (`?dataset=`) |
| Builds on platform | GIS teaser, comps API | Scope-based access expansion |
| Upgrades | Token scopes (`idx:access`) | Teaser-to-full conversion copy |

## API Key Re-Issuance as Growth Event

From `docs/go-cutover.md`: the Go migration requires customers to re-issue API keys. This is a forced touchpoint — use it for growth:

1. **Re-issuance email/dashboard prompt**: Include "What's new in Go" content
2. **New token experience**: Show available datasets and scopes at creation time
3. **Post-re-issuance**: Audit which endpoints each domain uses, suggest unexplored features

## Multi-MLS as Expansion Lever

The system supports `bridge_stellar` and `spark_beaches` (see `README.md`). Growth content beats:

1. **Cross-sell**: "Using Stellar? Beaches MLS is available on the same API key with `?dataset=beaches`"
2. **New MLS onboarding**: When a new dataset is added, create a content arc (see SKILL.md editorial arc pattern)
3. **Dataset comparison**: Content showing side-by-side RESO field coverage

## Anti-Patterns

### WARNING: Growth Tactics Without API Reliability

**The Problem:** Driving developers to an API that has replication lag, stale cache, or queue backlog.

**Why This Breaks:** First impressions are final for developers. A 500 error on the first API call after reading your content means permanent churn.

**The Fix:** Verify `/healthz` and `/readyz` pass. Check `GET /api/v1/bridge/stats` for replication status. Content beats must not run during known instability windows (migration periods, queue backlogs).

### WARNING: Ignoring the Scheduler as a Growth Signal

**The Problem:** The scheduler (`cmd/scheduler`) runs `mls.replication_kickoff` every minute. When replication lags, the API serves stale data — but content may still be driving traffic.

**The Fix:** Correlate content beats with scheduler health. If `replication_in_progress` is true for extended periods, pause acquisition content and focus on retention messaging.