# GoHighLevel Marketplace App Integration Design

**Date**: 2026-04-22  
**Project**: Quantyra GeoIDX  
**Scope**: GHL Marketplace OAuth integration, lead sync, webhooks, widget architecture

---

## Executive Summary

This design document specifies the complete GoHighLevel (GHL) Marketplace App integration for the Quantyra GeoIDX platform. The integration enables GHL agencies and real estate agents to install Quantyra's IDX solution directly into their GHL CRM, receiving verified MLS leads as contacts and opportunities automatically.

**Revenue Impact**: One-click install → instant lead flow → higher subscription conversions across GHL's 70,000+ agencies and 600,000+ businesses.

---

## Design Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Revenue Model | Hybrid (Agency + Location) | Agency installs unlock multiple locations; direct installs for individual agents. Maximum reach. |
| Lead Mapping | Contacts + Opportunities + Tags | Contacts for all leads; Opportunities for high-value (showing requests, pre-approvals); Tags for filtering and automation. |
| Subscription | External Stripe + Webhook Sync | Existing Laravel Cashier handles billing; webhook updates GHL subscription status via tags. |
| Scopes | Maximum (Full CRM Integration) | Requested all scopes for future flexibility: contacts, opportunities, locations, users, conversations, calendars, businesses, forms, campaigns, links. |
| Webhooks | Full-Flow (All Events) | INSTALL, UNINSTALL, ContactCreate/Update/Delete, ContactTagUpdate, OpportunityCreate/StatusUpdate/StageUpdate, NoteCreate, TaskCreate. |
| Audit Logging | Enhanced (Full Trail) | timestamp, location_id, token_id, api_endpoint, response_status, latency_ms, error_details. Meets Stellar MLS compliance. |
| Architecture | Modular Package (Service-Oriented) | Separation: OAuth/, Sync/, Webhooks/, Api/, Widgets/ modules. Clear boundaries, testable, scalable. |
| Widget Type | JS Embeddable Widgets | Users embed property search, lead capture, showcase widgets on GHL-hosted or external websites. |
| URL Registration | MLS Compliance Required | Users must register website URLs during onboarding for Stellar MLS compliance. |

---

## URL Architecture

| Service | Domain | Purpose |
|---------|--------|--------|
| IDX App | `idx.quantyralabs.cc` | IDX app for GHL and independent dashboard (frontend, checkout, embeds) |
| IDX API | `idx-api.quantyralabs.cc` | IDX API for API endpoints and JS embed widgets (OAuth, webhooks, GHL routes, widget loader) |
| IDX images | `idx-images.quantyralabs.cc` | IDX image rewrite proxy (MLS CDN in front of Bridge-approved paths) |

---

## OAuth 2.0 Configuration

### Endpoints

| Endpoint | Full URL |
|----------|----------|
| Redirect URI | `https://idx-api.quantyralabs.cc/oauth/leadconnector/callback` |
| Webhook URI | `https://idx-api.quantyralabs.cc/webhooks/leadconnector` |
| Installation URL | `https://idx-api.quantyralabs.cc/leadconnector/install` |
| Subscription Checkout | `https://idx.quantyralabs.cc/subscribe?ghl_location={locationId}` |

### OAuth Flow (Hybrid)

The integration supports two installation types:

1. **Agency Installation (`user_type=Company`)**:
   - Agency admin installs app
   - Receives Company-level access token + refresh token
   - Can generate Location tokens for each sub-account via `/oauth/locationToken`
   - Tokens stored with `userType='Company'` and `companyId`

2. **Location Installation (`user_type=Location`)**:
   - Individual agent/lender installs directly
   - Receives Location-level access token + refresh token
   - Token stored with `userType='Location'`, `companyId`, and `locationId`

### Scopes Requested

```
contacts.readonly contacts.write 
opportunities.readonly opportunities.write 
locations.readonly locations/tags.write 
users.readonly 
oauth.readonly oauth.write 
conversations.readonly conversations/message.write 
workflows.readonly 
calendars/events.write 
businesses.readonly 
forms.readonly 
links.write 
campaigns.readonly
```

---

## Directory Structure

