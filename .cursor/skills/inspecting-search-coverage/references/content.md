# Content Search Coverage

When to use: Auditing content quality, keyword relevance, and semantic coverage for property listings, MLS descriptions, and IDX marketing copy.

## Patterns

**MLS Listing Content Enrichment**
Bridge API responses include `Media`, `PropertyDescription`, and `PublicRemarks` fields. The `BridgeTeaser` service truncates this content for non-`idx:full` requests. Full access tokens should receive complete, unmodified MLS content to support rich snippet eligibility and semantic search relevance.

**GIS Parcel Context Layer**
The GIS proxy in `app/Services/GisProxyService.php` returns GeoJSON with a `meta` foreign member containing `source_used`, `county_hint`, and `context_layers`. This metadata can be used to enrich listing pages with neighborhood context, school districts, and parcel boundaries—improving topical authority without duplicating MLS data.

**GHL Lead Form Content Gaps**
Widget lead forms (`/widget/lead-form/{apiKey}`) collect user intent through `lead_type` parameters. Analyze `app/Ghl/Sync/Models/QuantyraLead` and `GhlLeadMapping` to identify high-value keywords from lead submissions that indicate content gaps in the main IDX interface.

## Warning

The teaser gating logic (3 listings for domain auth, 40 features for GIS teaser) can create thin content scenarios if teaser pages are inadvertently indexed. Ensure that partial content views either carry `noindex` directives or are canonicalized to their full-content equivalents when accessible via subscription.