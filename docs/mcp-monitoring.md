# MCP Monitor for AI Agents

The `cmd/mcp-monitor` binary exposes a Model Context Protocol (MCP) server that gives AI agents (Claude, Cursor, etc.) structured, live access to the rich operational monitoring data of the Quantyra IDX API.

## Why this exists

Instead of the agent having to ask a human to copy-paste dashboard JSON or logs, the agent can directly call high-signal tools like:

- `get_monitoring_snapshot`
- `get_queue_state`
- `get_gis_source_health`
- `inspect_job`
- etc.

## Running the MCP locally (recommended for agents)

```bash
# 1. Build
go build -o mcp-monitor ./cmd/mcp-monitor

# 2. Provide credentials (use a dedicated read-only user when possible)
export APP_ENV=local
export DB_RW_DSN="postgres://mcp_monitor:strongpass@127.0.0.1:5432/geoidxapi?sslmode=disable&application_name=idx-mcp-monitor"

# 3. Run (stdio transport)
./mcp-monitor
```

Point your AI coding agent at this binary as an MCP server (stdio).

## Security & Least Privilege

**Strongly recommended**: Create a dedicated read-only PostgreSQL user for the MCP monitor.

See the ready-to-use script: `docs/scripts/create_mcp_monitor_user.sql`

This user is granted `CONNECT` + `SELECT` only on the tables the MCP actually queries (jobs, replica_pages, gis_*, listings, audit logs, etc.). No write access, no ability to enqueue jobs or mutate state.

### Recommended DSN for production
```bash
export DB_RW_DSN="postgres://mcp_monitor:CHANGE_ME@your-patroni-primary:5432/geoidxapi?sslmode=require&application_name=idx-mcp-monitor"
```

### Why this matters
- The MCP server is intentionally read-only.
- Using a limited user reduces blast radius if the MCP process or the AI agent is ever compromised.
- Follows the same principle used for other read-heavy internal services.

Update the script with your actual database name, strong password, and any additional tables you want the MCP to be able to read in the future.

## Admin: Managing MCP Keys

Admins can create and revoke keys from the dashboard at `/dashboard/mcp-keys`.

- Keys support scopes: `monitor`, `comps`, `content`, and `api`.
- The plaintext secret is only shown once at creation time.
- Keys can be revoked instantly.

The MCP server validates keys on every tool call and records last-used timestamps.

## Available Tools (current)

**Monitoring & Ops** (`monitor` scope)
- `ping` — auth health check (OAuth or MCP key)
- `get_monitoring_summary` — lightweight incidents + queue totals (prefer over snapshot in agent loops)
- `get_monitoring_snapshot` — full operational snapshot
- `get_queue_state`
- `get_gis_source_health`
- `inspect_job`

**Comps / Valuation** (`comps` scope)
- `run_comps` — Full access to the Quantyra IDX comps/BPO engine.
- `get_comps_analysis_guide`, `suggest_comps_subject`, `validate_comps_subject`, `explain_comps_adjustments`, `estimate_value_range_from_subject`

**Content** (`content` scope)
- `search_listings_for_content` — PostGIS mirror search with public IDX visibility
- `query_gis_parcels_for_content` — placeholder for content-safe GIS

**API parity** (`api` scope; requires `MCP_API_*` env for live/RESO routes)
- `search_listings`, `search_listings_live`, `get_listing`, `get_property`
- `query_gis`, `autocomplete_cities`, `autocomplete_counties`
- RESO/stats proxy tools: `list_properties`, `list_listings`, `lookup`, `list_agents`, `get_agent`, `list_offices`, `list_openhouses`, `list_members`, `list_reso_offices`, `list_reso_openhouses`, `get_bridge_stats`, `list_pub_parcels`, `list_pub_assessments`, `list_pub_transactions`

### OAuth clients: no `mcp_key` in tool arguments

When connected via Grok Web OAuth, **do not pass `mcp_key`** in tool calls. Authorization comes from the `Authorization: Bearer <oauth_access_token>` header established at connect time. Scopes are enforced from the OAuth token and the auto-provisioned MCP key created at token exchange.

### Scope tiers

| Scope | Tools |
|-------|-------|
| `monitor` | Monitoring snapshot, queue, GIS health, inspect |
| `comps` | `run_comps` and comps helpers |
| `content` | `search_listings`, `search_listings_for_content`, `get_listing` (non-expanded mirror JSON only) |
| `api` | All content tools plus live search, expanded listing, RESO/GIS proxy (requires `MCP_API_*` env) |

