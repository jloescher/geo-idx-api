# Distribution Reference

Distribute release stories across Quantyra IDX API's available channels.

## Contents
- Distribution Channels
- Release Timing
- Multi-DC Rollout Communication
- Anti-Patterns

## Distribution Channels

| Channel | Surface | Reach | Format |
|---------|---------|-------|--------|
| Dashboard | `internal/handler/dashboard/handler.go` | All authenticated users | In-app banner or card |
| API docs | `docs/*.md` | Public + authenticated | Markdown with curl examples |
| OpenAPI spec | `docs/yaak-api-collection.json` → `GET /openapi.json` | API consumers | Schema changes |
| Git tags | Repository | Operators watching deploys | Conventional commit summary |
| Health endpoints | `GET /healthz`, `GET /readyz` | Monitoring systems | JSON status |

### WARNING: No email or notification system

**The Problem:** The codebase has no email service, webhook notification, or in-app notification system. Dashboard users have no way to learn about releases without visiting `/dashboard`.

**Why This Breaks:** Critical changes (auth token re-issuance, breaking API changes) may go unnoticed. The Go cutover (`docs/go-cutover.md`) explicitly says "Notify customers to re-issue API keys" but provides no notification mechanism.

**The Fix:** For breaking changes, use direct outreach (the invite system in `internal/service/auth` stores emails). For feature releases, update the marketing home page hero copy and docs. Future: add a `release_notes` table and dashboard notification banner.

## Release Timing

The scheduler (`cmd/scheduler`) runs on a fixed cron schedule:

| Job | Interval | Release implication |
|-----|----------|-------------------|
| `mls.replication_kickoff` | Every minute | Replication changes visible within 60s of deploy |
| `mls.proxy_cache_purge` | Every 15 min | Cache-dependent features need up to 15 min to propagate |
| `mls.purge_closed_listings` | Daily 03:05 | Data retention changes apply overnight |

Announce features after workers and schedulers are deployed and verified. Use the deploy order from `docs/coolify-deployment.md`:

1. Workers (all DCs) → 2. Schedulers → 3. APIs → 4. idx-images

## Multi-DC Rollout Communication

Production spans NYC (re-db) and ATL (re-node-02) with Cloudflare geo LB:

| Audience | What they need to know | When |
|----------|----------------------|------|
| API consumers | Nothing — traffic routes to healthy DC | After deploy completes |
| Operators | Both DCs deployed, one scheduler leader | During deploy |
| Dashboard users | Zero downtime expected | Before deploy |

### Rollout communication template

```markdown
## [Version] Deploy — YYYY-MM-DD

**Expected downtime:** None (rolling deploy across DCs)
**Deploy window:** [start time]–[end time] UTC

### What operators need
- Both DCs: workers first, then schedulers (confirm leader), then APIs
- New env vars: [list any]
- Migration: `goose -dir migrations up` on primary

### What API consumers need
- New: [features]
- Changed: [behavior changes]
- Action required: [breaking changes with deadline]
```

## Anti-Patterns

### WARNING: Announcing before deploy completes

**The Problem:** Distributing release stories while workers are still draining the old version's jobs.

**Why This Breaks:** API consumers test the announced feature against an instance still running old code. Support requests spike. Trust erodes.

**The Fix:** Verify all DCs are healthy (`GET /healthz`, `GET /readyz`) and workers have drained `jobs` before distributing any announcement.

### WARNING: Docs-only distribution for breaking changes

**The Problem:** Relying solely on `docs/` updates to communicate auth changes.

**Why This Breaks:** API consumers don't re-read docs unprompted. The Go cutover required API key re-issuance — without direct notification, users discover the break via 401 errors.

**The Fix:** For breaking changes, use every available channel: dashboard banner, docs update, and direct email to affected domains (emails are stored in `users` table).

See the **deploy-coolify** skill for multi-DC deploy specifics.
See the **writing-release-notes** skill for changelog generation.