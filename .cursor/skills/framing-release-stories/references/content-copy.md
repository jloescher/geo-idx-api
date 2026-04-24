# Content Copy

## When to use
Use when drafting user-facing messaging, error responses, email templates, or in-app guidance that must align with Stellar MLS compliance requirements and the Quantyra brand voice across Bridge proxy, GHL widgets, and billing surfaces.

## Project-relevant patterns

**MLS attribution line**
Every listing display must include the Stellar MLS attribution footer. In widget HTML shells (`/widget/search/{apiKey}`), inject via Blade partial that reads the `mls_agreement_acknowledged` flag from `ghl_registered_urls` and renders: "Data provided by Stellar MLS. Information deemed reliable but not guaranteed."

**Error message taxonomy**
The `DomainOrTokenAuth` middleware returns 401/403 responses. Standardize copy:
- 401 without domain/token: "Access denied. Please register your domain or contact your administrator for API access."
- 403 inactive domain: "Domain registration inactive. Contact support to reactivate your IDX subscription."
- 403 teaser limit: "Preview limit reached. Unlock full search to view more listings."

**Subscription tier naming**
Align marketing copy with `SubscriptionCatalog` definitions:
- Pro: "Essential IDX for single agents"
- Smart: "IDX with lead capture & GHL sync"
- Ultra: "Unlimited domains + developer API keys"
- Mega: "White-glove setup with SLA"

## Pitfalls
Never expose Bridge Data Output error messages directly to end users. The `BridgeProxyController` catches upstream 4xx/5xx and should return generic copy: "Listing data temporarily unavailable" while logging the actual Bridge response to `bridge_proxy_audit_logs` for ops review.