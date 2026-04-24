# Growth Engineering

## When to use

Building viral loops, automating lead workflows, or expanding GHL integration surface area. Reference when designing features that increase widget embed adoption or reduce churn.

## Patterns

**Widget Viral Loop** (`routes/ghl-widget.php`)
- Loader pattern: Single `<script>` tag embeds multiple widget types
- Lead attribution: `quantyra_leads.payload` stores source widget and UTM params
- Domain registration: Additional URLs in `ghl_registered_urls.additional_urls` (JSON array)

**Lead Sync Automation** (`app/Ghl/Sync/`)
- `SyncLeadToGhlJob` processes `quantyra_leads` → GHL contacts
- `GhlLeadMapping` configures pipeline/stage/tag assignment per `lead_type`
- `LeadSyncService` creates contacts via `GhlApiClient` with auto-audit logging

**Token Refresh Automation** (`routes/console.php`)
- Hourly: `ghl:refresh-tokens` prevents OAuth expiry
- Schedule: `php artisan schedule:run` or `schedule:work` in dev
- Queue: Requires running queue worker for background jobs

## Warning

Teaser gating (`BridgeTeaser` service) caps listings at 3 items for domain-authenticated requests. This is a revenue lever—do not cache teaser-applied responses as the underlying full data is still accessible via `idx:full` tokens. Cached bytes store full Bridge body; teaser applies after decompression.