```
idx-api/
├── app/
│   ├── Ghl/
│   │   ├── OAuth/
│   │   │   ├── Controllers/
│   │   │   │   ├── AuthorizeController.php
│   │   │   │   ├── CallbackController.php
│   │   │   │   ├── RefreshController.php
│   │   │   │   └── UrlRegistrationController.php
│   │   │   ├── Services/
│   │   │   │   ├── OAuthService.php
│   │   │   │   ├── TokenRefreshService.php
│   │   │   │   └── LocationTokenService.php
│   │   │   ├── Middleware/
│   │   │   │   ├── GhlAuthMiddleware.php
│   │   │   │   └── ValidateStateMiddleware.php
│   │   │   ├── Models/
│   │   │   │   ├── GhlOAuthToken.php
│   │   │   │   └── GhlInstalledLocation.php
│   │   │   │   ├── GhlRegisteredUrl.php
│   │   ├── Sync/
│   │   │   ├── Services/
│   │   │   │   ├── ContactCreator.php
│   │   │   │   ├── OpportunityCreator.php
│   │   │   │   ├── TagManager.php
│   │   │   │   ├── LeadSyncService.php
│   │   │   │   ├── SubscriptionSyncService.php
│   │   │   ├── Jobs/
│   │   │   │   ├── SyncLeadToGhlJob.php
│   │   │   │   ├── RefreshTokenJob.php
│   │   │   │   ├── SyncSubscriptionStatusJob.php
│   │   │   │   ├── BulkLocationTokenJob.php
│   │   │   ├── Models/
│   │   │   │   ├── GhlSyncLog.php
│   │   │   │   ├── GhlLeadMapping.php
│   │   ├── Webhooks/
│   │   │   ├── Controllers/
│   │   │   │   ├── WebhookController.php
│   │   │   ├── Handlers/
│   │   │   │   ├── InstallHandler.php
│   │   │   │   ├── UninstallHandler.php
│   │   │   │   ├── ContactCreateHandler.php
│   │   │   │   ├── ContactUpdateHandler.php
│   │   │   │   ├── ContactDeleteHandler.php
│   │   │   │   ├── ContactTagUpdateHandler.php
│   │   │   │   ├── OpportunityCreateHandler.php
│   │   │   │   ├── OpportunityStatusHandler.php
│   │   │   │   ├── OpportunityStageHandler.php
│   │   │   │   ├── NoteCreateHandler.php
│   │   │   │   ├── TaskCreateHandler.php
│   │   │   ├── Services/
│   │   │   │   ├── WebhookSignatureService.php
│   │   │   │   ├── WebhookDispatcher.php
│   │   │   ├── Models/
│   │   │   │   ├── GhlWebhookEvent.php
│   │   ├── Api/
│   │   │   ├── Clients/
│   │   │   │   ├── GhlApiClient.php
│   │   │   │   ├── ContactClient.php
│   │   │   │   ├── OpportunityClient.php
│   │   │   │   ├── LocationClient.php
│   │   │   │   ├── UserClient.php
│   │   │   │   ├── ConversationClient.php
│   │   │   │   ├── CalendarClient.php
│   │   │   │   ├── BusinessClient.php
│   │   │   │   ├── FormClient.php
│   │   │   │   ├── CampaignClient.php
│   │   │   │   ├── LinkClient.php
│   │   │   ├── Middleware/
│   │   │   │   ├── RateLimitMiddleware.php
│   │   │   │   ├── RetryMiddleware.php
│   │   ├── Controllers/
│   │   │   ├── GhlApiController.php
│   │   │   ├── InstallationController.php
│   │   ├── Widgets/
│   │   │   ├── Controllers/
│   │   │   │   ├── WidgetLoaderController.php
│   │   │   │   ├── WidgetConfigController.php
│   │   │   │   ├── WidgetSearchController.php
│   │   │   │   ├── WidgetLeadController.php
│   │   │   │   ├── WidgetShowcaseController.php
│   │   │   ├── Services/
│   │   │   │   ├── WidgetConfigService.php
│   │   │   │   ├── WidgetApiKeyService.php
│   │   │   │   ├── WidgetCorsService.php
│   │   │   ├── Middleware/
│   │   │   │   ├── ValidateWidgetApiKey.php
│   │   │   │   ├── ValidateWidgetOrigin.php
│   │   │   │   ├── WidgetRateLimit.php
│   │   │   ├── Models/
│   │   │   │   ├── GhlWidgetConfig.php
│   │   ├── Services/
│   │   │   ├── GhlAuditService.php
│   │   │   ├── GhlConfigService.php
│   │   ├── Routes/
│   │   │   ├── ghl-web.php
│   │   │   ├── ghl-api.php
│   │   │   ├── ghl-webhook.php
│   │   │   ├── ghl-widget.php
├── config/
│   ├── ghl.php
│   ├── ghl/oauth.php
│   ├── ghl/sync.php
│   ├── ghl/webhooks.php
├── database/
│   ├── migrations/
│   │   ├── ghl/
│   │   │   ├── 2026_04_22_000001_create_ghl_oauth_tokens_table.php
│   │   │   ├── 2026_04_22_000002_create_ghl_installed_locations_table.php
│   │   │   ├── 2026_04_22_000003_create_ghl_sync_logs_table.php
│   │   │   ├── 2026_04_22_000004_create_ghl_webhook_events_table.php
│   │   │   ├── 2026_04_22_000005_create_ghl_audit_logs_table.php
│   │   │   ├── 2026_04_22_000006_create_ghl_lead_mappings_table.php
│   │   │   ├── 2026_04_22_000007_create_ghl_registered_urls_table.php
│   │   │   ├── 2026_04_22_000008_create_ghl_widget_configs_table.php
│   ├── seeders/
│   │   ├── GhlConfigSeeder.php
├── tests/
│   ├── Ghl/
│   │   ├── OAuth/
│   │   │   ├── OAuthFlowTest.php
│   │   │   ├── TokenRefreshTest.php
│   │   │   ├── LocationTokenTest.php
│   │   ├── Sync/
│   │   │   ├── ContactCreatorTest.php
│   │   │   ├── OpportunityCreatorTest.php
│   │   │   ├── LeadSyncTest.php
│   │   ├── Webhooks/
│   │   │   ├── InstallHandlerTest.php
│   │   │   ├── UninstallHandlerTest.php
│   │   │   ├── WebhookSignatureTest.php
│   │   ├── Api/
│   │   │   ├── GhlApiClientTest.php
│   │   │   ├── ContactClientTest.php
│   │   ├── Widgets/
│   │   │   ├── WidgetApiKeyTest.php
│   │   │   ├── WidgetOriginTest.php
```

