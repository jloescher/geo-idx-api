# Distribution

When to use this reference: When planning how conversion surfaces reach users—through GHL widget embeds, dashboard notifications, email sequences, or the marketing site—ensuring consistent decision cues across all channels.

## Patterns

**Widget Channel Expansion**
Use `/widget/loader.js` embeds on partner sites to extend reach beyond `idx.quantyralabs.cc`. Ensure `WidgetConfig` passes plan upgrade URLs consistent with the main platform pricing. The 3-phase middleware (key validate → origin validate → CORS) gates access while allowing distribution across unlimited external domains.

**GHL Location Targeting**
Sync subscription status to `ghl_installed_locations.subscription_status` for in-CRM upsell campaigns. When `SubscriptionSyncService` pushes status updates, trigger GHL workflows targeting "trial" locations with time-sensitive upgrade offers before expiration.

**API Token as Distribution Control**
Issue `idx:full` Sanctum tokens sparingly to high-value partners. Each token represents full access distribution—track usage in `bridge_proxy_audit_logs` and cap via `GHL_SUBSCRIPTION_TAG_*` enforcement. Teaser-level distribution (domain auth) requires no token, lowering friction for widget adoption.

## Warning

Widget distribution across unregistered domains bypasses `domains` table validation if the API key is compromised. Rotate `widget_api_key` values in `ghl_registered_urls` regularly and enforce Origin header validation—distribution scale increases attack surface for unauthorized MLS data access.