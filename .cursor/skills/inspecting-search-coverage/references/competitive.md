# Competitive Search Coverage

When to use: Benchmarking idx-api search functionality against real estate platforms and identifying feature gaps for SEO differentiation.

## Patterns

**Filter Sophistication Comparison**
Bridge supports `filters` query parameters for advanced listing search. Compare your implementation's filter granularity (price range, property type, bedrooms, bathrooms, square footage) against Zillow, Realtor.com, and Redfin. The `BridgeProxyController` forwards filters directly—audit which filter combinations create unique, indexable result pages.

**Map-Based Search Parity**
The GIS proxy's `bbox` and radius search capabilities compete with map-based property discovery on major platforms. Benchmark against competitors' map search UX: support for drawing custom polygons, school boundary overlays, and commute-time search. The `GisProxyService` failover chain (statewide → county → degraded) should match competitor reliability.

**GHL Integration Differentiation**
Most IDX providers lack deep GoHighLevel CRM integration. The widget system (`/widget/loader.js`, `/widget/api/leads`) and lead sync pipeline (`SyncLeadToGhlJob`) create competitive moats. Ensure search functionality exposes lead capture opportunities that competitors cannot match—gated saved searches, automated follow-up triggers, and subscription-tier content access.

## Warning

Competitors like Zillow receive direct MLS data feeds with fewer compliance restrictions. The idx-api teaser gating and domain authorization requirements create inherent content depth limitations. Do not attempt to compete on listing volume alone—focus on agent workflow integration (GHL sync), local market expertise (GIS layers), and subscriber-exclusive content as differentiators.