# Measurement & Testing

## When to use
Adding analytics to sales pages, tracking widget lead conversion, or monitoring GHL OAuth funnel completion.

## Patterns

**Livewire dispatch for analytics**
Use `$this->dispatch('analytics-event', ['event' => 'pricing_toggle', 'interval' => 'yearly'])` in `SalesLandingPage` when users switch billing intervals. Listen in `resources/js/app.js` with Alpine.js and forward to Plausible/Google Analytics. Keeps analytics logic out of PHP business logic.

**Bridge proxy audit logging**
Every `/api/v1/*` call writes to `bridge_proxy_audit_logs` with `domain_slug`, `token_name`, `listing_count`. Query this table for usage-based conversion metrics: "Domains with >50 listing views in 7 days but no subscription" are upgrade candidates. Join with `ghl_installed_locations` for GHL-specific cohorts.

**GHL webhook event tracking**
The `ghl_webhook_events` table persists all marketplace events with `webhook_id`. Use this to compute install-to-lead time: compare `INSTALL` webhook `created_at` to first `quantyra_leads` row for that `ghl_location_id`. The `GhlAuditService` logs latency metrics—query `latency_ms` for API health dashboards.

## Warning
The `GHL_WEBHOOK_REQUIRE_SIGNATURE` env var controls signature verification. In local testing with tools like ngrok or Stripe CLI, you may need to disable this. Never disable in production—the webhook secret ensures events actually came from HighLevel, preventing fake install events from polluting conversion metrics.