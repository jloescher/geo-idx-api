# Strategy & Monetization

## When to use
Aligning lead gating with subscription tiers, designing upgrade prompts, or balancing teaser value with conversion pressure. Use when modifying the `SubscriptionCatalog` or tier feature gates.

## Patterns

### Tiered Teaser Limits
Map `gate_after_views` to billing tiers: Pro ($39) = 3 views, Smart ($79) = 5 views, Ultra ($179) = 10 views, Mega ($449) = unlimited. Configure `SubscriptionCatalog` to expose these limits to the widget middleware via `ghl_installed_locations.subscription_status`.

### Upgrade Prompt Context
When gating triggers, show contextual upgrade messaging: "Upgrade to Smart for instant phone verification and priority lead routing". Link to `IDX_PLATFORM_URL` checkout with pre-filled location context.

### Metered Overage for High-Volume Locations
Use Stripe metered billing for locations exceeding their tier's lead volume. The `SyncSubscriptionStatusJob` updates `ghl_installed_locations` flags that the widget middleware checks before accepting new leads.

## Warning
Teaser gating is a revenue lever but must comply with Stellar MLS PDA terms. Do not gate public parcel data from the GIS proxy—only MLS listing details from Bridge. The `idx:full` ability bypass should only unlock for paying subscribers with valid MLS agreements.