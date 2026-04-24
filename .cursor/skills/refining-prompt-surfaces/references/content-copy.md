# Content Copy

## When to use
Writing UI text for subscription tiers, error messages, widget loaders, or GHL onboarding flows in the IDX platform.

## Patterns

**Subscription tier naming consistency**
The `SubscriptionCatalog` defines four tiers: Pro, Smart, Ultra, Mega. Mirror these names exactly in all UI copy—Stripe price IDs (`STRIPE_PRICE_IDX_PRO_MONTHLY`, etc.) depend on this mapping. Use "Upgrade to [Tier]" not "Get [Tier]" to reinforce progression. Include metered overage warnings: "2M API calls/mo included, then $0.005 per 1K calls."

**MLS compliance disclaimers**
Widget embeds and GHL flows must display Stellar MLS attribution. Use the exact copy: "Listing data provided by Stellar MLS. Information deemed reliable but not guaranteed." Store this in `config/billing.php` or a localization file for consistency across `resources/views/widget/` templates.

**Error message granularity**
Bridge proxy errors should distinguish between auth failures (401/403) and upstream MLS issues. Use "Domain not registered for MLS access" for `DomainOrTokenAuth` failures—don't expose `BRIDGE_API_KEY` issues. For GHL OAuth, "Connection to HighLevel expired" is clearer than "Token invalid."

## Warning
Never include actual Stripe price amounts in Blade templates—always pull from `SubscriptionCatalog` or `config/billing.php`. Prices change between test/live modes and currencies. Hardcoded prices in copy cause billing disputes when the catalog updates.