---

## Database Schema

### Table: `ghl_oauth_tokens`

Core token storage for both Agency and Location tokens.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ghl_company_id` | VARCHAR(50) | Agency/company ID from GHL |
| `ghl_location_id` | VARCHAR(50) | Location ID (NULL for Agency tokens) |
| `ghl_user_id` | VARCHAR(50) | User who installed the app |
| `access_token` | TEXT | Encrypted JWT access token |
| `refresh_token` | TEXT | Encrypted refresh token |
| `refresh_token_id` | VARCHAR(50) | GHL refresh token identifier |
| `user_type` | ENUM | 'Company' or 'Location' |
| `expires_at` | TIMESTAMP | Access token expiry (~24 hours) |
| `refresh_expires_at` | TIMESTAMP | Refresh token expiry (1 year) |
| `scopes` | TEXT | JSON array of granted scopes |
| `is_bulk_install` | BOOLEAN | True for bulk agency installs |
| `installed_at` | TIMESTAMP | Installation timestamp |
| `last_refreshed_at` | TIMESTAMP | Last token refresh |
| `status` | ENUM | 'active', 'refreshed', 'expired', 'revoked' |
| `revoked_at` | TIMESTAMP | Token revocation timestamp |
| `revoke_reason` | VARCHAR(255) | Reason for revocation |
| `deleted_at` | TIMESTAMP | Soft delete for uninstall cleanup |

### Table: `ghl_installed_locations`

Tracks locations under each agency (for bulk installs).

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ghl_oauth_token_id` | BIGINT | Reference to ghl_oauth_tokens |
| `ghl_company_id` | VARCHAR(50) | Agency ID |
| `ghl_location_id` | VARCHAR(50) | Location ID |
| `location_name` | VARCHAR(255) | Location name |
| `location_address` | TEXT | Location address |
| `location_timezone` | VARCHAR(50) | Location timezone |
| `is_whitelabel` | BOOLEAN | Whitelabel flag |
| `whitelabel_domain` | VARCHAR(255) | Whitelabel domain |
| `whitelabel_logo_url` | TEXT | Whitelabel logo URL |
| `subscription_status` | ENUM | 'none', 'trial', 'active', 'past_due', 'cancelled' |
| `subscription_id` | VARCHAR(50) | Stripe subscription ID |
| `mls_request_count` | INTEGER | MLS request count (teaser stats) |
| `lead_count` | INTEGER | Lead count (teaser stats) |
| `last_activity_at` | TIMESTAMP | Last activity timestamp |
| `installed_at` | TIMESTAMP | Installation timestamp |
| `uninstalled_at` | TIMESTAMP | Uninstallation timestamp |
| `status` | ENUM | 'active', 'uninstalled' |

