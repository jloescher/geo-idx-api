# Technical Search Coverage

When to use: Auditing URL structure, crawlability, indexation controls, and technical SEO foundations for the Bridge MLS proxy and GHL widget endpoints.

## Patterns

**Bridge Listing URL Architecture**
The `/api/v1/listings` endpoint supports query parameters (`limit`, `offset`, `filters`) that should be reflected in canonical URL structures. Check `app/Http/Controllers/Api/BridgeProxyController.php` for parameter forwarding logic. Ensure filtered views have self-referencing canonicals and pagination uses `rel="next"`/`rel="prev"`.

**Image Proxy URL Normalization**
The `/images/{listingKey}/{photoId}` pattern in `app/Http/Controllers/Api/ImageProxyController.php` serves MLS photos with immutable cache headers. Verify that photo URLs in JSON responses (rewritten by `BridgeImageUrlRewriter`) use consistent canonical paths to avoid duplicate image indexing.

**Domain-Based Access Control Impact**
The `DomainOrTokenAuth` middleware gates all search endpoints. Ensure that `X-Domain-Slug` variations and `?domain=` query alternatives do not create indexable duplicate content. Teaser responses (capped at 3 listings) should carry `noindex` directives when they represent truncated views.

## Warning

Bridge query parameters are forwarded without sanitization to the upstream API. The `filters` parameter bypasses the 15-minute listings cache—aggressive filtering can cause cache fragmentation and expose internal parameter combinations to crawlers if not properly canonicalized.