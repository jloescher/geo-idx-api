---
name: instrumenting-product-metrics
description: Defines product events, funnels, and activation metrics for the Quantyra GeoIDX platform including GHL marketplace install flow, lead capture, billing conversions, and MLS data access patterns.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Instrumenting Product Metrics Skill

This skill instruments the Quantyra GeoIDX platform—covering GHL Marketplace onboarding, Bridge MLS proxy usage, lead capture widgets, and subscription billing—to enable funnel analysis and activation tracking.

## Quick Start

Add event tracking by writing to existing audit tables or emitting structured logs:

```php
// Bridge MLS proxy access (already logged automatically)
// See app/Services/Bridge/BridgeProxyAuditLogger.php

// Manual GHL API audit (service handles this automatically)
use App\Ghl\Services\GhlAuditService;
$audit->logApiCall($endpoint, $latencyMs, $status, $locationId);

// Lead captured funnel event
\App\Ghl\Sync\Models\QuantyraLead::create([
    'ghl_location_id' => $locationId,
    'lead_type' => 'showing_request', // maps to ghl_lead_mappings
    'payload' => json_encode($fields),
]);
```

Run aggregate queries for funnel analysis:

```bash
# GHL install funnel (OAuth → Installed)
psql -c "SELECT 
    COUNT(DISTINCT ghl_company_id) as started_oauth,
    COUNT(DISTINCT CASE WHEN status = 'active' THEN ghl_company_id END) as active_installs
FROM ghl_oauth_tokens;"

# Lead capture conversion (per location)
psql -c "SELECT 
    ghl_location_id,
    COUNT(*) as leads_captured,
    MAX(created_at) as last_capture
FROM quantyra_leads 
GROUP BY ghl_location_id;"

# MLS teaser vs full access (domain-scoped)
psql -c "SELECT 
    domain_slug,
    request_type,
    COUNT(*) as requests,
    AVG(listing_count) as avg_listings
FROM bridge_proxy_audit_logs 
WHERE logged_at > NOW() - INTERVAL '7 days'
GROUP BY domain_slug, request_type;"
```

## Key Concepts

**Activation Events** — Stored in database tables designed for audit and analytics:

| Event | Table | Key Fields |
|-------|-------|------------|
| GHL OAuth completed | `ghl_oauth_tokens` | `ghl_company_id`, `ghl_location_id`, `user_type`, `status` |
| Location installed | `ghl_installed_locations` | `subscription_status`, `mls_request_count`, `lead_count` |
| Widget lead captured | `quantyra_leads` | `lead_type`, `ghl_location_id`, `created_at` |
| Lead synced to GHL | `ghl_sync_logs` | `sync_status`, `ghl_contact_id`, `ghl_opportunity_id` |
| MLS data accessed | `bridge_proxy_audit_logs` | `domain_slug`, `token_name`, `request_type`, `listing_count` |
| Webhook received | `ghl_webhook_events` | `event_type`, `processing_status` |

**Teaser Gating Metrics** — Revenue-relevant caps tracked in:
- `bridge_proxy_audit_logs.listing_count` — actual items returned (capped at 3 for teaser)
- `ghl_installed_locations.mls_request_count` — API usage per location

**Subscription State Machine** — Managed via Laravel Cashier + custom fields:
- `ghl_installed_locations.subscription_status`: `none` → `trial` → `active`/`past_due` → `cancelled`

## Common Patterns

### Funnel: GHL Marketplace Install

Track progression from OAuth start to active widget:

```php
// 1. OAuth initiated (routes/ghl-web.php)
// GET /oauth/leadconnector/authorize
// Session state created, redirects to GHL

// 2. OAuth callback succeeded
// GET /oauth/leadconnector/callback
// Token persisted to ghl_oauth_tokens

// 3. URL registration completed
// POST /leadconnector/register-urls
// Creates ghl_registered_urls + ghl_widget_configs

// 4. Installation active
// ghl_installed_locations.status = 'active'
```

Query for drop-off analysis:
```sql
SELECT 
    date_trunc('day', created_at) as day,
    user_type,
    COUNT(*) as tokens_created,
    COUNT(CASE WHEN status = 'active' THEN 1 END) as still_active
FROM ghl_oauth_tokens
GROUP BY day, user_type;
```

### Funnel: Lead Capture to CRM Sync

```php
// Widget POST → quantyra_leads record created
// Dispatches SyncLeadToGhlJob (async queue)

// Job processes through LeadSyncService:
// - Maps lead_type to GhlLeadMapping config
// - Creates contact via GhlApiClient
// - Optionally creates opportunity
// - Writes ghl_sync_logs record
```

Track sync health:
```sql
SELECT 
    sync_status,
    COUNT(*),
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at))) as avg_sync_seconds
FROM ghl_sync_logs
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY sync_status;
```

### Metric: API Usage by Subscription Tier

Bridge proxy usage maps to billing via request volume and teaser limits:

```php
// In BridgeProxyController (domain-authenticated calls)
// ListingsCacheService handles cache hits
// BridgeTeaser applies caps for non-idx:full tokens

// Logged to bridge_proxy_audit_logs with:
// - domain_slug (links to subscription tier)
// - listing_count (revenue-relevant if capped)
// - request_type (listings, images, etc.)
```

### Instrumenting New Events

For new product surfaces, extend existing audit infrastructure:

```php
// Option A: Database table (for structured querying)
DB::table('product_events')->insert([
    'event_name' => 'widget_config_viewed',
    'user_id' => $locationId,
    'properties' => json_encode(['widget' => 'search', 'theme' => 'dark']),
    'occurred_at' => now(),
]);

// Option B: Structured log line (for log aggregation)
Log::channel('product')->info('widget.config_viewed', [
    'location_id' => $locationId,
    'widget' => 'search',
    'referrer' => $request->header('Referer'),
]);
```