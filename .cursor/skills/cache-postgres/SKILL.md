---
name: cache-postgres
description: |
  Manages PostgreSQL-based gzip cache (mls_search_cache) for MLS proxy responses.
  Use when: modifying cache TTL, adding cache partitions, implementing purge logic,
  integrating cache into new handlers, debugging X-IDX-Cache headers, or tuning
  MLS_PROXY_CACHE_RETENTION_DAYS / LISTINGS_CACHE_TTL / MLS_LOOKUP_CACHE_TTL.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Cache Postgres Skill

PostgreSQL-backed gzip cache for MLS proxy responses. Stores compressed upstream JSON in `mls_search_cache`, keyed by partition + SHA-256 fingerprint. On-demand only — no pre-warm. Purged every 15 minutes by a scheduled worker job.

This is **not** a Redis-style TTL cache. Rows are TTL-checked at read time (`time.Since(lastUpdated) > ttl`) and bulk-purged by the scheduler. There is no background expiry thread.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving cache-postgres, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.

Risk signals this skill can participate in:
- cache/shared state: Avoid module-level mutable state for serverless or multi-instance code. Use a provider or database primitive with clear concurrency behavior.
- database/concurrency: Prefer atomic statements, unique constraints, transactions, or provider primitives for coordination. Avoid select-then-insert/update counters unless protected by a lock or constraint. For state flips, use conditional writes such as UPDATE ... WHERE field IS NULL RETURNING instead of read-then-update. For relation creation such as organization membership, add a database uniqueness invariant and an idempotent insert/upsert path.



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
- [warning] relational-uniqueness-invariant: Membership/link/ownership creation should use a database uniqueness invariant plus idempotent insert/upsert behavior.

## Quick Start

### Verified Existing Pattern

```go
// internal/handler/bridge/handler.go — proxyCacheStore interface
type proxyCacheStore interface {
    Get(ctx context.Context, partition, fingerprint string) ([]byte, bool, error)
    Put(ctx context.Context, partition, fingerprint string, body []byte) error
}
```

```go
// internal/service/cache/proxy_cache.go — TTL-aware Get
func (p *ProxyCache) Get(ctx context.Context, partition, fingerprint string) ([]byte, bool, error) {
    ttl := p.TTLForPartition(partition)
    // SELECT compressed_data, last_updated ... WHERE partition_key=$1 AND fingerprint=$2
    if time.Since(lastUpdated) > ttl { return nil, false, nil }
    return gunzip(compressed)
}
```

### New Code Pattern

```go
// new code to add — new partition type for a custom route
func CustomPartition(domainSlug, feedCode, routeType string) string {
    return fmt.Sprintf("%s:%s:custom:%s", domainSlug, feedCode, routeType)
}
```

## Key Concepts

| Concept | Usage | Example |
|---------|-------|---------|
| Partition key | Groups cache entries by domain+feed+route type | `WebPartition("acme", "bridge_stellar", "listings.collection")` |
| Fingerprint | SHA-256 of method+path+sorted query+body | `FingerprintRequest(c, upstream)` |
| TTL dispatch | `:lookup` suffix → 30-day TTL; everything else → 15 min | `TTLForPartition(partition)` |
| Gzip storage | BYTEA column holds compressed JSON | `gzipBytes(body)` / `gunzip(compressed)` |
| Upsert | `ON CONFLICT DO UPDATE` — idempotent Put | `Put()` always safe to call |

## Common Patterns

### Cache-aware proxy handler

**When:** Adding a new MLS proxy route that should cache responses.

```go
// internal/handler/bridge/handler.go:finishProxy — existing pattern
fp := cache.FingerprintRequest(c, upstream)
if body, ok, err := h.proxyCache.Get(c.Context(), partition, fp); err == nil && ok {
    c.Set("X-IDX-Cache", "HIT")
    return c.Status(fiber.StatusOK).Send(body)
}
// ... upstream fetch ...
_ = h.proxyCache.Put(c.Context(), partition, fp, body)
c.Set("X-IDX-Cache", "MISS")
```

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **go** skill for Go code conventions
- See the **fiber** skill for HTTP handler patterns
- See the **queue-postgresql** skill for job queue integration
- See the **deploy-coolify** skill for multi-DC cache considerations
- See the **postgres** skill for database query patterns