### Table: `ghl_sync_logs`

Tracks lead sync attempts to GHL.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ghl_location_id` | VARCHAR(50) | GHL location ID |
| `quantyra_lead_id` | BIGINT | Reference to local leads table |
| `ghl_contact_id` | VARCHAR(50) | GHL contact ID after sync |
| `ghl_opportunity_id` | VARCHAR(50) | GHL opportunity ID (if created) |
| `sync_type` | ENUM | 'contact', 'opportunity', 'tag', 'subscription' |
| `lead_type` | VARCHAR(50) | 'showing_request', 'pre_approval', etc. |
| `sync_status` | ENUM | 'pending', 'success', 'failed', 'retrying' |
| `retry_count` | INTEGER | Number of retries |
| `error_message` | TEXT | Error message if failed |
| `request_payload` | JSONB | Data sent to GHL |
| `response_payload` | JSONB | GHL response |
| `created_at` | TIMESTAMP | Log creation timestamp |
| `completed_at` | TIMESTAMP | Sync completion timestamp |

### Table: `ghl_webhook_events`

Stores received GHL webhook payloads.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `webhook_id` | VARCHAR(50) | Unique webhook event ID from GHL |
| `event_type` | VARCHAR(50) | INSTALL, UNINSTALL, ContactCreate, etc. |
| `ghl_app_id` | VARCHAR(50) | GHL app ID |
| `ghl_company_id` | VARCHAR(50) | Company ID |
| `ghl_location_id` | VARCHAR(50) | Location ID |
| `payload` | JSONB | Full webhook JSON payload |
| `processing_status` | ENUM | 'received', 'processing', 'processed', 'failed' |
| `processed_at` | TIMESTAMP | Processing completion timestamp |
| `handler_class` | VARCHAR(255) | Handler that processed this event |

### Table: `ghl_audit_logs`

Stellar MLS compliance audit logging (enhanced full trail).

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `timestamp` | TIMESTAMP | Request timestamp |
| `ghl_company_id` | VARCHAR(50) | Company ID |
| `ghl_location_id` | VARCHAR(50) | Location ID |
| `ghl_user_id` | VARCHAR(50) | User ID |
| `ghl_oauth_token_id` | BIGINT | Token used for request |
| `listing_id` | VARCHAR(50) | MLS listing ID accessed |
| `mls_request_count` | INTEGER | MLS request count |
| `api_endpoint` | VARCHAR(255) | API endpoint called |
| `request_method` | VARCHAR(10) | HTTP method |
| `lead_type` | VARCHAR(50) | Lead type if applicable |
| `sync_status` | VARCHAR(50) | Sync status |
| `response_status` | INTEGER | HTTP response status |
| `latency_ms` | INTEGER | Request latency in milliseconds |
| `error_details` | TEXT | Error details if failed |
| `is_mls_data_access` | BOOLEAN | MLS data access flag |
| `compliance_verified` | BOOLEAN | Compliance verification flag |

### Table: `ghl_lead_mappings`

Maps Quantyra lead types to GHL objects/tags.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `quantyra_lead_type` | VARCHAR(50) | Lead type name |
| `creates_contact` | BOOLEAN | Creates contact flag |
| `creates_opportunity` | BOOLEAN | Creates opportunity flag |
| `opportunity_pipeline` | VARCHAR(50) | GHL pipeline ID |
| `opportunity_stage` | VARCHAR(50) | GHL stage ID |
| `default_tags` | JSONB | Tags to apply on creation |
| `domain_tag_prefix` | BOOLEAN | Add domain-specific tag |
| `is_high_value` | BOOLEAN | High-value lead (creates opportunity) |

### Table: `ghl_registered_urls`

MLS compliance URL storage for widget access.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ghl_oauth_token_id` | BIGINT | Reference to ghl_oauth_tokens |
| `ghl_location_id` | VARCHAR(50) | Location ID |
| `ghl_company_id` | VARCHAR(50) | Company ID |
| `primary_url` | VARCHAR(255) | Main website URL (required) |
| `additional_urls` | JSONB | Array of additional URLs |
| `widget_api_key` | VARCHAR(64) | Unique API key for widget access |
| `widget_access_enabled` | BOOLEAN | Widget access flag |
| `integration_type` | ENUM | 'ghl_website', 'external_website', 'both' |
| `mls_agreement_acknowledged` | BOOLEAN | MLS agreement acknowledgment |
| `stellar_mls_approved` | BOOLEAN | Stellar MLS approval flag |

