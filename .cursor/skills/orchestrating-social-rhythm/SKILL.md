---
name: orchestrating-social-rhythm
description: |
  Plans social content beats and distribution rhythm for Quantyra IDX API.
  Use when: planning release announcement cadence, scheduling feature launch content, coordinating docs updates with social posts, creating editorial arcs for MLS data capabilities, structuring multi-channel content calendars for developer-facing API products.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Orchestrating Social Rhythm Skill

Plan content beats and distribution rhythm for the Quantyra IDX API — a B2B developer platform for MLS data access, GIS parcels, and property search. Content surfaces are docs, the embedded dashboard (`internal/web/static/`), release notes, and API changelog entries.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving orchestrating-social-rhythm, verify against current docs FIRST:



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

### Content Beat Inventory

```bash
# Find existing content surfaces
ls internal/web/static/
ls docs/*.md
```

### New Content Calendar Entry

```go
// new code to add — scheduled content job type
// Add to scheduler job registry alongside mls.replication_kickoff
// {"type": "content.beat_reminder", "channel": "docs", "beat": "release-notes"}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Beat | Recurring content unit tied to a schedule | Weekly API tips, monthly changelog |
| Arc | Multi-post narrative spanning several beats | "GIS parcel teaser" launch series |
| Channel | Distribution surface in this project | `docs/`, dashboard, release notes |
| Trigger | Code event that spawns a content beat | Scheduler cron, merge to `staging` |

## Common Patterns

### Release-Triggered Content Beat

**When:** A feature merges to staging and needs announcement coordination.

```markdown
1. Feature PR merges to staging
2. Update docs/INDEX.md with new doc link
3. Add release note entry to docs/
4. Schedule social post for next beat window
5. Dashboard banner update in internal/web/static/
```

### Editorial Arc for Feature Launch

**When:** A major feature (e.g., GIS parcels, comps API) needs multi-beat rollout.

```markdown
Week 1: Teaser — "Coming soon" in dashboard + docs
Week 2: Launch — Release notes, API docs, social
Week 3: Deep-dive — Technical blog / docs expansion
Week 4: Social proof — Usage examples, customer quotes
```

## See Also

- [conversion-optimization](references/conversion-optimization.md)
- [content-copy](references/content-copy.md)
- [distribution](references/distribution.md)
- [measurement-testing](references/measurement-testing.md)
- [growth-engineering](references/growth-engineering.md)
- [strategy-monetization](references/strategy-monetization.md)

## Related Skills

- See the **frontend-design** skill for dashboard marketing page layouts
- See the **ux** skill for dashboard empty states and in-app messaging
- See the **go** skill for scheduler job registration patterns
- See the **deploy-coolify** skill for deployment timing coordination