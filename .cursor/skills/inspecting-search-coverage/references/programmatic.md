# Programmatic Search Coverage

When to use: Analyzing dynamically generated pages, URL patterns for property listings, and scaleable SEO strategies for large MLS datasets.

## Patterns

**Listing Detail Page Generation**
Individual listing endpoints (`/api/v1/listings/{listingId}`, `/api/v1/properties/{listingKey}`) serve as the foundation for programmatic SEO. Each listing should map to a canonical frontend URL pattern like `/listings/{city}/{address}-{listingKey}` to create indexable, location-rich pages at scale.

**ArcGIS Feature Programmatic Mapping**
The GIS proxy supports `bbox` and `lat`/`lng` + `radius` queries that can generate neighborhood-level landing pages. Use `GisProxyService` envelope queries to create programmatic pages for cities, ZIP codes, and custom neighborhoods with unique boundary geometries.

**GHL Location-Specific Content**
Installed locations (`ghl_installed_locations` table) with registered URLs (`ghl_registered_urls`) create opportunities for programmatic subdomains or subdirectories. Each location's widget embeds and IDX pages should generate unique, indexable content based on their specific MLS market area and agent branding.

## Warning

Programmatic generation of listing pages must respect MLS compliance requirements—Stellar MLS data cannot be displayed on unauthorized domains. The `domains` table controls which hostnames may display listing content; violating this can result in MLS access revocation and search engine penalties for duplicate content across unauthorized domains.