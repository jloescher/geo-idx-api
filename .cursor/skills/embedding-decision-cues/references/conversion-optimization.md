# Conversion Optimization

When to use this reference: When designing subscription flows, pricing page layouts, or teaser-gated content experiences to increase trial starts and plan upgrades across the Pro/Smart/Ultra/Mega tier ladder.

## Patterns

**Anchored Choice Architecture**
Present the Smart plan ($79/mo) as the visually prominent middle option between Pro ($39) and Ultra ($179). Use the decoy effect—position Smart as "Most Popular" with distinct elevation styling while Mega remains collapsed behind an "Enterprise" expander. In `SubscriptionCatalog`, set `recommended=true` on the Smart tier to trigger badge rendering and border emphasis.

**Teaser-to-Trial Progression**
Leverage the existing 3-listing cap in `BridgeTeaser` to demonstrate value before paywall. Display specific upgrade CTAs inline: "Showing 3 of 247 listings in this area—unlock full search with Pro." Log teaser exposure events to `bridge_proxy_audit_logs` for retargeting sequences.

**Loss-Averse Trial Framing**
Lead with "Your 14-day free trial starts now" rather than payment collection. In trial expiration UI, frame as "You will lose access to 247 saved listings" rather than "Your trial ends." Default the billing toggle to annual ($374/yr vs $468 monthly) with "Save 20%" prominently displayed.

## Warning

Avoid dark patterns like forced continuity without clear notice or fake scarcity on unlimited resources. The `BridgeProxyAuditLogger` records all teaser and conversion events—any manipulation that violates Stellar MLS transparency requirements will appear in compliance audits.