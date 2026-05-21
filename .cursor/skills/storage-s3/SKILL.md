---
name: storage-s3
description: |
  Configures S3-compatible object storage for image caching in the Quantyra IDX API.
  Use when: adding or modifying S3/MinIO/R2 object storage for image cache, migrating
  from filesystem cache to object storage, configuring multi-DC shared image cache,
  setting up presigned URLs for image delivery, or troubleshooting image storage issues.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Storage S3 Skill

S3-compatible object storage for the image proxy cache layer. The current implementation (`internal/handler/images/proxy.go`) uses an NVMe filesystem cache with SHA-256 key hashing. This skill covers adding a shared S3 backend so multi-DC deployments (NYC + ATL) share one image cache instead of maintaining separate local caches per region.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving storage-s3, verify against current docs FIRST:



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

### Existing Filesystem Pattern

```go
// internal/handler/images/proxy.go — current NVMe cache
func (h *Handler) cacheFile(listingKey, photoID string) string {
    sum := sha256.Sum256([]byte(listingKey + "/" + photoID))
    name := hex.EncodeToString(sum[:]) + ".bin"
    return filepath.Join(h.cfg.Images.Path, name[:2], name)
}
```

### New S3 Config Extension

```go
// new code to add — extend ImageCacheConfig
type S3Config struct {
    Endpoint    string
    Region      string
    Bucket      string
    AccessKey   string
    SecretKey   string
    PresignTTL  time.Duration
    PathStyle   bool // true for MinIO/R2
}

type ImageCacheConfig struct {
    Path string
    TTL  time.Duration
    S3   S3Config // new code to add
}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| SHA-256 cache key | Deterministic key from `listingKey/photoID` | `sha256("stellar:abc/1")` → `a1b2...bin` |
| Presigned GET URL | Time-limited direct S3 access, no proxy | `s3.PresignGetObject(ctx, 15*time.Minute)` |
| Dual-tier cache | S3 primary, filesystem L2 | Check S3 → fallback to disk → fetch upstream |
| Path-style addressing | Required for MinIO/R2 | `https://s3.example.com/bucket/key` |

## Common Patterns

### S3 Cache Read-Through

**When:** Replacing filesystem-only cache with shared S3 backend

```go
// new code to add — S3-aware Show handler
func (h *Handler) Show(c *fiber.Ctx) error {
    key := h.s3Key(listingKey, photoID)
    // Try S3 first (shared across DCs)
    if url, err := h.s3Client.PresignGET(c.Context(), key, h.cfg.Images.S3.PresignTTL); err == nil {
        return c.Redirect(url, fiber.StatusFound)
    }
    // Fallback to upstream fetch + S3 put
    // ... existing upstream fetch logic ...
    go h.s3Client.PutObject(c.Context(), key, body, contentType)
    return c.Send(body)
}
```

### Cache Key Mapping

**When:** Mapping filesystem SHA-256 keys to S3 object keys

```go
// new code to add — consistent key between disk and S3
func (h *Handler) s3Key(listingKey, photoID string) string {
    sum := sha256.Sum256([]byte(listingKey + "/" + photoID))
    return hex.EncodeToString(sum[:]) + ".bin"
}
```

## WARNING: Silent Cache Write Failures

**The Problem:**

```go
// BAD — errors silently discarded, cache never populates
_ = os.WriteFile(cachePath, body, 0o644)
go h.s3Client.PutObject(ctx, key, body, ct) // fire-and-forget goroutine
```

**Why This Breaks:**
1. Disk errors (ENOSPC, permissions) are invisible — every request hits upstream
2. Goroutine has no error channel — S3 failures are permanently lost
3. No retry or backfill mechanism when storage recovers

**The Fix:**

```go
// GOOD — log errors, allow monitoring to catch persistent failures
if err := os.WriteFile(cachePath, body, 0o644); err != nil {
    slog.Error("disk cache write failed", "path", cachePath, "err", err)
}
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **go** skill for Go error handling and concurrency patterns
- See the **docker** skill for container volume mounts and S3 sidecar config
- See the **deploy-coolify** skill for multi-DC deployment topology
- See the **cache-postgres** skill for PostgreSQL-backed caching patterns
- See the **proxy-web** skill for the MLS proxy layer that feeds images upstream