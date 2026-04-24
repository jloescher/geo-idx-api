---
name: engineering-referral-loops
description: Designs referral or partner loop mechanics for marketplace integrations, OAuth flows, lead exchange, and embeddable widgets. Focuses on secure token exchange, webhook event handling, and audit logging patterns for third-party partnerships.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

```

# Engineering Referral Loops Skill

Designs secure referral and partner integration mechanics for marketplace apps and third-party ecosystems. Centers on token-based authentication, event-driven synchronization, and embeddable distribution surfaces with full audit trails.

## Quick Start

1. **Map the OAuth handshake**: Identify authorization code flow endpoints (`/oauth/{partner}/authorize`, `/oauth/{partner}/callback`) and token storage (encrypted at rest, hash-based lookup).
2. **Define partner identity**: Create database rows for partner tokens with `access_token_hash`, `expires_at`, `scopes`, and `user_type` (agency vs location).
3. **Register callback URLs**: Store allowed origins for embeds and webhooks; validate Origin/Referer headers against registered URLs.
4. **Implement webhook ingestion**: Create inbox table (`{partner}_webhook_events`), signature verification middleware, and async job dispatch.
5. **Add embed surfaces**: Build widget loader pattern with API-key validation, CORS handling, and cross-origin lead ingestion.
6. **Enable audit logging**: Dual-channel logging (database + file) capturing endpoint, latency, and compliance flags for every partner interaction.

## Key Concepts

| Concept | Implementation |
|---------|----------------|
| **Token Exchange** | OAuth 2.0 Authorization Code Grant with PKCE; refresh tokens stored encrypted; hash-based Bearer lookup without plaintext exposure. |
| **Partner Scopes** | Space-delimited permissions stored on token; middleware checks abilities like `partner:access` vs `partner:full`. |
| **Webhook Inbox** | Idempotent event storage with `webhook_id` deduplication; async processing via queued jobs; signature verification with configurable secrets. |
| **Embed Widgets** | API-key gated (`{prefix}_{hash}`), origin-validated, CORS-enabled surfaces; throttle-protected lead ingestion endpoints. |
| **Lead Sync** | Inbound leads captured to local table with `lead_type` mapping; dispatch sync jobs to push to partner CRM with retry logic. |
| **Generation-based Cache** | For shared resources, fingerprint upstream source metadata; increment generation on change to invalidate edge caches. |

## Common Patterns

**OAuth Install Flow**  
`install` landing → `authorize` redirect → `callback` exchange → `register-urls` origin collection → `installation-complete` embed instructions.

**Agency-to-Location Token Exchange**  
Company-level token calls partner's `/oauth/locationToken` endpoint to obtain location-scoped tokens; store both with parent-child relationship.

**Widget Middleware Chain**  
API key validation → Origin/Referer check against `registered_urls` → CORS header append → rate limit → lead creation → async sync job dispatch.

**Webhook Signature Verification**  
Raw body HMAC-SHA256 with configured secret; header name configurable (`X-{Partner}-Signature`); toggle for local development.

**Subscription Status Sync**  
Webhook handlers update local `subscription_status`; tag-based reconciliation with partner CRM; status values: `none`, `trial`, `active`, `past_due`, `cancelled`.

**Audit Trail**  
Every API call and webhook logged with `logged_at`, `endpoint`, `latency_ms`, `response_status`, `partner_id`, `is_data_access`, and `compliance_verified` flags.