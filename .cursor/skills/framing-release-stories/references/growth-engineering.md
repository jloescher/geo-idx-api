# Growth Engineering

## When to use
Use when designing viral loops, referral programs, or expansion revenue features that leverage the GHL widget ecosystem, API token sharing, or subscription tier upgrades.

## Project-relevant patterns

**Widget viral coefficient**
Each `ghl_registered_urls` row generates a widget API key (`qh_...`). Track embedding sites via `Referer` headers in `bridge_proxy_audit_logs` to identify high-traffic domains that might upgrade from Pro (3 domains) to Ultra (unlimited).

**Lead sync as retention**
The `SyncLeadToGhlJob` creates contacts and opportunities in GHL. When `creates_opportunity=true` in `ghl_lead_mappings`, the lead appears in the agency's pipeline—this tight integration increases switching costs. Monitor `ghl_installed_locations.lead_count` for usage-based expansion triggers.

**API overage metered billing**
Ultra and Mega plans include 2M API calls/month with metered overage. The `STRIPE_PRICE_IDX_API_OVERAGE_METERED` price ID enables usage-based growth—ensure `BridgeProxyController` increments counters atomically before Stripe billing cycle alignment.

## Pitfalls
Avoid creating incentives that violate MLS compliance. Do not offer referral credits for "sharing your IDX feed"—this could encourage unauthorized domain sharing. Instead, incentivize GHL agency referrals (`ghl_oauth_tokens.user_type=Company`) where the agency controls location access per their Stellar MLS agreement.