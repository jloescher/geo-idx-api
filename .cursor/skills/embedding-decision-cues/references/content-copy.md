# Content Copy

When to use this reference: When writing UI text for pricing pages, plan comparison tables, upgrade prompts, and teaser-gated messaging that drives conversion without violating MLS compliance.

## Patterns

**Specificity Over Abstraction**
Replace "Unlock more features" with "Access 2M API calls/month and unlimited domain mappings." In `SalesLandingPage` Livewire component, pass concrete plan limits from `SubscriptionCatalog` into Blade templates. For teaser contexts, use dynamic counts: "Showing 3 of {actual_count} listings" pulled from Bridge cache metadata.

**Value-First Headers**
Lead pricing page sections with outcomes: "Close more buyers with Smart IDX" not "Smart Plan Features." Mirror this in widget loader scripts—`loader.js` should inject persuasive context about what the embedded search enables, not technical specifications.

**Social Proof Anchors**
Reference the GHL installed locations count in copy when available: "Join {count} real estate professionals using Quantyra IDX." In `SubscriptionCheckoutController`, pass `installation_count` to the checkout view for dynamic rendering. Avoid fabricated numbers—source from `ghl_installed_locations` table only.

## Warning

Never imply unlimited MLS data access in teaser contexts. Phrases like "Browse all listings" violate Stellar MLS Exhibit A when the user sees only 3 items. Always qualify with "Preview" or "Sample" and ensure `BridgeTeaser` application logic matches the copy承诺.