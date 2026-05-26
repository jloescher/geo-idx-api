---
name: framing-release-stories
description: |
  Builds launch narratives, rollout checklists, and release marketing assets for Quantyra IDX API.
  Use when: planning a feature launch, drafting customer-facing release announcements, preparing rollout communications, building launch checklists for MLS/GIS/search/dashboard features, framing technical changes as customer value stories, or responding to "launch plan", "rollout", "release announcement", "go-to-market", "ship comms".
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Framing Release Stories Skill

Turn shipped commits into customer-facing narratives for Quantyra IDX API. This API-first B2B product serves real estate developers integrating MLS data — launch stories must translate infrastructure wins (multi-DC replication, scheduler locks) into developer value (faster listings, fewer outages).

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving framing-release-stories, verify against current docs FIRST:



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

### Extract shippable features from git history

```bash
# new code to add — get conventional commits since last release
git log --oneline --no-merges v1.2.0..HEAD
```

### Map commit scopes to customer value

| Commit scope | Customer story angle | Audience |
|-------------|---------------------|----------|
| `sync`, `bridge`, `spark` | "Faster listing updates, fewer gaps" | API consumers |
| `search`, `comps` | "New property analysis capabilities" | API consumers |
| `gis` | "Parcel data now in your search results" | API consumers |
| `dashboard`, `auth` | "Easier setup and key management" | Dashboard users |
| `scheduler`, `queue` | "More reliable background processing" | operators |
| `docker`, `ci`, `go` | "Faster deploys, smaller images" | operators |

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Value translation | Turn `feat(sync): fair replication` into "Listings update faster with no provider starvation" | MLS Proxy narrative |
| Audience split | API consumers see feature benefits; operators see infra wins | Two-section release |
| Breaking change framing | Lead with migration path, not the break | "Re-issue API keys from /dashboard" |
| Rollout gate | Deploy order: workers → schedulers → APIs → idx-images | Multi-DC checklist |

## Common Patterns

### Pattern: Release story template

**When:** Preparing a customer-facing release announcement.

```markdown
## What's new in [version]

### For developers integrating MLS data
- **[Feature]** — [Value statement]. [Endpoint or config reference].
  Before: [old behavior]. After: [new behavior].

### For operators running idx-api
- **[Infra change]** — [Operational benefit]. [Env var or deploy step].

### Action required
- [Breaking change with migration steps and deadline]

### Upgrade path
[Deployment steps or link to docs/coolify-deployment.md]
```

### Pattern: Rollout checklist

**When:** Planning the deployment sequence for a release.

```markdown
Copy this checklist and track progress:
- [ ] Pre-flight: `go test ./...` passes
- [ ] Migration: `goose -dir migrations up` on primary
- [ ] Deploy workers (all DCs)
- [ ] Deploy schedulers (confirm one leader in logs)
- [ ] Deploy APIs
- [ ] Deploy idx-images
- [ ] Verify: `GET /healthz` and `GET /readyz` on each DC
- [ ] Smoke: `POST /api/v1/search` returns listings
- [ ] Monitor: replication kickoff in scheduler logs
- [ ] Post-deploy: update release notes in docs/
- [ ] Customer comms: send announcement to dashboard users
```

## See Also

- [conversion-optimization](references/conversion-optimization.md)
- [content-copy](references/content-copy.md)
- [distribution](references/distribution.md)
- [measurement-testing](references/measurement-testing.md)
- [growth-engineering](references/growth-engineering.md)
- [strategy-monetization](references/strategy-monetization.md)

## Related Skills

- See the **writing-release-notes** skill for commit-to-changelog generation
- See the **deploy-coolify** skill for deployment-specific rollout steps
- See the **queue-postgresql** skill for queue/job change communication
- See the **geospatial** skill for GIS feature narrative framing
- See the **auth-api-token** skill for auth change messaging
- See the **frontend-design** skill for dashboard update narratives