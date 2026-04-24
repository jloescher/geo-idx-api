# Conversion Optimization for Partner Loops

## When to use
When designing OAuth flows, widget embeds, or API onboarding where drop-off at any step costs leads. Focus on minimizing friction between install intent and activated usage.

## Project-relevant patterns

**Multi-step OAuth with progress persistence**
Break installation into discrete recoverable steps: `install` landing → `authorize` redirect → `callback` exchange → `register-urls` → `complete`. Store intermediate state in session with `pending_oauth_token_id` so users can resume if interrupted.

**Teaser-gated upsell flow**
Offer limited preview (3 listings, simplified GIS parcels) without registration, then gate full access behind OAuth or subscription. This mirrors the Bridge proxy teaser pattern where `idx:access` abilities show capped results while `idx:full` unlocks complete data.

**One-click embed distribution**
Generate widget scripts with pre-authenticated API keys (`qh_{hash}`) after URL registration. The loader pattern (`/widget/loader.js`) lets partners paste a single script tag without configuring tokens manually.

## Warning
Don't require URL registration before showing value. The GHL flow lets users authorize first, then register domains—this captures the OAuth token even if they abandon domain setup. Reversing the order loses leads at the top of funnel.