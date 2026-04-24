# Product Analytics

When to use: Tracking user behavior through onboarding funnels, API usage patterns, and subscription conversion events. Use for identifying drop-off points, measuring time-to-value, and correlating widget installs with lead volume.

## Patterns

**Funnel Tracking via Audit Logs**
`ghl_audit_logs` table captures `logged_at`, `api_endpoint`, `latency_ms`, and `is_mls_data_access` flags. Query progression from `/leadconnector/install` → `/oauth/leadconnector/callback` → `/leadconnector/register-urls` to calculate conversion rates. Include `ip_address` and `user_agent` for bot filtering. Join with `ghl_webhook_events` to correlate webhook delivery with API activity.

**Feature Adoption via Database Counters**
`ghl_installed_locations` tracks `mls_request_count` and `lead_count` per location. Dashboard queries these to show "You're in the top 10% of agents" percentile rankings. Cache hit ratios from `listings_cache` indicate Bridge API dependency—sudden drops suggest configuration issues. GIS `gis_cache` includes `cache_hit` metadata in GeoJSON response for client-side instrumentation.

**Subscription Event Correlation**
Stripe Cashier webhook handler updates `subscription_status` on `ghl_installed_locations`. Join with `quantyra_leads` creation timestamps to measure trial-to-paid conversion by lead volume. Tag GHL contacts with `quantyra-trial` or `quantyra-active` via `SubscriptionSyncService` for cohort analysis in GHL reporting.

## Warning

Never log raw OAuth tokens, Sanctum PATs, or Stripe signing secrets to analytics systems. `access_token_hash` (SHA-256) is safe for correlation; plaintext tokens in logs violate PCI-DSS and GHL marketplace security requirements. Redact `api_key` query parameters from URL logs in `bridge_proxy_audit_logs`.