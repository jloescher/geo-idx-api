# Measurement Testing

## When to use
Track tool effectiveness, subscription conversion rates, and system health through audit logs, subscription status monitoring, and A/B variant testing.

## Patterns

**Dual-Channel Audit Logging**
`GhlAuditService` writes to both `ghl_audit_logs` (PostgreSQL) and optional file channel (`GHL_AUDIT_LOG_PATH`). Track widget impressions via `request_type='widget_load'`, lead submissions via `request_type='lead_ingest'`, and conversion events via subscription status changes in `ghl_installed_locations`.

**Subscription Status Funnel**
The `subscription_status` column cycles through `none → trial → active → past_due → cancelled`. Stripe webhooks update this via `SubscriptionSyncService`, which propagates tags back to GHL CRM for campaign segmentation. Test checkout flows using `./scripts/stripe-dev.sh trigger-checkout-completed`.

**Widget Variant Testing**
Create multiple `ghl_widget_configs` rows for the same location with different `gate_after_views` values. The widget API key determines which configuration serves, allowing parallel testing of gating thresholds without code changes.

## Warning
The `RefreshDomainListingsCacheJob` runs every 15 minutes per active domain—monitor queue depth during load tests. Stalled queue workers prevent cache refresh, causing stale data that degrades conversion rates silently.