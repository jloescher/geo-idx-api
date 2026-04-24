# Content Copy

## When to use
Craft messaging for the sales landing page, widget embeds, and lead forms that communicates value while respecting MLS display restrictions and driving subscription upgrades.

## Patterns

**Tiered Feature Positioning**
The `SubscriptionCatalog` defines four tiers (Pro $39, Smart $79, Ultra $179, Mega $449) with escalating domain limits and API call allocations. Copy should emphasize the "teaser" vs "full" distinction without promising specific listing counts that violate Stellar MLS Exhibit A.

**Widget Loader Script Messaging**
The `/widget/loader.js` endpoint returns configurable loader text. Use `ghl_widget_configs.widget_theme` and color fields to match agency branding while maintaining Quantyra value proposition in fallback copy when custom text isn't provided.

**Lead Form Field Labels**
`QuantyraLead` captures standard fields (name, email, phone) plus `lead_type` mapped via `ghl_lead_mappings`. Form copy should reference the location's registered domain (`ghl_registered_urls.primary_url`) for context-aware messaging like "Get alerts for [Domain] listings."

## Warning
Avoid hardcoding specific listing quantities in marketing copy—the `BRIDGE_TEASER_LIMIT` (default 3) is configurable per environment and may change based on MLS agreement amendments.