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

- Keys support scopes: `monitor` and `content`.
- The plaintext secret is only shown once at creation time.
- Keys can be revoked instantly.

The MCP server validates keys on every tool call and records last-used timestamps.

## Available Tools (current)

**Monitoring & Ops**
- `ping`
- `get_monitoring_snapshot`
- `get_queue_state`
- `get_gis_source_health`
- `inspect_job`

**Comps / Valuation (for Grok Connectors)**
- `run_comps` — Full access to the Quantyra IDX comps/BPO engine.
- `get_comps_analysis_guide` — Rich guidance on how to best use `run_comps`.
- `suggest_comps_subject` — Given an address + basic details, returns a ready-to-use SubjectInput object.
- `validate_comps_subject`, `explain_comps_adjustments`, `estimate_value_range_from_subject`

**Content Generation Tools** (require `content` scope)
- `search_listings_for_content` — Safe, limited-field search over listings for blogs/reports.
- `query_gis_parcels_for_content` — Safe GIS data for neighborhood/market content.

To use comps tools, the MCP key must have the `comps` scope. For content tools, use the `content` scope.

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
   - Redirect URIs: `https://grok.x.ai` (or the exact redirect URI Grok provides)

   You can manage all OAuth clients (list, create, revoke) from this admin page.

2. In Grok Web, go to **Settings → Connectors → New Connector → Custom**.

3. Enter your public MCP URL:
   ```
   https://your-mcp-domain.com/mcp
   ```
   (This must be served over HTTPS.)

4. Grok will redirect you to our `/oauth/authorize` endpoint.
   - Log in with your Quantyra dashboard account.
   - Select which of your MCP keys you want to grant access to.
   - Approve the request.

5. Grok will receive an access token and can now call the MCP tools.

This flow is powered by the dual-auth system in the MCP server: direct `mcp_...` bearer tokens continue to work for local/CLI use, while Grok Web uses short-lived OAuth access tokens.

**Important**: The remote MCP endpoint must be publicly reachable over HTTPS. Plain HTTP is rejected for the OAuth flow.

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
- All tools enforce scope checks (`monitor`, `comps`, `content`).
- Content tools use strict field projection and are explicitly documented as "research only".
- No write paths exist in the current tool set.
- Timeouts and result limits are applied on queries.
- All tool calls are traceable via the `mcp_keys.last_used_at` column.

## Related

- Admin dashboard monitoring: `docs/admin-dashboard.md`
- Production data safety rules: `AGENTS.md`
