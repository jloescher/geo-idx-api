# Growth Engineering for Partner Loops

## When to use
When scaling from dozens to thousands of partner locations through caching, queue workers, and automated maintenance that reduces operational toil.

## Project-relevant patterns

**Generation-based cache invalidation**
Store `cache_generation` in `gis_source_states` table. When weekly probes detect ArcGIS layer metadata changes, increment generation—this invalidates both edge cache (Laravel Cache) and origin cache (PostgreSQL `gis_cache`) without manual purges.

**Scheduled token maintenance**
Run `ghl:refresh-tokens` hourly via `routes/console.php` with `withoutOverlapping()`. Proactive refresh before `expires_at` prevents service outages when GHL tokens expire mid-day.

**Domain-scoped listing cache**
Cache `GET /api/v1/listings` per `domain_slug` in PostgreSQL with 15-minute TTL. Skip cache when `?filters=` present to ensure filtered queries always hit upstream for accuracy. This amortizes Bridge API costs across high-traffic domains.

## Warning
Queue worker failures on token refresh jobs can silently accumulate. Monitor `failed_jobs` table for `RefreshGhlTokens` or `SyncLeadToGhlJob` failures—unhandled exceptions there don't bubble up to the OAuth callback and partners will experience broken sync without alerts.