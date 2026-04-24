# Distribution

## When to use
Use when designing content delivery systems, API response flows, or multi-channel data synchronization in the IDX platform.

## Patterns

### Tiered Cache Distribution
The GIS proxy demonstrates a 3-tier distribution strategy:

```php
// GisProxyService - 3-level cache cascade
// 1. Laravel Cache (edge) - 15min TTL
if ($cached = Cache::get($cacheKey)) {
    return $cached;
}

// 2. PostgreSQL gis_cache (origin) - 30-90 days
if ($row = GisCache::validForSource($source)->first()) {
    return $this->hydrateFromOrigin($row);
}

// 3. ArcGIS fetch + async filesystem backup
$geoJson = $this->fetchFromArcGis($bbox);
PersistGisGeoJsonBackupJob::dispatch($geoJson, $cacheKey);
```

### Queue Isolation by Channel
Separate critical paths from background work:

```php
// routes/console.php - Channel-specific queues
Schedule::command('ghl:refresh-tokens')
    ->hourly()
    ->withoutOverlapping()
    ->onOneServer();

// Jobs dispatched to dedicated queues
ProcessGhlWebhookJob::dispatch($event)->onQueue('webhooks');
SyncLeadToGhlJob::dispatch($lead)->onQueue('sync');
RefreshGisSourceMetadataJob::dispatch()->onQueue(config('gis.queue', 'default'));
```

### Webhook Fanout with Idempotency
GHL webhooks use inbox pattern with deduplication:

```php
// WebhookDispatcher - idempotent processing
$event = GhlWebhookEvent::firstOrCreate(
    ['webhook_id' => $payload['webhookId']],
    ['event_type' => $type, 'payload' => $payload, 'processing_status' => 'received']
);

if ($event->wasRecentlyCreated) {
    ProcessGhlWebhookJob::dispatch($event);
}
```

## Pitfall
Never distribute MLS listing data without Origin header validation. Widget middleware enforces registered URLs—bypassing this violates Stellar MLS compliance.