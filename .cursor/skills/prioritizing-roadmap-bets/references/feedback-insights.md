# Feedback & Insights

## When to use

Scoring initiatives that collect subscriber input, support telemetry, or synthesize qualitative data into prioritization signals. Apply when betting on NPS surveys, support ticket classification, or GHL webhook-driven health signals that inform roadmap tradeoffs.

## Project-relevant patterns

**Webhook health as proxy**: The `ghl_webhook_events` table captures INSTALL/UNINSTALL and CRM events. High-impact bets correlate uninstall spikes with recent deploys or subscription changes — effort is SQL aggregation on `processing_status` and `event_type`, not new data collection.

**Lead type mapping feedback**: `GhlLeadMapping` (seeded by `GhlConfigSeeder`) defines how `quantyra_leads` become GHL contacts. Medium-effort bets add per-location customization of these mappings, but require validation that `SyncLeadToGhlJob` handles malformed payloads without retry loops.

**Widget error telemetry**: The widget middleware chain validates API keys and origins. Low-effort bets log validation failures (e.g., origin mismatch) to `ghl_audit_logs` with `compliance_verified=false`, exposing configuration errors that block lead capture.

## Warning

Do not build feedback mechanisms that bypass the existing audit infrastructure. The `GhlAuditService` dual-channel logging (database + file) is designed for MLS compliance — custom feedback tables may miss the `is_mls_data_access` flagging required for Stellar MLS reporting obligations.