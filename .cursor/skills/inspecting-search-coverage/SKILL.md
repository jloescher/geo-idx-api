---
name: inspecting-search-coverage
description: Audits technical and on-page search coverage across the idx-api codebase, including Bridge MLS listing filters, GIS parcel queries, and GHL widget search endpoints.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Inspecting Search Coverage Skill

This skill audits search functionality across the Quantyra IDX API, covering Bridge MLS proxy filters (`/api/v1/listings`), GIS parcel geometry queries (`/api/v1/gis`), and GHL widget search surfaces. It identifies query parameter handling, validation logic, and coverage gaps.

## Quick Start

```bash
# Find all search/filter query parameter handlers
grep -r "filters\|query\|search" app/Http/Controllers/Api/ --include="*.php" | head -30

# Locate GIS bbox/radius search implementation
grep -r "bbox\|radius\|latitude\|longitude" app/Services/ --include="*.php"

# Check Bridge query parameter forwarding
grep -r "BRIDGE\|forward\|query" app/Services/Bridge/ --include="*.php" | grep -i param

# List all controller methods handling search
grep -rn "public function.*search\|public function.*filter\|public function.*list" app/Http/Controllers/Api/
```

## Key Concepts

**Bridge Filter Forwarding**: The `BridgeProxyController` forwards `filters` query parameters to Bridge Data Output. When `filters` is present, caching is bypassed. Check `app/Http/Controllers/Api/BridgeProxyController.php`.

**GIS Spatial Queries**: The `GisProxyService` handles `bbox` (west,south,east,north) and `lat`/`lng` + `radius` (meters) parameters. Spatial queries convert to ArcGIS feature server queries with envelope intersection. See `app/Services/GisProxyService.php`.

**Teaser Gating Impact**: Search results are capped (3 listings, 40 GIS features) for non-`idx:full` tokens. The `BridgeTeaser` service truncates list-shaped JSON after the proxy fetch.

**Query Parameter Validation**: GIS uses `GisProxyRequest` form request for bbox/radius validation. Bridge passes most parameters through without validation, with exceptions (`domain`, `teaser` are stripped).

## Common Patterns

**Audit Bridge Search Coverage**:
```bash
# Check what query params are forwarded vs stripped
grep -A5 "query\|input" app/Http/Controllers/Api/BridgeProxyController.php | head -40
```

**Audit GIS Search Coverage**:
```bash
# Review bbox vs radius parameter handling
grep -B2 -A10 "bbox\|buildBoundingBox" app/Services/GisProxyService.php

# Check coordinate validation patterns
grep -rn "numeric\|min\|max" app/Http/Requests/GisProxyRequest.php
```

**Check Filter Cache Bypass Logic**:
```bash
# Find where filters disable caching
grep -rn "filters" app/Services/Bridge/ListingsCacheService.php
```

**Verify Widget Search Surfaces**:
```bash
# Widget search/lead-form endpoints
grep -rn "widget.*search\|widget.*form" routes/ghl-widget.php