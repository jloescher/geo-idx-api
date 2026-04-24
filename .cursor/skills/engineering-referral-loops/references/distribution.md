# Distribution Mechanics for Embeddable Widgets

## When to use
When designing how partners consume your service through JavaScript embeds, API keys, or cross-origin integrations where origin validation and CORS matter.

## Project-relevant patterns

**Three-phase widget middleware**
Validate API key → check Origin/Referer against `registered_urls` → append CORS headers. This pattern prevents unauthorized domains from loading widgets while allowing legitimate embeds to POST leads cross-origin.

**API-key scoped configuration**
Distribute unique widget API keys (`qh_{hash}`) per installation stored in `ghl_registered_urls`. Each key maps to a location's theme config and allowed origins, enabling multi-tenant embeds without session cookies.

**Graceful degradation with fallbacks**
When ArcGIS parcel data fails, return `meta.degraded=true` with a Leaflet-compatible OSM tile URL in `meta.leaflet_fallback`. Widgets can switch to raster tiles without breaking the user experience.

## Warning
Never trust the Origin header alone for write operations. The widget lead POST validates the API key *and* Origin, but the key is the authoritative credential—Origin checking alone can be spoofed in some client configurations.