Tool responses include `effective_scopes` (union of OAuth token scopes and granted key scopes).

Stdio/local clients may still pass `mcp_key` per tool call or use `Bearer mcp_...` on HTTP.

More tools will be added over time.

## Using with Grok Connectors

You have two main ways to connect the Quantyra IDX MCP to Grok:

### Option 1: Local stdio (Recommended for most development)
Run the binary locally with a key:

```bash
export MCP_KEY="mcp_xxxxxxxxxxxxxxxx"
./mcp-monitor
```

Then add it as a stdio MCP server in Grok (or via `.grok/mcp.toml` for project-scoped use).

### Option 2: Remote via OAuth (Best for Grok Web Custom Connectors)
The production `idx-api-mcp` service (deployed in Coolify) now supports a full OAuth 2.1 Authorization Code + PKCE flow.

This allows you to add it as a **Custom MCP Connector** directly in Grok Web without manually managing bearer tokens.

#### Steps to connect in Grok Web

1. As an admin, register a client at **`/dashboard/oauth-clients`**:
   - Name: `Grok Web`
   - Client ID: `grok-web`
   - Redirect URIs (one per line, **exact** match — include every URI Grok may send):
     ```
     https://grok.com
     https://grok.x.ai
     https://grok.com/api/mcp/auth_callback
     https://grok.x.ai/api/mcp/auth_callback
     https://grok.com/connector/oauth-exchange-code
     https://grok.x.ai/connector/oauth-exchange-code
     https://grok.com/connect/oauth-exchange-code
     https://grok.x.ai/connect/oauth-exchange-code
     https://grok.com/connectors-oauth-exchange-code
     https://grok.x.ai/connectors-oauth-exchange-code
     ```
     Grok has used **three different** callback shapes (not interchangeable): `/connect/…`, `/connector/…`, and hyphenated `/connectors-oauth-exchange-code`. Register each plus `www` host variants if needed.

     **idx-api-web** also accepts known Grok callback paths for `client_id=grok-web` on `grok.com` / `grok.x.ai` hosts (see `grokWebRedirectURIAllowed` in `internal/handler/oauth/helpers.go`) so minor path changes do not require an emergency DB edit.

     If you see `redirect_uri is not allowed for this client`, use **`received_redirect_uri`** from the JSON error. Trailing slashes on a registered URI are normalized automatically.

   You can manage all OAuth clients (list, create, revoke) from this admin page.

2. In Grok Web, go to **Settings → Connectors → New Connector → Custom**.

3. Enter your public MCP URL:
   ```
   https://your-mcp-domain.com/mcp
   ```
   (This must be served over HTTPS.)

4. Grok will redirect you to our `/oauth/authorize` endpoint.
   - Log in with your Quantyra dashboard account.
   - Review requested scopes and approve (an MCP key is created automatically for this OAuth client).

5. Grok receives access + refresh tokens and can call MCP tools **without passing `mcp_key` in tool arguments**.

   - Access tokens default to **24h** (`OAUTH_ACCESS_TOKEN_TTL`).
   - Refresh tokens default to **30 days** (`OAUTH_REFRESH_TOKEN_TTL`) with rotation on use.
   - Re-authorize only when refresh fails or you need scope changes.

This flow is powered by the dual-auth system in the MCP server: direct `mcp_...` bearer tokens continue to work for local/CLI use, while Grok Web uses OAuth access tokens.

**Important**: The remote MCP endpoint must be publicly reachable over HTTPS. Plain HTTP is rejected for the OAuth flow.

#### Production Gotchas & Live Debugging Tools (Coolify + Traefik)

When running the remote MCP behind Coolify + Traefik + Cloudflare, several subtle issues can break RFC 9728 Protected Resource Metadata (PRM) discovery that Grok Web (and other OAuth clients) rely on:

- The 401 challenge response from `/mcp` must include a valid `WWW-Authenticate: Bearer resource_metadata="..."` header pointing at a working `/.well-known/oauth-protected-resource` document.
- Early versions of the dual-auth layer emitted a broken link (`https://.well-known/...`) even when `MCP_PUBLIC_URL` was correctly set. This completely blocked Grok Web Custom Connector flows while raw `mcp_...` keys continued to work.

**What we implemented to make the OAuth path reliable:**

