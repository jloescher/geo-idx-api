# Product Analytics

## When to use

Scoring initiatives that add telemetry, usage dashboards, or data-driven prioritization inputs. Apply when betting on observability features in Pulse/Telescope, custom metrics pipelines, or billing metering hooks for Stripe overage.

## Project-relevant patterns

**Audit log as analytics source**: `bridge_proxy_audit_logs` and `ghl_audit_logs` already capture request counts, latency, and domain-level activity. High-impact bets aggregate these for subscriber-facing usage dashboards without new instrumentation — effort is query optimization on existing PostgreSQL tables.

**Stripe metered billing hooks**: The `SubscriptionCatalog` supports metered overage via `STRIPE_PRICE_IDX_API_OVERAGE_METERED`. Medium-effort analytics bets wire usage counters to Cashier's metered events, but risk double-charging if `bridge_proxy_audit_logs` deduplication is not handled.

**Octane/Pulse metrics**: FrankenPHP concurrency and request timing are visible via Pulse when `PULSE_ENABLED`. Low-effort bets expose these metrics to subscribers for transparency; high-effort bets build custom Pulse cards for domain-specific cache hit ratios.

## Warning

Do not instrument new analytics without checking existing audit tables first. `ghl_sync_logs` already tracks lead sync success/failure; `ghl_webhook_events` captures inbound webhook volume. Duplicate logging increases storage costs and creates consistency risks across the 29 migration tables.