# OAuth Auto-Key Design (2026-06-01)

Design spec for auto-provisioning MCP keys on OAuth completion, refresh tokens, consent UI fixes, and related MCP production fixes.

## Goals

1. **Auto-key on OAuth token exchange** — create or update one MCP key per `(user, oauth_client)` with requested scopes.
2. **Refresh tokens** — 24h access TTL, 30d refresh with rotation.
3. **Consent UX** — scope-only consent, dark-theme contrast, `api` scope visible.
4. **MCP parity** — `get_listing` content fallback, `run_comps` listing alias, effective scopes in tool envelope.

## Auto-key flow

1. User completes OAuth consent (scopes only, no key pickers).
2. On `POST /oauth/token` with `authorization_code`, after PKCE validation:
   - Parse and validate scopes against allowlist: `monitor`, `comps`, `content`, `api`.
   - `FindOrCreateOAuthKey(userID, clientID, clientName, scopes)` → key ID.
   - Store access token with `granted_mcp_key_ids = [keyID]`.
3. Re-auth updates scopes in place on the existing active key (partial unique index).

Key name: `OAuth: {client.Name}`. Plaintext key is never shown to the user.

## Refresh token model

- Table `oauth_refresh_tokens` with hashed token, client_id, user_id, scope, granted_mcp_key_ids, expires_at, revoked_at.
- `grant_type=refresh_token` rotates refresh token and issues new access token.
- Admin revoke of access tokens also revokes refresh tokens for that client/user.

Env: `OAUTH_ACCESS_TOKEN_TTL` (default 24h), `OAUTH_REFRESH_TOKEN_TTL` (default 30d).

## Consent UI

- Use `LoginPage` pattern: `.center-page` + `.card`, no light-theme override.
- Scope bullets include `api`.
- Remove manual MCP key checkbox section.

## Session stickiness

- MCP sessions are in-memory; multi-origin routing without stickiness breaks Grok (`Invalid session ID`).
- Cloudflare cookie affinity requires **paid Load Balancing** — not available with DNS-only or multiple A records alone.
- **Without CF LB:** single idx-api-mcp origin (one A record for `mcp.*`) or one hostname per server; see `docs/mcp-monitoring.md`.
- No PostgreSQL session store in v1.

## Migration (`00015_oauth_auto_key.sql`)

- `mcp_keys.oauth_client_id` FK to `oauth_clients(client_id)`.
- Partial unique index on `(created_by_user_id, oauth_client_id) WHERE revoked_at IS NULL AND oauth_client_id IS NOT NULL`.
- `oauth_refresh_tokens` table.

## Testing checklist

- First OAuth authorize creates auto key with requested scopes.
- Re-auth updates scopes in place (same key ID).
- Refresh token grant works without re-consent.
- `get_listing` with content-only scope returns non-expanded JSON.
- `run_comps` accepts `subject.type=listing`.
- Tool envelope includes `effective_scopes`.