- `buildResourceMetadataURL()` helper in `cmd/mcp-monitor/main.go` that strongly prefers `MCP_PUBLIC_URL`, normalizes it correctly, and respects `X-Forwarded-Proto` / `X-Forwarded-Host`.
- A production safety net: when all header/env sources produce an empty host, it falls back to the known public URL for this service so the broken link can never appear again.
- Startup diagnostic logging (visible in container logs) that prints the exact `MCP_PUBLIC_URL` / `OAUTH_AUTH_SERVER` values the Go process sees at HTTP startup, plus an example of what the helper currently produces.
- Live unauthenticated debug endpoint:

  ```
  GET https://mcp.quantyralabs.cc/debug/oauth-config
  ```

  Returns JSON showing:
  - `process_mcp_public_url` and `process_oauth_auth_server` as seen by `os.Getenv`
  - `produced_resource_metadata_url` (what `buildResourceMetadataURL` actually returns right now)
  - A note explaining what a mismatch means

**Observed gotcha (June 2026):** The broken URL (`https://.well-known/oauth-protected-resource`) was traced to a helper bug in `buildResourceMetadataURL()`: it used `strings.Index("/mcp")`, which matched the host segment in `https://mcp...` and truncated the base to `https:/`. The fix now uses `net/url.Parse` and derives `scheme://host` safely.

**What to expect now:** If `MCP_PUBLIC_URL=https://mcp.quantyralabs.cc/mcp`, then `/debug/oauth-config` should report:
- `produced_resource_metadata_url=https://mcp.quantyralabs.cc/.well-known/oauth-protected-resource`

If it does not, the running container likely has an old image or stale deployment, not just a header/env mismatch.

**Required environment variables for the `idx-api-mcp` Coolify app (OAuth flow):**

```env
MCP_PUBLIC_URL=https://mcp.quantyralabs.cc/mcp
OAUTH_AUTH_SERVER=https://idx.quantyralabs.cc
MCP_HTTP_ENABLED=true
MCP_API_INTERNAL_URL=https://idx.quantyralabs.cc
MCP_API_DOMAIN_SLUG=your-domain-slug
MCP_API_SERVICE_TOKEN=<idx:full PAT plaintext>
OAUTH_ACCESS_TOKEN_TTL=24h
OAUTH_REFRESH_TOKEN_TTL=720h
```

Interim note: until you deploy auto-key OAuth, you can manually add `api` to an existing MCP key and select it on the legacy consent flow. After deploy, re-authorize Grok once; keys are provisioned automatically.

These must be present at **runtime** (not just build time). The dual-auth layer uses them to construct correct PRM links and to route unauthenticated requests to the OAuth consent flow.

#### MCP session stickiness (multi-origin)

Streamable HTTP MCP sessions are **in-memory per process**. If `initialize` hits server A and `tools/call` hits server B, Grok sees `Invalid session ID`.

**Cloudflare session affinity (`__cflb`) requires a paid Load Balancer** (Traffic → Load Balancing, proxied orange-cloud pool). It does **not** apply to:

- DNS-only (gray-cloud) records
- Multiple **A records** on one hostname without Load Balancing (Cloudflare or client DNS round-robin between origins)

**Recommended without Cloudflare Load Balancing (two servers, two A records):**

| Approach | Setup |
|----------|--------|
| **Single MCP origin (simplest)** | Run **one** idx-api-mcp container (one Coolify app, replica count 1). Point `mcp.quantyralabs.cc` to **one** A record / one server IP. Keep two A records for `idx.quantyralabs.cc` API if you want — MCP does not need the same multi-origin layout. |
| **One hostname per server** | e.g. `mcp-nyc.quantyralabs.cc` → NYC IP only, `mcp-atl.quantyralabs.cc` → ATL IP only. Set `MCP_PUBLIC_URL` to the hostname Grok actually uses (single URL). Do not round-robin two IPs on the same MCP hostname. |
| **Paid option later** | Cloudflare Load Balancer with Session Affinity on `/mcp*` if you need one hostname and multiple MCP replicas. |

Traefik on each Coolify host only sticks sessions **within that host**. Two public IPs mean two independent Traefiks — DNS must send a client to the **same** IP for the whole Grok session (fragile with round-robin A records).

Smoke test after connect:

1. `initialize` → capture `Mcp-Session-Id` response header
2. `tools/call` with the same session header and Bearer token
3. Expect tool results, not session errors

See also the troubleshooting notes in the commit history around `df10eb9` and `8051a9f` for the earlier diagnosis timeline and follow-up fix.

#### Post-consent failures (PRM OK, Grok still says “Unable to authorize app”)

