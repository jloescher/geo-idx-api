---
name: strengthening-upgrade-moments
description: Improves upgrade prompts and paywall messaging across the GeoIDX platform, including teaser gating, subscription tier positioning, and conversion-optimized copy in Livewire components, Blade templates, and API responses.
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
model: sonnet
---

# Strengthening Upgrade Moments Skill

This skill helps improve upgrade prompts and paywall messaging across the Quantyra GeoIDX platform, from the 3-item teaser gate in the Bridge proxy to the subscription checkout flow and GHL widget surfaces.

## Quick Start

1. Identify the upgrade surface: Bridge teaser (3 items), GIS feature caps, GHL widget gates, or the sales landing page
2. Read the current implementation in `app/Services/Bridge/BridgeTeaser.php`, `app/Livewire/Marketing/SalesLandingPage.php`, or `app/Billing/SubscriptionCatalog.php`
3. Check `config/billing.php` for tier definitions (Pro $39, Smart $79, Ultra $179, Mega $449)
4. Review `resources/views/` for Blade templates rendering upgrade CTAs
5. Ensure paywall copy aligns with the `SubscriptionCatalog` feature matrix

## Key Concepts

**Teaser Gating**: The Bridge proxy caps non-`idx:full` responses at 3 listings. The `BridgeTeaser` service applies this after cache decompression. Upgrade messaging should clarify what "full access" unlocks (unlimited listings, GIS layers, API calls).

**Subscription Tiers**: Four tiers defined in `SubscriptionCatalog` — Pro (3 domains, teaser), Smart (5 domains, full GHL), Ultra (unlimited domains, 2M API calls), Mega (unlimited everything, SLA). Each tier needs distinct value proposition copy.

**Upgrade Contexts**:
- **API responses**: `meta.teaser=true` flag in GIS/Bridge responses — clients render inline upsells
- **GHL widgets**: `gate_after_views` and `require_otp` in `ghl_widget_configs` — embeddable surfaces need subtle upgrade nudges
- **Dashboard**: `DashboardController` shows subscription status, usage counters, and upgrade CTAs
- **Sales page**: `SalesLandingPage` Livewire component with billing interval toggle (monthly/annual 20% off)

**MLS Compliance Constraints**: All upgrade messaging must preserve Stellar MLS Exhibit A requirements — no full MLS display without subscription, teaser counts hard-capped at 3 for domain-only auth.

## Common Patterns

**Teaser-to-Upgrade Flow**: When `BridgeTeaser::apply()` truncates the list, the response should include upgrade signaling. Check `BridgeProxyController` for how `X-Domain-Slug` vs Bearer token paths diverge.

**Pricing Page Patterns**: The sales landing uses Livewire for real-time billing toggle. Plans are rendered from `SubscriptionCatalog::plans()` with Stripe price IDs. Annual discount is 20% — ensure copy reflects savings.

**GHL Location Scoping**: Agency tokens (Company) vs Location tokens affect upgrade messaging. `ghl_installed_locations.subscription_status` drives gated feature access — use `SubscriptionSyncService` to push Stripe status to GHL.

**Widget API Key Gates**: Widgets validate origins against `ghl_registered_urls`. Upgrade prompts in widget surfaces must respect CORS and use the `qh_*` API key for tracking attribution.

**Metered Overage Messaging**: Ultra/Mega plans include 2M API calls; overage is metered. API responses nearing limits should include soft warnings before hard gates.

**Cashier Webhook Handling**: `stripe/webhook` updates subscription status. Ensure upgrade confirmation emails and in-app messaging align with `SubscriptionCheckoutController` session flash patterns.