### Table: `ghl_widget_configs`

Widget configuration per location.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ghl_location_id` | VARCHAR(50) | Location ID |
| `ghl_registered_url_id` | BIGINT | Reference to registered URLs |
| `widget_theme` | ENUM | 'light', 'dark', 'custom' |
| `primary_color` | VARCHAR(7) | Primary hex color |
| `secondary_color` | VARCHAR(7) | Secondary hex color |
| `font_family` | VARCHAR(100) | Font family |
| `default_widget_type` | ENUM | 'search', 'showcase', 'lead-form' |
| `listings_per_page` | INTEGER | Listings per page |
| `map_enabled` | BOOLEAN | Map display flag |
| `show_lead_form` | BOOLEAN | Lead form display flag |
| `default_search_area` | VARCHAR(100) | Geographic filter |
| `property_types` | JSONB | Property types filter |
| `min_price` | INTEGER | Minimum price filter |
| `max_price` | INTEGER | Maximum price filter |
| `gate_after_views` | INTEGER | Lead gate threshold |
| `require_otp` | BOOLEAN | OTP requirement flag |

---

## Routes Summary

| Route | Method | Purpose | Middleware |
|-------|--------|---------|------------|
| `/oauth/leadconnector/authorize` | GET | Redirect to GHL authorization URL | `web` |
| `/oauth/leadconnector/callback` | GET | Handle OAuth callback, exchange code for tokens | `web`, `ValidateState` |
| `/oauth/leadconnector/refresh` | POST | Manual token refresh (admin use) | `web`, `auth` |
| `/leadconnector/register-urls` | GET | URL registration form (after OAuth) | `web`, `GhlSession` |
| `/leadconnector/register-urls` | POST | Submit registered URLs | `web`, `GhlSession` |
| `/leadconnector/install` | GET | App installation landing page | `web` |
| `/leadconnector/installation-complete` | GET | Success page with widget codes | `web`, `GhlSession` |
| `/webhooks/leadconnector` | POST | Receive all GHL webhook events | `VerifyGhlWebhookSignature` |
| `/api/leadconnector/leads` | GET | List leads for authenticated GHL location | `GhlAuth` |
| `/api/leadconnector/leads/{id}` | GET | Single lead details | `GhlAuth` |
| `/api/leadconnector/subscriptions` | GET | Subscription status for location | `GhlAuth` |
| `/api/leadconnector/stats` | GET | Teaser stats (MLS request count, lead count) | `GhlAuth` |
| `/api/leadconnector/config` | GET | GHL configuration for location | `GhlAuth` |
| `/widget/loader.js` | GET | Widget loader script | `ValidateWidgetApiKey` |
| `/widget/config/{apiKey}` | GET | Widget configuration | `ValidateWidgetApiKey`, `ValidateWidgetOrigin` |
| `/widget/search/{apiKey}` | GET | Property search widget | `ValidateWidgetApiKey`, `ValidateWidgetOrigin` |
| `/widget/lead-form/{apiKey}` | GET | Lead capture widget | `ValidateWidgetApiKey`, `ValidateWidgetOrigin` |
| `/widget/showcase/{apiKey}` | GET | Featured listings showcase | `ValidateWidgetApiKey`, `ValidateWidgetOrigin` |
| `/widget/api/leads` | POST | Create lead from widget | `ValidateWidgetApiKey`, `ValidateWidgetOrigin` |
| `/leadconnector/embed/{locationId}` | GET | Redirect to full IDX embed on IDX App | `public` |

---

## Lead-to-GHL Object Mapping

### Tag Mapping Configuration

| Lead Type | Default Tags | Creates Opportunity? | Pipeline Stage |
|-----------|--------------|---------------------|----------------|
| `showing_request` | `quantyra-lead`, `showing-request`, `{domain}` | Yes | New Lead |
| `pre_approval` | `quantyra-lead`, `pre-approval`, `{domain}` | Yes | New Lead |
| `journey_question` | `quantyra-lead`, `journey-question`, `{domain}` | No | - |
| `general_inquiry` | `quantyra-lead`, `general-inquiry`, `{domain}` | No | - |
| `property_view` | `quantyra-lead`, `property-view`, `{domain}` | No | - |

### Subscription Status Tags (synced from Stripe)

| Stripe Status | GHL Tag Applied | Tag Removed |
|---------------|-----------------|-------------|
| `active` | `quantyra-active` | `quantyra-past-due`, `quantyra-cancelled` |
| `past_due` | `quantyra-past-due` | `quantyra-active` |
| `canceled` | `quantyra-cancelled` | `quantyra-active`, `quantyra-past-due` |
| `trialing` | `quantyra-trial` | - |

---

## Webhook Event Handlers

| Handler | Event Type | Primary Action | Secondary Actions |
|---------|------------|----------------|-------------------|
| `InstallHandler` | `INSTALL` | Store tokens in `ghl_oauth_tokens` | Create installed_location record, dispatch BulkLocationTokenJob if bulk, audit log |
| `UninstallHandler` | `UNINSTALL` | Revoke tokens | Mark location uninstalled, cleanup sync logs, audit log |
| `ContactCreateHandler` | `ContactCreate` | Log event | Update analytics, skip if created by Quantyra |
| `ContactUpdateHandler` | `ContactUpdate` | Log event | Check tag changes, update subscription status |
| `ContactDeleteHandler` | `ContactDelete` | Mark lead deleted | Cleanup references |
| `ContactTagUpdateHandler` | `ContactTagUpdate` | Check subscription tags | Update subscription_status in installed_locations |
| `OpportunityCreateHandler` | `OpportunityCreate` | Log to analytics | Track pipeline velocity |
| `OpportunityStatusHandler` | `OpportunityStatusUpdate` | Update status tracking | Trigger notifications if converted |
| `OpportunityStageHandler` | `OpportunityStageUpdate` | Track stage progression | Calculate lead-to-close time |
| `NoteCreateHandler` | `NoteCreate` | Store note reference | Link to Quantyra lead |
| `TaskCreateHandler` | `TaskCreate` | Store task reference | Link to Quantyra lead |

---

## Widget Architecture

### Widget Types

| Widget | Description | Embed Method |
|--------|-------------|--------------|
| `Property Search` | MLS listing search with map/list view | JS script + div container |
| `Lead Capture Form` | Showing request, pre-approval forms | JS script + div container |
| `Property Showcase` | Featured listings carousel | JS script + div container |
| `Market Stats` | Neighborhood statistics teaser | JS script + div container |
| `Full IDX Page` | Complete IDX experience | iframe or JS SPA |

### Widget Embed Code Examples

**Property Search Widget**:
```html
<div id="quantyra-search-widget"></div>
<script
  src="https://idx-api.quantyralabs.cc/widget/loader.js"
  data-widget="search"
  data-api-key="{WIDGET_API_KEY}"
  data-location-id="{GHL_LOCATION_ID}"
  data-theme="light"
