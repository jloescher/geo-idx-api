# Content Copy Reference

Write release copy for Quantyra IDX API — developer-focused, B2B, and MLS-compliant.

## Contents
- Copy Surfaces
- Tone and Terminology
- Audience-Specific Copy
- Anti-Patterns

## Copy Surfaces

| Surface | Location | Updates with release |
|---------|----------|---------------------|
| Hero tagline | `internal/handler/marketing/handler.go` | New capability headlines |
| Dashboard cards | `internal/handler/dashboard/handler.go` | Setup flow copy changes |
| API docs | `docs/*.md` | New endpoints, changed params |
| Release notes | `docs/` or git tags | Full changelog |
| OpenAPI spec | `docs/yaak-api-collection.json` | New routes, schemas |

## Tone and Terminology

This is a **developer API** for real estate IDX integrations. Copy rules:

| Rule | Example |
|------|---------|
| Lead with API benefit, not infra detail | "Search listings by bounding box" not "PostGIS spatial index" |
| Use RESO standard field names in examples | `ListPrice`, `StandardStatus`, `ModificationTimestamp` |
| Specify dataset when MLS-specific | "Stellar (Bridge)" or "Beaches (Spark)" — never just "MLS" |
| Include curl or HTTP examples | Every feature announcement needs a working request |
| Name env vars for operator copy | `SCHEDULER_LEADER_LOCK_ID`, `WORKER_QUEUES` |

### Terminology consistency

| Use | Don't use |
|-----|-----------|
| dataset | feed/source (in API context) |
| replication | sync/crawl |
| token | key/secret (in PAT context) |
| listing | property/record (in mirror context) |
| persist | save/store (in worker context) |

## Audience-Specific Copy

### API consumers (developers integrating IDX)

```markdown
**New:** Search listings by polygon boundary

POST /api/v1/search now accepts a `polygon` parameter for custom area searches.

\`\`\`bash
curl -X POST https://idx-api.quantyralabs.cc/api/v1/search \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"polygon": [[-80.1,25.8],[-80.1,25.9],[-80.0,25.9],[-80.0,25.8]]}'
\`\`\`
```

### Operators (running idx-api infrastructure)

```markdown
**Changed:** Scheduler leader lock ID is now configurable

Set `SCHEDULER_LEADER_LOCK_ID` (default: 913374211) to avoid collisions when
running multiple scheduler pairs. Required for multi-DC deployments.
See docs/coolify-deployment.md §7.
```

## Anti-Patterns

### WARNING: Jargon without context

**The Problem:** Release copy that says "fair replication pipeline" without explaining what changed for the user.

**Why This Breaks:** API consumers don't care about queue fairness internals. They care that "Bridge listings no longer delay Beaches updates."

**The Fix:** Always write the user-facing consequence first, then the technical detail for operators who need it.

### WARNING: Marketing copy that ignores MLS compliance

**The Problem:** Announcing MLS data features without noting compliance requirements. Spark (Beaches) has display rules documented in `docs/spark/spark-compliance.md`.

**Why This Breaks:** Customers who implement based on launch copy may violate MLS display rules, creating legal exposure.

**The Fix:** Include a compliance note when announcing MLS data features: "Review display requirements in docs/spark/spark-compliance.md before going live."

## Release copy checklist

Copy this checklist and track progress:
- [ ] Feature name is clear and customer-beneficial
- [ ] Includes working HTTP/curl example
- [ ] Specifies which datasets are affected (stellar, beaches, or both)
- [ ] Breaking changes lead with migration path
- [ ] Compliance notes included for MLS data features
- [ ] Operator copy names exact env vars and config
- [ ] No internal code references (package names, Go types) in customer copy

See the **writing-release-notes** skill for structured changelog generation.
See the **frontend-design** skill for dashboard copy updates.