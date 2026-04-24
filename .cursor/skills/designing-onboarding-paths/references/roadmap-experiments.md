# Roadmap & Experiments

When to use: Testing onboarding variations, pricing page layouts, and feature gating strategies before full rollout. Use for A/B testing teaser limits, domain validation strictness, and checkout flow optimizations.

## Patterns

**Teaser Limit Experimentation**
`BridgeTeaser` service caps listings at 3 items for non-`idx:full` tokens. Make limit configurable per-domain via `domains.teaser_limit` column (default 3). Test 5-item teasers for specific `domain_slug` patterns to measure lead conversion lift. Gate with `config('bridge.teaser_experiment_domains')` array to avoid cache pollution.

**Pricing Page Variant Testing**
`SalesLandingPage` Livewire component renders subscription tiers from `SubscriptionCatalog`. Support `?variant=` query parameter to swap plan ordering—test annual-first vs monthly-first presentation. Track checkout initiation rates via `SubscriptionCheckoutController` logging to `stripe_checkout_sessions` table with `referrer_variant` column.

**Progressive Rollout via Feature Flags**
GIS proxy supports `layers` parameter reserved for future multi-layer orchestration. Use `config('gis.experimental_layers')` boolean to enable county-specific layers (Pinellas, Hillsborough) for beta domains only. Validate against `GIS_FLORIDA_MLS_CODES` before exposing new endpoints to prevent production regression.

## Warning

Experiments affecting OAuth flows or token storage must maintain backward compatibility—GHL marketplace apps cannot version their redirect URIs. Never experiment with `GHL_WEBHOOK_REQUIRE_SIGNATURE` or `GHL_REDIRECT_URI` in production; these are contractual requirements with GHL and Stellar MLS.