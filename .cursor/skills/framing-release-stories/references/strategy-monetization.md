# Strategy & Monetization

## When to use
Use when evaluating pricing changes, tier restructuring, or new revenue streams that affect the `SubscriptionCatalog`, Stripe price IDs, or the relationship between teaser gating and paid conversion.

## Project-relevant patterns

**Teaser as price anchor**
The default teaser cap (3 listings) is defined in `BridgeTeaser`. When testing higher conversion tiers, adjust via `config('bridge.teaser_limit')` but maintain compliance—Stellar MLS requires attribution and prohibits full listing display without authenticated access. The teaser is the primary incentive to subscribe.

**Plan capability mapping**
`SubscriptionCatalog` defines:
- Pro ($39/mo): 3 domains, teaser gating, basic GHL app
- Smart ($79/mo): +5 domains, OTP phone/email, full GHL app
- Ultra ($179/mo): Unlimited domains, 2M API calls, developer keys
- Mega ($449/mo): Unlimited everything, custom branding, SLA

Any release adding features must update both `config/billing.php` and the corresponding Stripe product metadata to keep the pricing page (`SalesLandingPage`) accurate.

**Token ability gating**
Sanctum abilities (`idx:access` vs `idx:full`) map to subscription tiers. `idx:access` sees teasers; `idx:full` sees full payloads. When introducing new API surfaces (e.g., `/api/v1/gis`), decide whether to gate behind existing abilities or create new ones—new abilities require migration of existing `personal_access_tokens` rows.

## Pitfalls
Do not hardcode plan logic in controllers. The `SubscriptionCatalog` class is the single source of truth. A release that adds "GIS full access" must update the catalog's `hasGisFullAccess()` method and ensure the middleware check uses this method rather than checking `$user->subscription('default')->price` directly—Stripe price IDs change during promotions.