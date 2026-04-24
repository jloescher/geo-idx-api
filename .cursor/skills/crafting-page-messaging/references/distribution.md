# Distribution

## When to use

Planning how new pages, features, or pricing changes reach the target audience—whether through organic search, GHL Marketplace discovery, email, or widget embeds.

## Project-relevant patterns

**GHL Marketplace as primary channel** — The install URL (`/leadconnector/install`) is the top-of-funnel entry point. Optimize this page for GHL users who found the app in the marketplace. Lead with "Works inside your existing GHL account" to reduce perceived switching cost.

**Widget embeds as viral loop** — Each installed location generates a widget API key (`qh_...`). The widget itself is a distribution mechanism—agents embedding it on their sites become advocates. Include subtle "Powered by GeoIDX" attribution in widget footers for awareness.

**Domain-specific SEO** — The public IDX platform (`idx.quantyralabs.cc`) supports custom domains for higher tiers. When customers point their own domains, their SEO benefits compound. Document this clearly in Mega tier sales materials.

## Pitfalls

Do not rely on IDX API endpoints (`/api/v1/listings`) being discoverable by search engines. They require authentication and return JSON. Ensure the public IDX platform (`idx.*`) has proper meta tags, sitemap, and canonical URLs for listing pages if SEO is a goal.