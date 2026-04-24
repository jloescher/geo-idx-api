# Roadmap & Experiments

When to use: Planning A/B tests for onboarding copy, pricing presentation, and feature gating strategies.

## Project-Relevant Patterns

### Teaser Limit Experimentation
The `BridgeTeaser::limit` is configurable via `TEASER_LISTING_COUNT` env (default 3). Run A/B tests by varying this per domain via `domains.teaser_override` column. Measure impact on upgrade conversion by correlating with `SubscriptionCheckoutController` checkout session creation events.

### Pricing Interval Framing
The `SalesLandingPage` Livewire component supports toggling monthly vs annual billing. The experiment-ready pattern stores the user's selected interval preference in session and passes it to Stripe Checkout via `subscription_data[trial_period_days]` metadata. Track which interval drives higher LTV, not just initial conversion.

### OAuth Landing Variants
Test install-to-authorize conversion by varying the `/leadconnector/install` blade template: variant A emphasizes "Free IDX search for your leads"; variant B emphasizes "MLS-compliant listing display". Store variant assignment in session and correlate with `OAUTH_COMPLETED` audit events.

## Pitfall
Avoid running experiments that change the OAuth redirect URI or widget loader URL structure. These are registered in the GHL Marketplace app configuration and cannot vary per-user. Keep experiments within the idx-api domain where you control routing and rendering.
