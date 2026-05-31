# Cache Postgres — Workflows

## Contents
- Adding a New Cached Route
- Changing Cache TTL
- Debugging Cache HIT/MISS
- Manual Cache Purge
- Adding a New Partition Type

## Adding a New Cached Route

Copy this checklist and track progress:

- [ ] 1. Define the route handler in the appropriate handler file
- [ ] 2. Choose or create a partition function (WebPartition, ResoPartition, or new)
- [ ] 3. Call `cache.FingerprintRequest(c, upstream)` for the fingerprint
- [ ] 4. Check `proxyCache.Get(ctx, partition, fp)` before upstream fetch
- [ ] 5. Set `X-IDX-Cache: HIT` or `MISS` response header
- [ ] 6. Rewrite images before `proxyCache.Put()`
- [ ] 7. Audit-log the cache status
- [ ] 8. Test: identical request within TTL returns HIT, different params returns MISS

```go
// new code to add — minimal cached proxy route
func (h *Handler) NewRoute(c *fiber.Ctx) error {
    cli := h.factory.ForRequest(c)
    upstream := "https://upstream.example.com/path"
    partition := cache.WebPartition(h.domainSlug(c), h.feedCode(c), "newroute")
    fp := cache.FingerprintRequest(c, upstream)

    if body, ok, err := h.proxyCache.Get(c.Context(), partition, fp); err == nil && ok {
        c.Set("X-IDX-Cache", "HIT")
        return c.Status(fiber.StatusOK).Send(body)
    }

    status, body, _, err := cli.Proxy(c, upstream)
    if err != nil { return fiber.NewError(fiber.StatusBadGateway, err.Error()) }
    if status >= 200 && status < 300 {
        body = images.RewriteBytes(h.rewriter, body, mlspoxy.Feed(c).Dataset, "")
        _ = h.proxyCache.Put(c.Context(), partition, fp, body)
        c.Set("X-IDX-Cache", "MISS")
    }
    c.Set("Content-Type", "application/json")
    return c.Status(status).Send(body)
}
```

## Changing Cache TTL

1. Identify which TTL tier: `LISTINGS_CACHE_TTL` (15 min default) or `MLS_LOOKUP_CACHE_TTL` (30 days default)
2. Set the environment variable on **all** Coolify apps (web + worker + scheduler share config)
3. No restart needed for already-cached rows — TTL is checked at read time against `last_updated`
4. Existing rows do not retroactively change TTL — they expire based on the new value on next read

```bash
# Example: reduce listings cache to 5 minutes
export LISTINGS_CACHE_TTL=300
```

1. Make changes
2. Validate: `GOFLAGS=-mod=mod go build ./cmd/...`
3. If build fails, fix and repeat step 2
4. Deploy and verify: identical request within TTL returns `X-IDX-Cache: HIT`

## Debugging Cache HIT/MISS

**Cache always MISS?** Check these in order:

1. **Is the partition non-empty?** `finishProxy` skips cache when `partition == ""`
2. **Is TTL shorter than request interval?** Default `ListingsCacheTTL` is 900s (15 min). If you wait longer, the row is stale.
3. **Is the fingerprint changing between requests?** `FingerprintRequest` includes sorted query params. A single param difference = different fingerprint.
4. **Is the purge job running?** Check scheduler logs for `mls proxy cache purge`. If `MLS_PROXY_CACHE_RETENTION_DAYS` is very low, rows are purged before reuse.

```sql
-- Inspect cache contents for a domain
SELECT partition_key, fingerprint, length(compressed_data) AS bytes,
       last_updated, NOW() - last_updated AS age
FROM mls_search_cache
WHERE partition_key LIKE 'acme:%'
ORDER BY last_updated DESC
LIMIT 20;
```

```sql
-- Check if TTL is the issue (compare age to configured TTL)
SELECT partition_key,
       EXTRACT(EPOCH FROM (NOW() - last_updated)) AS age_seconds
FROM mls_search_cache
WHERE partition_key LIKE '%:lookup%'
LIMIT 5;
-- If age_seconds > 2592000 (30 days in seconds), lookup rows are stale
```

## Manual Cache Purge

For urgent cache invalidation (e.g., upstream data corrected):

```sql
-- Purge all cache for a specific domain
DELETE FROM mls_search_cache WHERE partition_key LIKE 'acme:%';

-- Purge only listings cache (not lookup)
DELETE FROM mls_search_cache WHERE partition_key LIKE 'acme:%:web:%';

-- Purge lookup cache to force refresh
DELETE FROM mls_search_cache WHERE partition_key LIKE '%:lookup';
```

For full purge across all domains:

```sql
TRUNCATE mls_search_cache;
```

The scheduler's purge job runs every 15 minutes and handles normal expiry. Manual `DELETE` is only for urgent invalidation.

## Adding a New Partition Type

If the existing four partition functions don't fit (Web, Reso, Search, Lookup):

1. Add a new function in `internal/service/cache/canonical.go`
2. If it needs a different TTL, add a suffix check in `TTLForPartition` and a new config field in `internal/config/config.go`
3. Use the new partition in your handler

```go
// new code to add — canonical.go
func CustomPartition(domainSlug, feedCode, routeType string) string {
    return fmt.Sprintf("%s:%s:custom:%s", domainSlug, feedCode, routeType)
}

// new code to add — proxy_cache.go TTLForPartition addition
func (p *ProxyCache) TTLForPartition(partition string) time.Duration {
    if stringsHasSuffix(partition, ":lookup") {
        return p.cfg.Bridge.LookupCacheTTL
    }
    if stringsHasSuffix(partition, ":custom") {
        return p.cfg.Bridge.CustomCacheTTL // new config field
    }
    return p.cfg.Bridge.ListingsCacheTTL
}
```

Copy this checklist and track progress:

- [ ] 1. Add partition function in `canonical.go`
- [ ] 2. Add config field in `config.go` with env var
- [ ] 3. Add TTL tier in `TTLForPartition` if needed
- [ ] 4. Use new partition in handler
- [ ] 5. Build: `GOFLAGS=-mod=mod go build ./cmd/...`
- [ ] 6. Test: verify correct TTL with SQL inspection

## Integration with Other Systems

- **Scheduler** (See the **queue-postgresql** skill): Enqueues `mls.proxy_cache_purge` every 15 min on the `default` queue. Only the leader scheduler enqueues (advisory lock).
- **Worker** (See the **queue-postgresql** skill): `handleProxyCachePurge` calls `RefreshJob.Run()` which calls `ProxyCache.PurgeExpired()`.
- **Bridge handler** (See the **fiber** skill): `finishProxy` is the single integration point — all proxy routes flow through it.
- **Hybrid search** (See the **go** skill): `SearchPartition` caches the live-upstream leg of split searches.
- **Image rewriting**: Must happen before `Put()` — cached responses include rewritten image URLs.