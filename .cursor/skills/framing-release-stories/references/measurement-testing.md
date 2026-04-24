# Measurement & Testing

## When to use
Use when defining success metrics for releases, setting up monitoring dashboards, or designing A/B tests across the Bridge proxy, GHL integration, and billing flows.

## Project-relevant patterns

**Proxy audit log metrics**
Every release should specify which `bridge_proxy_audit_logs` columns to monitor:
- `request_type='listings'` with `listing_count` for volume
- `domain_slug` distribution to verify tenant isolation
- `token_name` ratio to track Sanctum vs domain-auth usage

**GHL sync health indicators**
The `ghl_sync_logs` table provides end-to-end visibility:
- `sync_status='pending'` > 5 minutes indicates queue backup
- `sync_status='failed'` with `response_payload` containing GHL error codes triggers alerts
- Correlate `ghl_audit_logs.latency_ms` with `ghl_sync_logs.created_at` to find API degradation

**Stripe webhook verification**
Cashier webhooks must arrive at `/stripe/webhook`. Monitor:
- Signature verification failures (wrong `STRIPE_WEBHOOK_SECRET`)
- Missing `checkout.session.completed` events (CLI forwarding stopped)
- Subscription status mismatches between Stripe and `ghl_installed_locations.subscription_status`

## Pitfalls
The GIS proxy has degraded mode (`meta.degraded=true`) that serves empty features with OSM fallback tiles. Do not count this as success in availability metrics—track `degraded` responses separately and alert when `source_used` != `pinellas_enterprise_parcels` for consecutive requests in Pinellas County bounding boxes.