If `/debug/oauth-config` shows a correct `produced_resource_metadata_url` and Grok reaches the Quantyra consent screen, but authorization fails **after** you click **Authorize**, the problem is usually on **`idx.quantyralabs.cc`** (idx-api-web), not idx-api-mcp.

**Symptom:** Grok rejects the redirect because `state` is missing or empty on the callback URL.

**Check:**

1. After consent, inspect the browser redirect to Grok’s `redirect_uri`. The query string must include **both** `code=...` and a **non-empty** `state=...` matching what Grok sent on the initial `/oauth/authorize` request.
2. Grep idx-api-web logs for `POST /oauth/token` — look for `invalid_grant`, `client_id mismatch`, `redirect_uri mismatch`, or `failed to store access token` (JSONB insert issues on `granted_mcp_key_ids`).
3. Confirm the OAuth client’s registered redirect URIs use an **exact** match (not prefix-only) for the `redirect_uri` Grok sends.
4. Grok Web uses PKCE **S256**; `code_challenge` is required on authorize and `code_verifier` on token exchange.

**OAuth routes:** `GET/POST /oauth/authorize`, `POST /oauth/token` on **idx-api-web** (`OAUTH_AUTH_SERVER` / `IDX_PLATFORM_URL`). Deploy idx-api-web NYC + ATL after OAuth handler changes; idx-api-mcp alone is not sufficient.

### Example System Prompt for Grok when using Comps

```
You have access to a high-quality residential comps and BPO engine via the run_comps tool.

When a user asks for comparable sales or a value opinion:
1. First call get_comps_analysis_guide if you are unsure about modes or best practices.
2. Use suggest_comps_subject when the user only gives an address.
3. Call run_comps with the best possible SubjectInput you can construct.
4. Always present both the raw data and the human-readable summary from the tool.
5. Clearly state assumptions and suggest what additional property details would improve the analysis.
```

This pattern produces excellent results while keeping full MCP power available.

## Human-Friendly Output

All tools return a consistent envelope:

```json
{
  "generated_at": "...",
  "key_name": "Claude Content Agent",
  "oauth_client_id": "grok-web",
  "granted_scopes": ["comps", "content", "monitor"],
  "notes": "Human-readable explanation of what this data means and suggested next steps.",
  "data": { ... }
}
```

## Development Notes

- The monitor server lives in `internal/mcp/monitor/`.
- It reuses the existing `MonitoringService` and `MonitoringRepo` heavily.
- Authentication is performed against the `mcp_keys` table.
- The binary is intentionally read-only.

## Deployment

### Local / Agent Use (Recommended for most teams)
Run the binary locally on the machine where your AI agent runs (via stdio). This is the simplest and most secure pattern.

### Coolify Deployment Story (as an internal service)

You can deploy `mcp-monitor` as its own Coolify application (similar to how the main API, worker, and scheduler are deployed).

Recommended topology:
- Deploy as a lightweight internal service (not exposed publicly).
- Use the dedicated `mcp_monitor` read-only database user (see script above).
- Connect via the internal Docker network or Tailscale.
- Expose via stdio for local agents or as a private SSE endpoint (with strong auth) for remote agents.

Example Coolify environment variables for the MCP service:
```bash
MCP_KEY=          # optional - can be left empty if keys are passed per-call
APP_ENV=production
DB_RW_DSN=postgres://mcp_monitor:xxx@patroni-primary:5432/geoidxapi?sslmode=require&application_name=idx-mcp-monitor
```

Security recommendations when deployed:
- Never expose the MCP port publicly.
- Use Coolify internal networking + Tailscale where possible.
- Consider adding a simple bearer or mTLS layer if exposing over the network.
- Monitor usage via the `last_used_at` timestamps on `mcp_keys`.

See also: `docs/coolify-deployment.md` for patterns used by the main services.

### Production Safety Checklist
- [ ] Using dedicated read-only `mcp_monitor` DB user
- [ ] MCP binary not publicly reachable
- [ ] Keys have the minimum required scopes
- [ ] `last_used_at` monitoring in place
- [ ] Regular key rotation policy for long-lived agent keys

### Safety Guardrails Built Into the MCP Server
- All tools enforce scope checks (`monitor`, `comps`, `content`, `api`).
- Content tools use strict field projection and are explicitly documented as "research only".
- No write paths exist in the current tool set.
- Timeouts and result limits are applied on queries.
- All tool calls are traceable via the `mcp_keys.last_used_at` column.

## Related

- Admin dashboard monitoring: `docs/admin-dashboard.md`
- Production data safety rules: `AGENTS.md`