></script>
```

**Lead Capture Widget**:
```html
<div id="quantyra-lead-widget"></div>
<script
  src="https://idx-api.quantyralabs.cc/widget/loader.js"
  data-widget="lead-form"
  data-api-key="{WIDGET_API_KEY}"
  data-location-id="{GHL_LOCATION_ID}"
  data-form-type="showing-request"
></script>
```

**Full IDX iframe**:
```html
<iframe
  src="https://idx.quantyralabs.cc/embed/{LOCATION_ID}"
  width="100%"
  height="800px"
  frameborder="0"
></iframe>
```

---

## Installation Flow

```
1. User clicks "Install" in GHL Marketplace
2. Redirect to /oauth/leadconnector/authorize
3. User authorizes in GHL OAuth server
4. GHL redirects to /oauth/leadconnector/callback?code=xxx
5. Exchange code for tokens, store in ghl_oauth_tokens
6. Redirect to /leadconnector/register-urls (URL registration form)
7. User enters primary URL, additional URLs, integration type
8. Submit form → Store in ghl_registered_urls
9. Generate widget_api_key, create ghl_widget_config default
10. Display success page with widget embed codes
11. User embeds widgets on their website(s)
12. Leads flow through widget → idx-api → GHL CRM
```

---

## GHL Marketplace App Registration Checklist

### Phase 1: Create App
- App Name: `Quantyra GeoIDX`
- Category: `Real Estate / IDX`
- Company: `Quantyra Labs LLC`
- Website: `https://quantyralabs.cc`

