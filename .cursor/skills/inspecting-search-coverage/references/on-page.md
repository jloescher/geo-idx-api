# On-Page Search Coverage

When to use: Reviewing title tags, meta descriptions, heading hierarchy, and content optimization for IDX marketing pages and GHL widget embeds.

## Patterns

**Livewire Marketing Component SEO**
The `SalesLandingPage` Livewire component in `app/Livewire/Marketing/` renders the main IDX sales page. Since Livewire dynamically updates content, ensure that initial page loads contain complete title tags and meta descriptions server-side, and that JavaScript-enhanced content degrades gracefully for crawlers.

**Widget Embed Content Isolation**
Widget endpoints (`/widget/search/{apiKey}`, `/widget/showcase/{apiKey}`) in `routes/ghl-widget.php` return HTML shells for iframe embeds. These should use appropriate `X-Robots-Tag: noindex, nofollow` headers to prevent widget content from competing with primary listing pages in search results.

**Dashboard vs Marketing Page Differentiation**
Authenticated dashboard views (`/dashboard`, `/billing/checkout`) should universally carry `noindex` directives to prevent search engines from indexing subscriber-specific content. Marketing pages (`/`, `/leadconnector/install`) need optimized `<title>` and `<meta name="description">` tags targeting real estate agent onboarding keywords.

## Warning

The GHL OAuth callback and widget loader endpoints may inadvertently expose location-specific content without proper robots controls. Verify that all `/leadconnector/*` and `/widget/*` routes include appropriate indexing directives—location-specific data in these flows can create thin content issues if indexed.