# Cache Postgres — Patterns

## Contents
- Partition and Fingerprint Strategy
- TTL Dispatch by Partition Suffix
- On-Demand Cache with Upsert
- Purge via Scheduled Job
- Anti-Patterns

## Partition and Fingerprint Strategy

Every cached response is identified by a composite key: `(partition_key, fingerprint)`.

**Partition** groups entries by tenant and route type:

```go
// internal/service/cache/canonical.go — existing partition functions
func WebPartition(domainSlug, feedCode, auditType string) string {
    return fmt.Sprintf("%s:%s:web:%s", domainSlug, feedCode, auditType)
}

func ResoPartition(domainSlug, feedCode, entity string) string {
    return fmt.Sprintf("%s:%s:reso:%s", domainSlug, feedCode, entity)
}

func SearchPartition(domainSlug, feedCode string) string {
    return fmt.Sprintf("%s:%s:search", domainSlug, feedCode)
}

func LookupPartition(domainSlug, feedCode string) string {
    return fmt.Sprintf("%s:%s:lookup", domainSlug, feedCode)
}
```

**Fingerprint** is a SHA-256 hash of the HTTP method, upstream path, sorted query params (excluding `domain`), and POST body. This means identical requests from different domains share the same fingerprint but sit in different partitions.

```go
// internal/service/cache/canonical.go
func FingerprintRequest(c *fiber.Ctx, upstreamPath string) string {
    h := sha256.New()
    fmt.Fprintf(h, "%s\n%s\n", c.Method(), upstreamPath)
    // sorted query params (skipping "domain"), then body
    return hex.EncodeToString(h.Sum(nil))
}
```

### DO: Use canonical partition functions

```go
partition := cache.WebPartition(domainSlug, feedCode, "listings.collection")
```

### DON'T: Hardcode partition strings

```go
// BAD — fragile, bypasses TTL dispatch
partition := domainSlug + ":web:listings"
```

**Why:** `TTLForPartition` relies on the `:lookup` suffix convention. Hardcoded strings bypass this logic and silently get the wrong TTL.

## TTL Dispatch by Partition Suffix

TTL is determined at read time by inspecting the partition string suffix:

```go
// internal/service/cache/proxy_cache.go
func (p *ProxyCache) TTLForPartition(partition string) time.Duration {
    if stringsHasSuffix(partition, ":lookup") {
        return p.cfg.Bridge.LookupCacheTTL  // 720h (30 days) default
    }
    return p.cfg.Bridge.ListingsCacheTTL     // 900s (15 min) default
}
```

| Partition suffix | TTL | Config env | Default |
|-----------------|-----|------------|---------|
| `:lookup` | 30 days | `MLS_LOOKUP_CACHE_TTL` | `720h` |
| Everything else | 15 minutes | `LISTINGS_CACHE_TTL` | `900s` |

### WARNING: Adding a new TTL tier

If you need a third TTL tier (e.g., 1 hour for GIS), add a new suffix check in `TTLForPartition` and a new config field. Do NOT override TTL per-request — the partition convention keeps TTL logic centralized and testable.

## On-Demand Cache with Upsert

Cache writes use `ON CONFLICT DO UPDATE` — always idempotent:

```go
// internal/service/cache/proxy_cache.go
func (p *ProxyCache) Put(ctx context.Context, partition, fingerprint string, body []byte) error {
    compressed, err := gzipBytes(body)
    if err != nil { return err }
    _, err = p.db.Pool.Exec(ctx, `
        INSERT INTO mls_search_cache (partition_key, fingerprint, compressed_data, last_updated)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (partition_key, fingerprint) DO UPDATE
        SET compressed_data = EXCLUDED.compressed_data, last_updated = NOW()
    `, partition, fingerprint, compressed)
    return err
}
```

Cache errors on `Put` are intentionally ignored in handlers (`_ = h.proxyCache.Put(...)`) — a failed cache write degrades to cache miss, not a 500.

### WARNING: Never propagate cache errors to clients

```go
// BAD — cache failure becomes a user-facing error
if err := h.proxyCache.Put(ctx, partition, fp, body); err != nil {
    return fiber.NewError(500, "cache write failed")
}

// GOOD — cache is a performance optimization, not a dependency
_ = h.proxyCache.Put(ctx, partition, fp, body)
```

**Why:** The proxy works correctly without cache. A cache write failure means the next request hits upstream again — acceptable degradation. Propagating cache errors causes cascading failures that are hard to debug.

## Purge via Scheduled Job

Stale rows are purged by a scheduled job, not by per-row TTL:

```go
// internal/scheduler/scheduler.go — every 15 minutes
s.addJob(ctx, "mls-proxy-cache-purge", "0 */15 * * * *", "default", queue.TypeMLSProxyCachePurge)

// internal/service/cache/proxy_cache.go
func (p *ProxyCache) PurgeExpired(ctx context.Context) (int64, error) {
    days := p.cfg.MLS.ProxyCacheRetentionDays  // default 30
    if days <= 0 { days = 30 }
    cutoff := time.Now().AddDate(0, 0, -days)
    tag, err := p.db.Pool.Exec(ctx, `DELETE FROM mls_search_cache WHERE last_updated < $1`, cutoff)
    return tag.RowsAffected(), nil
}
```

Config: `MLS_PROXY_CACHE_RETENTION_DAYS` (default 30).

### WARNING: Do not add per-row TTL columns

PostgreSQL does not have native per-row TTL like DynamoDB. The project's approach — bulk `DELETE WHERE last_updated < cutoff` on a schedule — is correct. Adding a `expires_at` column sounds cleaner but adds index overhead and migration complexity for no functional gain.

## Anti-Patterns

### WARNING: In-memory cache for MLS responses

```go
// BAD — lost on restart, not shared across DCs
var responseCache sync.Map

// GOOD — use ProxyCache which persists to PostgreSQL
proxyCache.Get(ctx, partition, fingerprint)
```

**Why:** This system runs in multi-DC (NYC + ATL) with multiple API replicas per DC. In-memory cache is lost on restart, not shared between instances, and causes thundering herd to upstream MLS APIs.

### WARNING: Cache without image rewriting

```go
// BAD — stores raw upstream URLs that may break for end users
h.proxyCache.Put(ctx, partition, fp, rawUpstreamBody)

// GOOD — rewrite image URLs before caching (existing pattern in finishProxy)
body = images.RewriteBytes(h.rewriter, body, feed.Dataset, listingKey)
h.proxyCache.Put(ctx, partition, fp, body)
```

**Why:** Upstream image URLs point to Bridge/Spark CDN hosts. The rewriter maps these to the idx-images proxy. Caching pre-rewrite means clients get broken image URLs that never get fixed.