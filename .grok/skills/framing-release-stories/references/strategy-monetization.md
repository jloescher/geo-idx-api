# Strategy and Monetization Reference

Positioning and monetization context for Quantyra IDX API release stories.

## Contents
- Product Positioning
- Monetization Signals
- Competitive Positioning in Releases
- Anti-Patterns

## Product Positioning

Quantyra IDX API is positioned as: **MLS infrastructure for real estate IDX sites**. Key differentiators:

| Differentiator | Technical basis | Story angle |
|---------------|-----------------|-------------|
| Multi-MLS unification | `?dataset=stellar\|beaches` param | "One API for multiple MLS feeds" |
| PostGIS-backed search | `internal/service/search/postgis.go` | "Sub-second geospatial queries" |
| No Redis dependency | PostgreSQL job queue | "Simpler infrastructure, fewer moving parts" |
| Multi-DC | NYC + ATL with Patroni | "Regional failover without downtime" |
| Image delivery | NVMe filesystem cache + nginx proxy | "Fast MLS photo delivery" |

### Positioning in release stories

Frame each release against these differentiators:

```markdown
## [Feature] — [Value statement]

**Why this matters:** [Connect to a differentiator above]

Before: [How users accomplished this before, or couldn't]
After: [New capability with the feature]

**Example:**
[Working curl or HTTP example]
```

## Monetization Signals

### WARNING: No usage-based metering

**The Problem:** The codebase tracks API calls via `audit_logs` but has no usage metering, rate limiting by tier, or billing integration. All tokens have `idx:full` scope.

**Why This Breaks:** Release stories cannot be tied to revenue impact. You cannot demonstrate ROI from feature launches without correlating API usage to customer value.

**The Fix:** Use `audit_logs` as a proxy for usage intensity. Track per-token request volume before and after feature announcements to demonstrate adoption.

### Available monetization proxies

| Proxy | Source | What it indicates |
|-------|--------|-------------------|
| Active tokens per domain | `personal_access_tokens` + `domains` | Customer depth |
| Request volume per token | `audit_logs` | API dependency |
| Dataset diversity | `listings.dataset_slug` usage in search | Multi-MLS value capture |
| Image cache hit rate | `X-IDX-Cache` header | Infrastructure efficiency |

## Competitive Positioning in Releases

When framing release stories against competitors (direct MLS API access, other IDX providers):

| Competitive angle | Release story framing |
|-------------------|----------------------|
| "We could just call Bridge directly" | "idx-api normalizes Bridge and Spark into one schema — no separate integrations" |
| "Other IDX providers have more feeds" | "Deep integration: PostGIS mirror, GIS parcels, comps BPO in one service" |
| "We need real-time data" | "Minute-by-minute replication with fair pipeline (no provider starvation)" |
| "Cost of running our own stack" | "Three Docker containers, no Redis, PostgreSQL-native queue" |

### Release positioning template

```markdown
## [Feature] makes [job-to-be-done] [faster/easier/more reliable]

Developers integrating MLS data typically [describe the pain point].
With [feature], you can [describe the new capability].

Unlike [competitor approach], idx-api [differentiator].

\`\`\`bash
curl -X POST $IDX_API/api/v1/search \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query": "example"}'
\`\`\`
```

## Anti-Patterns

### WARNING: Feature lists without value framing

**The Problem:** Release notes that enumerate shipped commits without translating to customer value.

**Why This Breaks:** "feat(sync): fair MLS replication pipeline" means nothing to a developer choosing between idx-api and a competitor. They need to hear "Bridge and Beaches data never waits on each other."

**The Fix:** Every feature in a release story needs a "Why this matters" sentence connecting the technical change to the positioning differentiator.

### WARNING: Announcing infra improvements as customer features

**The Problem:** Telling API consumers about scheduler advisory locks, goose migrations, or Docker layer caching.

**Why This Breaks:** Developers don't choose APIs based on infrastructure internals. These belong in operator release notes, not customer-facing stories.

**The Fix:** Split every release into two sections: "For developers" (API features, data access) and "For operators" (infra, deploy, config). See `docs/INDEX.md` for the audience-appropriate doc structure already used in this project.

See the **writing-release-notes** skill for commit categorization by audience.
See the **deploy-coolify** skill for operator-facing deploy communication.
See the **geospatial** skill for GIS feature positioning.