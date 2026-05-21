---
name: proxy-web
description: |
  Configures web proxy for MLS API integration with caching and rate limiting.
  Use when: adding or modifying MLS proxy routes, cache behavior, upstream client logic,
  image rewriting, request fingerprinting, feed selection, or proxy-related middleware.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Proxy Web Skill

MLS proxy layer that forwards RESO OData and web API requests to Bridge/Spark upstreams with gzip-compressed PostgreSQL caching, SHA-256 request fingerprinting, and multi-provider factory selection. All proxy routes flow through `finishProxy` — cache check → upstream call → image rewrite → cache store → audit log.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving proxy-web, verify against current docs FIRST:



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

### Existing Proxy Flow (handler → cache → upstream → response)

```go
// internal/handler/bridge/handler.go — finishProxy is the single choke point
func (h *Handler) finishProxy(c *fiber.Ctx, auditType string, cli mlspoxy.ProxyClient,
    upstream, listingKey, partition string) error {
    fp := cache.FingerprintRequest(c, upstream)
    // 1. Cache lookup
    if body, ok, _ := h.proxyCache.Get(c.Context(), partition, fp); ok {
        c.Set("X-IDX-Cache", "HIT")
        return c.Status(fiber.StatusOK).Send(body)
    }
    // 2. Upstream call
    status, body, hdr, err := cli.Proxy(c, upstream)
    // 3. Image rewrite + cache store
    body = images.RewriteBytes(h.rewriter, body, feed.Dataset, listingKey)
    _ = h.proxyCache.Put(c.Context(), partition, fp, body)
    return c.Status(status).Send(body)
}
```

### Adding a New Proxy Route

```go
// new code to add — follow the existing pattern in handler.go
func (h *Handler) MyEntity(c *fiber.Ctx) error {
    return h.proxyReso(c, "myentity.collection", "MyEntity")
}
```

## Key Concepts

| Concept | Location | Notes |
|---|---|---|
| `ProxyClient` interface | `internal/mlspoxy/factory.go` | `Proxy()` and `ProxyUpstream()` per provider |
| Factory selection | `Factory.ForRequest()` | Reads `MLSFeedDef` from Fiber locals |
| Cache fingerprinting | `internal/service/cache/canonical.go` | SHA-256 of method + path + sorted query + body |
| Cache partitions | `WebPartition`, `ResoPartition`, `LookupPartition` | Domain-scoped TTL isolation |
| Image rewriting | `internal/mlspoxy/images/rewrite.go` | Rewrites CDN URLs to local `/images/` path |
| Auth middleware | `DomainToken` + `MLSAccess` | Domain header/Referer or Bearer PAT |
| Image disk cache | `internal/handler/images/proxy.go` | SHA-256 → 2-char dir sharding |

## Common Patterns

### Cache Partitioning by TTL

```go
// Lookup endpoints get 30-day TTL; listings get 15-minute TTL
partition := cache.LookupPartition(domainSlug, feedCode)  // long TTL
partition := cache.ResoPartition(domainSlug, feedCode, entity) // short TTL
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **go** skill for Go language patterns
- See the **fiber** skill for HTTP routing and middleware
- See the **auth-api-token** skill for token-based authentication
- See the **auth-domain** skill for domain verification
- See the **queue-postgresql** skill for job queue processing
- See the **cache-postgres** skill for PostgreSQL-backed caching