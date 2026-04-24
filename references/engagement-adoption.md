# Engagement & Adoption

When to use: Increasing feature usage after initial onboarding, driving upgrades from trial to paid, and reducing subscriber churn.

## Project-Relevant Patterns

### Tiered Teaser Gates
The Bridge proxy implements engagement-driven feature exposure via `idx:access` vs `idx:full` Sanctum abilities. Free/trial domains see 3-listing teasers (`BridgeTeaser::apply()`) while prompting upgrade. This creates natural adoption pressure toward paid tiers without hard paywalls.

### Widget Embed Verification
Track actual usage via `/widget/loader.js` hits with unique API keys. Store `last_embed_load_at` on `ghl_widget_configs` and surface "X days since last widget load" alerts in the subscriber dashboard to prompt re-engagement.

### Lead Volume Milestone Emails
The `SyncLeadToGhlJob` accumulates `leads_this_month` on `GhlInstalledLocation`. Cross-reference with subscription tier limits (Pro=50, Smart=200, Ultra=1000, Mega=unlimited). Trigger in-dashboard banners when approaching limits, not after — this drives upgrade conversion better than post-limit blocking.

## Pitfall
Do not rely solely on subscription status for engagement cues. A subscriber may be "active" in Cashier terms but disengaged if widget embed was removed from their site. Always verify embed loads are occurring within the trailing 30 days before marking accounts as healthy.
