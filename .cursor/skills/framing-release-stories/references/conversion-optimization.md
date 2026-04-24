# Conversion Optimization

## When to use
Use when designing or refining user flows that transition visitors into authenticated leads or paying subscribers—especially for the Livewire marketing pages, GHL widget embeds, and Stripe checkout flows.

## Project-relevant patterns

**Teaser-to-full gating sequence**
Domain-authenticated users see capped listing previews (3 items). The conversion moment occurs when a lead form widget is submitted or a user clicks "Unlock full search" to hit the Stripe checkout. Ensure the transition from `/widget/search/{apiKey}` teaser to `billing.checkout` route preserves any search context in session.

**Progressive disclosure in widget loader**
The `/widget/loader.js` endpoint can vary its initialization payload based on the `data-widget` type. For `lead-form` widgets, defer MLS listing data entirely and focus on address capture; for `showcase` widgets, show highest-quality teasers first to drive engagement before gating.

**Checkout friction reduction**
The `SalesLandingPage` Livewire component supports annual/monthly toggle with 20% discount framing. When users arrive from GHL widgets with `?plan=ultra` or similar, pre-select that tier and collapse the toggle to reduce decision fatigue.

## Pitfalls
Do not rely solely on `idx:full` token grants for conversion tracking. Many leads convert via GHL widget submissions that create `quantyra_leads` rows without ever hitting the IDX dashboard—ensure `SyncLeadToGhlJob` also writes conversion attribution to `ghl_sync_logs.request_payload` for accurate funnel analysis.