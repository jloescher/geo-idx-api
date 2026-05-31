# Proxy Web — Patterns Reference

## Contents
- Factory Pattern for Multi-Provider Selection
- Cache Fingerprinting and Partitioning
- Image Proxy with Disk Cache
- Auth Middleware Chain
- Upstream URL Construction
- Anti-Patterns

---

## Factory Pattern for Multi-Provider Selection

The `mlspoxy.Factory` selects Bridge or Spark client based on the feed definition set by `MLSAccess` middleware. Never construct clients directly in handlers.

```go
// internal/mlspoxy/factory.go
func (f *Factory) ForRequest(c *fiber.Ctx) ProxyClient {
    def, _ := c.Locals(ctxkeys.MLSFeedDef).(dom.FeedDefinition)
    if def.Provider == "spark" {
        return &sparkWrapper{spark.NewClient(f.cfg)}
    }
    return &bridgeWrapper{bridge.NewClient(f.cfg, def)}
}
```

**DO:** Use the factory from `Handler` to get the correct client.
**DON'T:** Hard-code `bridge.Client{}` in handler methods — Spark requests will fail silently.

---

## Cache Fingerprinting and Partitioning

Every proxied request is fingerprinted with SHA-256 (method + upstream path + sorted query params + POST body). The `domain` query parameter is excluded to prevent cache poisoning.

```go
// internal/service/cache/canonical.go
func FingerprintRequest(c *fiber.Ctx, upstreamPath string) string {
    h := sha256.New()
    _, _ = fmt.Fprintf(h, "%s\n%s\n", c.Method(), upstreamPath)
    keys := make([]string, 0, len(c.Queries()))
    for k := range c.Queries() {
        if strings.EqualFold(k, "domain") { continue }
        keys = append(keys, k)
    }
    sort.Strings(keys)
    for _, k := range keys {
        _, _ = fmt.Fprintf(h, "%s=%s\n", k, c.Query(k))
    }
    if len(c.Body()) > 0 { h.Write(c.Body()) }
    return hex.EncodeToString(h.Sum(nil))
}
```

Partitions scope cache by domain and route type:

| Partition function | Format | TTL |
|---|---|---|
| `WebPartition` | `domain:feed:web:auditType` | `LISTINGS_CACHE_TTL` (15m) |
| `ResoPartition` | `domain:feed:reso:entity` | `LISTINGS_CACHE_TTL` (15m) |
| `LookupPartition` | `domain:feed:lookup` | `MLS_LOOKUP_CACHE_TTL` (30d) |
| `SearchPartition` | `domain:feed:search` | `LISTINGS_CACHE_TTL` (15m) |

**DO:** Use `LookupPartition` for rarely-changing metadata (field lists, enumerations).
**DON'T:** Use `ResoPartition` for lookup endpoints — you'll burn through cache with short TTLs on stable data.

---

## Image Proxy with Disk Cache

Images use NVMe filesystem cache, not PostgreSQL. SHA-256 key → 2-character directory sharding prevents single-directory limits.

```go
// internal/handler/images/proxy.go
func (h *Handler) cacheFile(listingKey, photoID string) string {
    sum := sha256.Sum256([]byte(listingKey + "/" + photoID))
    name := hex.EncodeToString(sum[:]) + ".bin"
    return filepath.Join(h.cfg.Images.Path, name[:2], name)
}
```

**DO:** Set `Cache-Control: public, max-age=31536000, immutable` — images are content-addressable.
**DON'T:** Store images in PostgreSQL — binary BLOBs bloat the database and add I/O overhead.

---

## Auth Middleware Chain

Proxy routes require both `DomainToken` and `MLSAccess` middleware in order:

1. `DomainToken` — domain header/Referer or Bearer PAT with TXT-verified domain
2. `MLSAccess` — resolves `?dataset=` to feed code + `FeedDefinition`, checks domain allowlist

```go
// internal/api/middleware/mls_access.go
func MLSAccess(cfg config.Config, _ *repository.DomainRepo) fiber.Handler {
    resolver := mls.NewResolver(cfg)
    return func(c *fiber.Ctx) error {
        if mls.BypassGIS(c.Path()) { return c.Next() }
        code, err := resolver.ResolveFeedCode(c)
        // sets MLSFeedCode + MLSFeedDef in Fiber locals
        c.Locals(ctxkeys.MLSFeedCode, code)
        c.Locals(ctxkeys.MLSFeedDef, resolver.FeedDefinition(code))
        return c.Next()
    }
}
```

**DO:** Always apply both middleware to new proxy routes.
**DON'T:** Skip `MLSAccess` — without it, `Factory.ForRequest` gets a zero-value `FeedDefinition` and defaults to Bridge regardless of `?dataset=`.

---

## Upstream URL Construction

Bridge and Spark have different URL structures. The handler delegates to provider-specific path builders.

```go
// Bridge: {HOST}/{PATH_PREFIX}/{DATASET}/reso/odata/{Entity}
upstream = bc.ResoURL(entity, ds)

// Spark: {API_HOST}/{VERSION}/{LIVE_RESO_ROOT}/{Entity}
upstream = h.cfg.Spark.APIHost + "/" + h.cfg.Spark.APIVersion + "/" + h.cfg.Spark.LiveResoRoot + "/" + entity
```

**DO:** Use the existing `proxyWeb`, `proxyReso`, `proxyResoLookup` methods — they handle URL construction per provider.
**DON'T:** Build URLs manually in new handlers. Add a new `proxy*` method if the URL pattern differs.

---

## Anti-Patterns

### WARNING: In-Memory Cache for Proxy Responses

**The Problem:** Storing proxy responses in a Go map or sync.Pool appears fast locally but causes cache stampede, stale data, and OOM in production with multiple replicas.

**Why This Breaks:** Two API replicas don't share the in-memory cache. Each miss triggers an upstream call. Under load, this amplifies requests to Bridge/Spark, risking rate-limit bans.

**The Fix:** Use `ProxyCache` (PostgreSQL-backed, gzip-compressed). Multi-DC replicas share the same cache rows.

### WARNING: Skipping the Factory for Client Selection

**The Problem:** Directly constructing `bridge.Client{}` bypasses Spark support. Spark requests return 404 or wrong data.

**Why This Breaks:** The `MLSAccess` middleware resolves `?dataset=beaches` → Spark provider, but if the handler ignores the factory, it sends Beaches queries to Bridge endpoints.

**The Fix:** Always call `h.factory.ForRequest(c)` to get the correct `ProxyClient`.

### WARNING: Forwarding Internal Params to Upstream

**The Problem:** Sending `?domain=mydomain` to Bridge/Spark leaks internal routing info and may cause upstream errors.

**Why This Breaks:** Bridge treats unknown query parameters as OData filters, producing empty or error responses.

**The Fix:** `FingerprintRequest` excludes the `domain` parameter. Ensure new proxy methods follow the same pattern — never forward internal params to upstream.

---

For queue processing and cache purging, see the **queue-postgresql** skill. For authentication flows, see the **auth-api-token** and **auth-domain** skills.