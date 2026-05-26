# Content Copy Reference

## Contents
- Voice and Tone
- Copy Surfaces in the Codebase
- API Error Messages as Copy
- Release Notes Structure
- Anti-Patterns

## Voice and Tone

Quantyra IDX API targets developers and technical real estate professionals. Copy rules:

1. **Direct**: Lead with what the endpoint returns, not why it matters philosophically
2. **Technical**: Use RESO field names (`ListingKey`, `StandardStatus`), not invented abstractions
3. **Concise**: One sentence per concept. No filler paragraphs.

## Copy Surfaces in the Codebase

| Surface | Location | Copy type |
|---------|----------|-----------|
| API docs | `docs/*.md` | Endpoint reference, parameters, examples |
| Dashboard | `internal/web/static/` | In-app messaging, onboarding, empty states |
| Release notes | `docs/` | Feature changelog |
| Health endpoints | `cmd/api/` | Status messages (`/healthz`, `/readyz`) |
| Error responses | `internal/handler/` | API error JSON |

## API Error Messages as Copy

Error messages are read by developers integrating the API. They are copy.

```go
// BAD - Generic, no action path
{"error": "Unauthorized"}

// GOOD - Specific, tells developer what to fix
{"error": "Invalid API token. Re-issue at /dashboard"}
```

## Release Notes Structure

Follow the git commit convention already in use (see `git log`):

```markdown
## [Date] — [scope]: [imperative description]

**What changed:** 1-2 sentences

**API impact:** Breaking / Non-breaking / New endpoint

**Migration:** Required steps (if any)
```

Commit message prefixes from the repo: `feat()`, `fix()`, `chore()`. Mirror these in release note sections.

## Anti-Patterns

### WARNING: Marketing Jargon in API Docs

**The Problem:** Developers scanning docs for `POST /api/v1/search` parameters do not care about "revolutionary" or "next-generation" anything.

**Why This Breaks:** Jargon adds zero information and signals that the docs are maintained by non-technical writers. Developer trust drops.

**The Fix:** Use the exact RESO field names and HTTP methods. "Returns Active and Pending listings filtered by PostGIS bounding box" is complete.

### WARNING: Inconsistent Dataset Naming

**The Problem:** Mixing "Bridge" and "Stellar" or "Spark" and "Beaches" without context confuses readers.

**The Fix:** Use the convention from `README.md`: `bridge_stellar` (or just "Stellar"), `spark_beaches` (or just "Beaches"). The `?dataset=` parameter uses `stellar` and `beaches`. Copy must match.