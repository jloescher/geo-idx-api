# Growth Engineering

## When to use
Implement infrastructure patterns that reduce marginal costs, enable rapid experimentation, and create viral distribution loops through the widget embed system.

## Patterns

**Multi-Tier GIS Caching**
The GIS proxy uses three cache layers: (1) Laravel Cache edge (`GIS_EDGE_CACHE_TTL` default 900s), (2) PostgreSQL `gis_cache` origin (30-90 day max age), (3) Filesystem backup. Generation-based invalidation via `gis_source_states` ensures cache hits without stale data, reducing ArcGIS egress costs as volume scales.

**Widget Viral Loop**
Each installed location receives a unique widget API key (`qh_*` prefix). Embed snippets in `/leadconnector/installation-complete` encourage agencies to place widgets on client sites. The Origin validation prevents unauthorized use while allowing unlimited legitimate embeds per registered domain.

**Queue-Based Lead Processing**
`POST /widget/api/leads` creates `QuantyraLead` records synchronously but dispatches `SyncLeadToGhlJob` asynchronously. This pattern absorbs traffic spikes during marketing campaigns without dropping leads or slowing response times. Monitor via `ghl_sync_logs.sync_status` for failed GHL CRM writes.

## Warning
The `GIS_MAX_BBOX_SPAN_DEG` guard (default 0.35 degrees) rejects abusive queries with HTTP 422. Ensure map zoom constraints in widget JavaScript prevent users from triggering this limit during normal pan/zoom operations.