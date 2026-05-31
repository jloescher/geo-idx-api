# Distribution Reference

## Contents
- Available Channels
- Scheduling Content with Scheduler Patterns
- Beat Cadence Matrix
- Anti-Patterns

## Available Channels

This project has limited but focused distribution surfaces:

| Channel | Audience | Content type |
|---------|----------|-------------|
| `docs/` | Developers integrating the API | Reference, guides, changelog |
| Dashboard (`/dashboard`) | Active customers | In-app announcements, token management |
| Release notes (git log) | Technical stakeholders | Feature/fix summaries |
| `docs/INDEX.md` | New visitors | Documentation navigation |

## Scheduling Content with Scheduler Patterns

The project uses `robfig/cron/v3` for job scheduling (see `cmd/scheduler`). Apply the same cadence thinking to content:

```go
// Scheduler patterns from cmd/scheduler (existing)
// Every minute:  mls.replication_kickoff
// Every 15 min:  mls.proxy_cache_purge
// Daily 03:05:   mls.purge_closed_listings
// Monday 06:30:  gis.probe_sources

// Content cadence alignment:
// - Feature releases: tied to merge cadence (no fixed cron)
// - Docs updates: immediately on feature merge
// - Social beats: weekly, aligned with scheduler quiet periods
```

## Beat Cadence Matrix

| Beat | Frequency | Trigger | Channel |
|------|-----------|---------|---------|
| Release notes | Per merge | PR merge to `staging` | `docs/`, git log |
| API changelog | Per feature | New endpoint or breaking change | `docs/INDEX.md` |
| GIS data source update | Weekly (Monday) | `gis.probe_sources` completion | Dashboard |
| Onboarding refresh | Monthly | Manual | Dashboard, docs |
| Deep-dive doc | Bi-weekly | Feature backlog | `docs/` |

## Anti-Patterns

### WARNING: Announcing on Channels That Don't Exist

**The Problem:** Planning distribution to "Twitter, LinkedIn, newsletter" when the project has none of these configured.

**Why This Breaks:** Creates phantom commitments. The distribution plan must match actual surfaces.

**The Fix:** Only plan beats for channels that exist in the project. Add new channels explicitly as infrastructure work before scheduling content for them.

### WARNING: Content Unrelated to Code Events

**The Problem:** Generic real estate market commentary on a developer API's channels.

**The Fix:** Every beat must trace to a code artifact: a new endpoint, a docs page, a scheduler job, a migration. If the content doesn't reference something in the repo, it belongs on a different channel.

## Checklist

Copy this checklist and track progress:
- [ ] All beats reference real code artifacts (endpoints, docs, migrations)
- [ ] Release notes published before social announcements
- [ ] Dashboard banner content matches current beat
- [ ] No phantom channels in the distribution plan
- [ ] Beat cadence aligned with scheduler quiet periods