### Phase 2: Branding
- Logo: 512x512 PNG
- Preview images: 3-5 screenshots
- Video demo: Optional 30-60 second walkthrough

### Phase 3: Auth Configuration
- App Type: `Public` (after testing with Private)
- Redirect URL: `https://idx-api.quantyralabs.cc/oauth/leadconnector/callback`
- Installation URL: `https://idx-api.quantyralabs.cc/leadconnector/install`
- Scopes: All scopes listed above

### Phase 4: Webhooks
- Webhook URL: `https://idx-api.quantyralabs.cc/webhooks/leadconnector`
- Events: All 11 events toggled ON

### Phase 5: Pricing
- Pricing Model: `Free` (subscription handled externally via Stripe)
- Distribution: `Public`
- Target Audience: Real Estate Agents, Lenders, Agencies
- Geo: United States

### Phase 6: Review & Submit
1. Test OAuth flow with Private app setting
2. Verify all routes respond correctly
3. Test lead sync end-to-end
4. Switch to Public, submit for GHL review
5. Approval takes 1-5 business days

---

## Environment Variables

```bash
# GoHighLevel Marketplace App Configuration
GHL_CLIENT_ID=your-client-id-here
GHL_CLIENT_SECRET=your-client-secret-here
GHL_REDIRECT_URI=https://idx-api.quantyralabs.cc/oauth/leadconnector/callback
GHL_WEBHOOK_URI=https://idx-api.quantyralabs.cc/webhooks/leadconnector
GHL_INSTALLATION_URL=https://idx-api.quantyralabs.cc/leadconnector/install

# GHL API Configuration
GHL_API_BASE_URL=https://services.leadconnectorhq.com
GHL_API_VERSION=2021-07-28

# OAuth Scopes (Maximum - Full CRM Integration)
GHL_SCOPES="contacts.readonly contacts.write opportunities.readonly opportunities.write locations.readonly locations/tags.write users.readonly oauth.readonly oauth.write conversations.readonly conversations/message.write workflows.readonly calendars/events.write businesses.readonly forms.readonly links.write campaigns.readonly"

# Token Refresh Configuration
GHL_TOKEN_REFRESH_ENABLED=true
GHL_TOKEN_REFRESH_INTERVAL_HOURS=22

# Webhook Event Subscriptions
GHL_WEBHOOK_EVENTS="INSTALL UNINSTALL ContactCreate ContactUpdate ContactDelete ContactTagUpdate OpportunityCreate OpportunityStatusUpdate OpportunityStageUpdate NoteCreate TaskCreate"

# Stripe → GHL Sync Tags
GHL_SUBSCRIPTION_TAG_ACTIVE="quantyra-active"
GHL_SUBSCRIPTION_TAG_PAST_DUE="quantyra-past-due"
GHL_SUBSCRIPTION_TAG_CANCELLED="quantyra-cancelled"
GHL_SUBSCRIPTION_TAG_TRIAL="quantyra-trial"

# Audit Logging (Stellar MLS Compliance)
GHL_AUDIT_LOG_ENABLED=true
GHL_AUDIT_LOG_PATH=/var/log/geoidx/ghl_audit.log
```

---

## Deployment Checklist

1. Add GHL environment variables to Dokploy; configure **idx-api** image build with **project root** as context and Dockerfile **`Dockerfile.idx-api`** (see **`README.md`** at project root).
2. Deploy idx-api service
3. Run GHL database migrations: `php artisan migrate --path=database/migrations/ghl`
4. Seed GHL configuration: `php artisan db:seed --class=GhlConfigSeeder`
5. Verify routes respond
6. Start queue workers for ghl-sync, ghl-webhooks, ghl-maintenance queues
7. Configure scheduled token refresh cron
8. Create GHL Marketplace App (Private for testing)
9. Test OAuth flow end-to-end
10. Test lead sync
11. Switch app to Public, submit for review
12. Monitor webhooks, sync logs, audit logs post-approval

---

## Out of Scope

- Changes to `geo-web/` folder (unless explicitly told)
- Changes to `mobile/` folder (unless explicitly told)
- Changes to `docs/gohighlevel-oauth-documentation.md` (reference only)

---

## Next Steps

Invoke the `writing-plans` skill to create a detailed implementation plan for this design.