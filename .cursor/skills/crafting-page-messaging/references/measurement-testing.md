# Measurement & Testing

## When to use

Setting up analytics, A/B tests, or success metrics for page changes—understanding what "better" means and how to detect it.

## Project-relevant patterns

**Trial-start as North Star** — For marketing pages, the primary conversion is Stripe Checkout session creation. Track this with the `billing.checkout` route event. Secondary: GHL OAuth start (`leadconnector.oauth.authorize`), which indicates install intent even if billing happens later.

**GHL location activation depth** — Not all installs are equal. A location that registers URLs and embeds a widget is more valuable than one that completes OAuth and goes dormant. Measure "locations with widget loads" separately from "locations with OAuth tokens."

**Teaser-to-upgrade funnel** — Track impressions of teaser-limited listings (3 items), then upgrade click-through rate. If teaser CTR is low, the constraint may be invisible or the upgrade path unclear. If it's high but conversion low, the pricing page may be the problem.

## Pitfalls

Avoid measuring "API call volume" as a marketing success metric. High Bridge proxy traffic can come from a single misconfigured widget or bot. Correlate API volume with distinct domain count or paying location count to measure real adoption.