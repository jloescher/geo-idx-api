# On-Page Search Coverage

When to use: Reviewing title tags, meta descriptions, heading hierarchy, and content optimization for IDX marketing pages and GHL widget embeds.

## Patterns

**Platform marketing home**
The static Blade view [`resources/views/marketing/home.blade.php`](resources/views/marketing/home.blade.php) is the primary public entry on the platform host. Keep `<title>` and `<meta name="description">` accurate in that file so crawlers see complete metadata without relying on client-side JS.

**Widget Embed Content Isolation**
Widget endpoints (`/widget/search/{apiKey}`, `/widget/showcase/{apiKey}`) in `routes/ghl-widget.php` return HTML shells for iframe embeds. These should use appropriate `X-Robots-Tag: noindex, nofollow` headers to prevent widget content from competing with primary listing pages in search results.

**Dashboard vs marketing differentiation**
Authenticated dashboard and Filament user dashboard views should use `noindex` where appropriate so search engines do not index account-specific content. The platform home and GHL install surfaces need clear titles and descriptions for onboarding keywords.

## Warning

The GHL OAuth callback and widget loader endpoints may inadvertently expose location-specific content without proper robots controls. Verify that all `/leadconnector/*` and `/widget/*` routes include appropriate indexing directives—location-specific data in these flows can create thin content issues if indexed.