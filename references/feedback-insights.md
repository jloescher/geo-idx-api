# Feedback & Insights

When to use: Collecting qualitative input from GHL subscribers and identifying friction points in the installation flow.

## Project-Relevant Patterns

### Installation Completion Survey
After `/leadconnector/installation-complete`, optionally redirect to a short 3-question form: "How easy was the installation?" (1-5), "What almost stopped you?" (text), "Which GHL feature do you use most?" (select). Store in `ghl_installation_feedback` table with `location_id` foreign key for follow-up.

### Error Path Logging
When OAuth callbacks fail (invalid state, token exchange error), log to `ghl_oauth_errors` with full context: `error_type`, `request_params`, `user_agent`, `ip_hash`. Aggregate weekly to identify documentation gaps or GHL API changes breaking the flow.

### Support Context Enrichment
Include `location_id` and `widget_api_key` in all dashboard error messages. When users screenshot errors, this context lets support cross-reference with `ghl_audit_logs` and `bridge_proxy_audit_logs` for the trailing 7 days without requiring manual lookup.

## Pitfall
Do not block the installation flow on feedback collection. The survey must be optional and post-completion only. Forcing feedback before OAuth authorization causes significant drop-off in the already-high-friction GHL Marketplace install funnel.
