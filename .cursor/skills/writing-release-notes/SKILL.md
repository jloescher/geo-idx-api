---
name: writing-release-notes
description: |
  Drafts release notes tied to shipped features for Quantyra IDX API.
  Use when: generating changelogs from git history, preparing deployment notes, writing customer-facing release summaries, documenting shipped features across MLS proxy, GIS, search, dashboard, and infrastructure surfaces, or responding to "what changed", "release notes", "changelog", "what's new".
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Writing Release Notes Skill

Draft release notes for Quantyra IDX API by parsing conventional commits, mapping changes to product surfaces, and producing audience-appropriate summaries. This project uses `type(scope): message` commits, a multi-service Docker architecture (api/worker/scheduler), and two distinct audiences: API consumers (MLS proxy, GIS, search, comps) and operators (deployment, queues, env vars).

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving writing-release-notes, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.




Required wiring surfaces:
- runtime/infrastructure config: Dockerfile
- nearest typed request/context boundary
- handler/procedure boundary before external side effects

Side-effect barrier:
- Place guards before external APIs, auth mutations, email sends, analytics events, storage writes, and database mutations.


Fallback policy:
- Prefer provider-native/platform-managed primitives by default when no explicit override exists.
- Follow clear user/project overrides, but mention the native alternative and tradeoff.
- Fallbacks must be durable, multi-instance safe, and atomic under concurrency.

Verification rules:
- [error] native-or-explicit-override: Use the provider-native primitive first unless the user/project explicitly overrides it.
- [error] atomic-fallback: Fallback counters must be atomic under concurrency.

## Quick Start

### Parse commits into release sections

```bash
# new code to add — extract conventional commits since a ref
git log --oneline --no-merges v1.2.0..HEAD
```

### Map scopes to product surfaces

| Commit scope | Product surface | Audience |
|-------------|----------------|----------|
| `sync`, `bridge`, `spark`, `mls` | MLS Replication & Proxy | API consumers, operators |
| `scheduler`, `queue`, `worker` | Background Processing | operators |
| `search`, `comps` | Search & Analytics | API consumers |
| `gis` | GIS & Parcels | API consumers |
| `dashboard`, `auth` | Dashboard & Auth | dashboard users, operators |
| `docker`, `ci`, `go` | Infrastructure | operators |
| `docs` | Documentation | all |

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Conventional commits | Parse `type(scope): message` to auto-categorize | `feat(sync): fair MLS replication pipeline` |
| Product surface mapping | Group changes by affected API area | GIS, MLS Proxy, Dashboard |
| Audience tiers | Separate notes for API consumers vs operators | new endpoint vs new env var |
| Breaking changes | Flag API-breaking or deployment-breaking changes prominently | auth token format change |
| Deployment impact | Note migration, env var, queue changes for operators | new `SCHEDULER_LEADER_LOCK_ID` |

## Common Patterns

### Pattern: Release note structure

**When:** Writing the final release document.

```markdown
## [version] — YYYY-MM-DD

### Breaking Changes
- **auth**: Re-issued API keys required (SHA-256 tokens)

### Features
- **MLS Proxy**: Fair replication pipeline for Bridge and Spark
- **GIS**: Parcel teaser endpoint with coordinate rounding
- **Search**: BPO engine for comps analysis

### Fixes
- **Bridge**: Hydrate Rooms, UnitTypes, OpenHouses after replication

### Infrastructure
- Scheduler advisory lock for multi-DC deployments (`SCHEDULER_LEADER_LOCK_ID`)
- Proxy cache purge job (15-minute interval)
```

### Pattern: Generate release notes from git range

**When:** Preparing notes for a tagged release or staging deploy.

```bash
# new code to add
# 1. Collect commits
git log --oneline --no-merges v1.0.0..HEAD

# 2. Check for breaking changes
git log --oneline v1.0.0..HEAD | grep -i 'break\|remov\|deprecat\|rename'

# 3. Categorize: feat/fix → user-facing; chore/refactor → operator-facing; docs → all
```

## See Also

- [activation-onboarding](references/activation-onboarding.md)
- [engagement-adoption](references/engagement-adoption.md)
- [in-app-guidance](references/in-app-guidance.md)
- [product-analytics](references/product-analytics.md)
- [roadmap-experiments](references/roadmap-experiments.md)
- [feedback-insights](references/feedback-insights.md)

## Related Skills

- See the **deploy-coolify** skill for deployment-specific release notes
- See the **queue-postgresql** skill for queue/job changes in releases
- See the **geospatial** skill for GIS feature release language
- See the **auth-api-token** skill for auth-breaking change documentation
- See the **go** skill for Go-specific change documentation