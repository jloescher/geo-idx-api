# Measurement and Testing for Marketplace Integrations

## When to use
When verifying webhook handlers, token refresh logic, or caching behavior; when instrumenting partner flows for compliance and debugging.

## Project-relevant patterns

**Dual-channel audit logging**
Log every GHL API call and webhook to both database (`ghl_audit_logs`) and optional file (`ghl_audit.log`). Include `latency_ms`, `response_status`, `is_mls_data_access`, and `compliance_verified` flags for post-hoc analysis.

**Webhook signature testing**
Use configurable signature verification (`GHL_WEBHOOK_REQUIRE_SIGNATURE=false` in local) to test handlers without valid signatures. In tests, dispatch `ProcessGhlWebhookJob` directly to verify handler logic separate from signature middleware.

**Teaser enforcement testing**
Assert that non-`idx:full` responses contain ≤3 items in list-shaped JSON. Cache tests should verify that teaser truncation happens *after* cache retrieval so stored bytes remain canonical full payloads.

## Warning
Don't test against real OAuth providers in CI. The GHL token refresh tests should mock `services.leadconnectorhq.com` responses—actual token exchanges during tests will fail with revoked credentials and pollute the token table with invalid rows.