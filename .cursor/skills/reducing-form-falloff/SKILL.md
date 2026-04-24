---
name: reducing-form-falloff
description: Improves lead capture forms to reduce drop-off
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Reducing Form Falloff Skill

This skill improves lead capture form completion rates by implementing progressive gating, streamlined widget embeds, and optimized cross-origin form handling within the Quantyra GHL widget system.

## Quick Start

1. Review widget routes: `routes/ghl-widget.php`
2. Check gating config: `app/Ghl/Widgets/` and `ghl_widget_configs` table
3. Inspect lead form views: `resources/views/ghl/widgets/`
4. Examine middleware chain: `app/Ghl/Widgets/Middleware/`
5. Test lead capture: POST `/widget/api/leads` with valid `api_key` and matching Origin

## Key Concepts

- **Progressive Gating**: Configure `gate_after_views` in `ghl_widget_configs` to delay lead capture requirements until users have engaged with listings
- **3-Phase Middleware**: Widget requests flow through API key validation → Origin/Referer validation against `ghl_registered_urls` → CORS header append
- **CORS Preflight**: `OPTIONS /widget/api/leads` validates registered origins before allowing POST
- **Widget API Keys**: Public `qh_*` keys (stored in `ghl_registered_urls.widget_api_key`) scope requests to locations without exposing OAuth tokens
- **Lead Type Mapping**: `GhlLeadMapping` config routes `quantyra_leads` to GHL contacts/opportunities/tags based on `lead_type`

## Common Patterns

### Implement Progressive Gating
Edit `ghl_widget_configs.gate_after_views` to show ungated MLS content before requiring lead form completion; pair with `require_otp` for phone verification gating.

### Reduce Form Fields
Minimize required fields in `QuantyraLead` payload; use GHL enrichment to backfill contact details after initial email capture.

### Fix Origin Mismatch Errors
Ensure `ghl_registered_urls.primary_url` and `additional_urls` include all variants (www/non-www, http/https) to prevent CORS preflight failures that block submissions.

### Add Form Auto-Save
Implement partial lead persistence in widget JS to recover abandoned forms when users return to the registered origin.

### Track Drop-off Points
Log form step completions to `ghl_audit_logs` with `is_mls_data_access=false` to identify where users abandon the funnel.