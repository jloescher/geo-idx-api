---
name: embedding-decision-cues
description: |
  Applies behavioral nudges, urgency signals, and progress mechanics to Quantyra IDX's
  invite-only dashboard and GIS teaser tier to improve setup completion and tier upgrades.
  Use when: modifying hero copy, dashboard setup flow, token issuance pages, GIS teaser
  responses, domain verification messaging, invitation flows, or any user-facing text in
  internal/handler/marketing, internal/handler/dashboard, internal/web/layout.go, or
  internal/service/gis/teaser.go.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Embedding Decision Cues

Applies behavioral economics patterns (loss aversion, progress motivation, scarcity, social proof) to the Quantyra IDX dashboard and GIS teaser tier. This is a B2B API platform — conversion means **completing domain setup** and **upgrading from staging to production access**, not traditional ecommerce checkout.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving embedding-decision-cues, verify against current docs FIRST:



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

### Verified Existing Pattern — One-Time Token Display

```go
// internal/handler/dashboard/handler.go — VerifyTXT
body := `<div class="card"><h1>Domain verified</h1>
<p>Save this production token now — it will not be shown again.</p>
<div class="token-box" id="token">` + web.Esc(plain) + `</div>
<p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
```

### Verified Existing Pattern — GIS Teaser Tier Gate

```go
// internal/service/gis/teaser.go — applies loss aversion via truncated data
func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
    if fullAccess { return geojson, false }
    // Caps features to 40, rounds coords to 4 decimal places
    truncated := truncateFeatureCollection(fc, maxFeatures)
    roundFeatureCollectionCoords(fc, decimals)
    ...
}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Loss aversion | One-time token/invite display | "will not be shown again" in `VerifyTXT` |
| Progress motivation | Setup completion flow | Domain → Verify → Token sequence |
| Scarcity / exclusivity | Invite-only registration | Admin-gated invitation links |
| Teaser tier | GIS data truncation for `idx:access` tokens | `applyTeaser()` with configurable caps |
| Status badges | Verification progress signaling | `badge-verified` / `badge-pending` CSS |

## Common Patterns

### Adding a Decision Cue to Dashboard Copy

**When:** Improving setup completion rates by adding progress signaling.

```go
// new code to add — progress step indicator in Dashboard()
b.WriteString(`<div class="setup-progress">
<span class="step active">1. Add domain</span>
<span class="step">2. Verify DNS</span>
<span class="step">3. Get API token</span>
</div>`)
```

### Enhancing the GIS Teaser Upgrade Cue

**When:** Users with staging tokens hit the feature cap and need an upgrade nudge.

```go
// new code to add — in GIS handler after applyTeaser returns truncated=true
if truncated {
    c.Set("X-GIS-Teaser", "true")
    c.Set("X-GIS-Upgrade", "Verify a domain for full parcel data")
}
```

## See Also

- [conversion-optimization](references/conversion-optimization.md)
- [content-copy](references/content-copy.md)
- [distribution](references/distribution.md)
- [measurement-testing](references/measurement-testing.md)
- [growth-engineering](references/growth-engineering.md)
- [strategy-monetization](references/strategy-monetization.md)

## Related Skills

- See the **ux** skill for dashboard interaction patterns
- See the **frontend-design** skill for CSS/layout guidance
- See the **auth-api-token** skill for token lifecycle and scopes
- See the **geospatial** skill for GIS tier configuration
- See the **go** skill for Go code patterns