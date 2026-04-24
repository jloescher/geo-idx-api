# Content Copy for Partner Integrations

## When to use
When writing user-facing text in OAuth screens, widget loaders, API error responses, or webhook documentation that partners will expose to their customers.

## Project-relevant patterns

**Ability-scoped error messages**
Return different 403 messages based on token abilities. For `idx:access` tokens hitting limits: "Upgrade for full MLS data." For missing tokens: "Register your domain to access listings." This aligns messaging with the authorization layer.

**Webhook event documentation**
Document webhook payloads with real examples showing `webhookId`, `event_type`, and normalized types (`INSTALL` vs `APPINSTALL`). Partners implement handlers faster when they see exact field shapes and deduplication strategies.

**Feature-gated feature descriptions**
In the subscription catalog (Pro/Smart/Ultra/Mega tiers), describe features by their gating mechanism: "3 domains with teaser gating" vs "Unlimited domains, 2M API calls/mo, developer keys." Make the restriction the feature description.

## Warning
Avoid exposing internal token hashes or raw database IDs in user-facing copy. The `access_token_hash` is for lookup only—never include it in widget config JSON or error responses, even in debug mode.