# Proxy Web — Workflows Reference

## Contents
- Adding a New RESO Proxy Route
- Adding a New Web Proxy Route
- Modifying Cache TTL Behavior
- Adding a New MLS Provider
- Debugging Cache Misses
- Deploying Proxy Changes

---

## Adding a New RESO Proxy Route

Copy this checklist and track progress:
- [ ] Add handler method in `internal/handler/bridge/handler.go` calling `proxyReso` or `proxyResoWithKey`
- [ ] Register route in `internal/api/routes.go` with `DomainToken` + `MLSAccess` middleware
- [ ] Add audit type string to audit logging (matches `auditType` param)
- [ ] Test: `GET /api/v1/{entity}` returns `X-IDX-Cache: MISS` on first call, `HIT` on second
- [ ] Verify image URLs in response are rewritten (check `IDX_IMAGES_PUBLIC_URL`)

```go
// new code to add in handler.go
func (h *Handler) MyEntities(c *fiber.Ctx) error {
    return h.proxyReso(c, "myentities.collection", "MyEntity")
}
func (h *Handler) MyEntity(c *fiber.Ctx) error {
    return h.proxyResoWithKey(c, "myentities.detail",
        "MyEntity('"+c.Params("myEntityKey")+"')", c.Params("myEntityKey"))
}
```

```go
// new code to add in routes.go — middleware chain is critical
mlsGroup.Get("/myentities", bridgeHandler.MyEntities)
mlsGroup.Get("/myentities/:myEntityKey", bridgeHandler.MyEntity)
```

**Feedback loop:**
1. Make changes
2. Validate: `go build ./cmd/api && go test ./internal/handler/bridge/...`
3. If validation fails, fix and repeat
4. Confirm cache headers (`X-IDX-Cache`) appear in responses

---

## Adding a New Web Proxy Route

Web routes use Bridge's non-RESO path structure (agents, offices, etc.):

```go
// new code to add — uses WebPartition for cache keying
func (h *Handler) Teams(c *fiber.Ctx) error {
    return h.proxyWeb(c, "teams.collection", "teams")
}
```

Web routes differ from RESO routes in URL construction — they use `WebURL(path, dataset)` instead of `ResoURL(entity, dataset)`.

---

## Modifying Cache TTL Behavior

Cache TTL is per-partition, controlled by two environment variables:

| Variable | Default | Applies to |
|---|---|---|
| `LISTINGS_CACHE_TTL` | 900s (15m) | All partitions except `:lookup` |
| `MLS_LOOKUP_CACHE_TTL` | 720h (30d) | Partitions ending in `:lookup` |

The `TTLForPartition` method in `internal/service/cache/proxy_cache.go` decides:

```go
func (p *ProxyCache) TTLForPartition(partition string) time.Duration {
    if stringsHasSuffix(partition, ":lookup") {
        return p.cfg.Bridge.LookupCacheTTL
    }
    return p.cfg.Bridge.ListingsCacheTTL
}
```

**To add a new TTL tier:** Add a new partition suffix and matching condition. Do not override TTL per-request — that breaks cache consistency.

Stale rows are purged by the `mls.proxy_cache_purge` scheduled job (every 15 min). See the **queue-postgresql** skill for job details.

---

## Adding a New MLS Provider

This is the most complex proxy workflow. The provider must implement `ProxyClient`:

1. Create `internal/mlspoxy/{provider}/client.go` implementing `Proxy()` and `ProxyUpstream()`
2. Add provider config struct to `internal/config/config.go`
3. Add provider detection in `Factory.ForRequest()` based on `FeedDefinition.Provider`
4. Add URL construction helpers (like `bridge.Client.WebURL`, `bridge.Client.ResoURL`)
5. Update `internal/service/mls/resolver.go` to resolve `?dataset=` to the new provider
6. Add feed definition to `internal/domain/` with dataset mapping
7. Test with `?dataset={new_dataset}` against a real endpoint

**DO:** Follow the Bridge/Spark pattern exactly — wrapper struct + delegation.
**DON'T:** Add provider-specific logic in `finishProxy`. Keep it provider-agnostic.

---

## Debugging Cache Misses

1. Check `X-IDX-Cache` header — `HIT` means cache served, `MISS` means upstream called
2. Verify partition key matches between requests (domain slug + feed code must be identical)
3. Check TTL — `mls_search_cache.last_updated` must be within TTL window
4. Verify fingerprint — sorted query params + body must match exactly
5. Check `MLS_PROXY_CACHE_RETENTION_DAYS` — purged rows are gone permanently

```sql
-- Inspect cache entries for a domain
SELECT partition_key, fingerprint, length(compressed_data) AS bytes,
       last_updated, NOW() - last_updated AS age
FROM mls_search_cache
WHERE partition_key LIKE '{domainSlug}:{feedCode}%'
ORDER BY last_updated DESC LIMIT 20;
```

---

## Deploying Proxy Changes

Proxy changes touch the `api` service only (not worker/scheduler). Deployment order:

1. Build: `docker build -f Dockerfile --target api -t idx-api:local .`
2. Test: `go test ./internal/handler/bridge/... ./internal/service/cache/... ./internal/mlspoxy/...`
3. Deploy `api` container first — proxy cache is read-through, no migration needed
4. Verify `/healthz` and a known proxy route (e.g., `GET /api/v1/properties`)
5. Check `X-IDX-Cache` headers return expected values

**Multi-DC note:** Both NYC and ATL API containers share the same `mls_search_cache` table. Deploy both regions before verifying cache behavior. See the **deploy-coolify** and **hosting-tailscale** skills for multi-DC patterns.

**Cache purge after deploy:** If response shape changed (e.g., new fields), purge affected partitions:

```sql
DELETE FROM mls_search_cache WHERE partition_key LIKE '%:reso:{Entity}';
```

The scheduler's `mls.proxy_cache_purge` job will clean expired entries on its next run. See the **queue-postgresql** skill.

---

For authentication and access control, see the **auth-api-token** and **auth-domain** skills. For Go language and Fiber framework patterns, see the **go** and **fiber** skills.