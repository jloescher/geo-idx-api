# Measurement & Testing

When to use this reference: When implementing A/B tests, tracking conversion funnels, or validating that decision cues actually improve subscription metrics across the billing lifecycle.

## Patterns

**Audit-Log Attribution**
Log all conversion events to `bridge_proxy_audit_logs` and `ghl_audit_logs` with structured metadata: `teaser_shown`, `plan_selected`, `trial_started`, `upgrade_completed`. Use `GhlAuditService` for dual-channel logging (database + file) to ensure attribution survives even if one storage layer degrades.

**Tier-Level Funnel Analysis**
Segment metrics by subscription tier in `SubscriptionCatalog` reporting. Track Pro→Smart and Smart→Ultra upgrade velocities separately—the behavioral cues that drive initial signup differ from those driving expansion revenue. The 3-listing teaser should show distinct conversion curves for domain-authenticated vs token-authenticated traffic.

**Webhook-Driven Experimentation**
Use GHL webhook events (`INSTALL`, `APPUNINSTALL`) to trigger experiment cohort assignment. When `WebhookDispatcher` routes install events, append experiment variant IDs to `ghl_installed_locations` metadata for downstream analysis without modifying core subscription flows.

## Warning

Do not use `teaser=true` responses to track individual user behavior across sessions—the teaser flag in `BridgeTeaser` is session-scoped and lacks persistent identity. Attribution requires the `ghl_location_id` or `domain_slug` foreign keys; never attempt to fingerprint from MLS listing data itself.