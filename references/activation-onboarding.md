# Activation & Onboarding

When to use: Designing multi-step flows that convert new GHL Marketplace installs into fully activated widget subscribers.

## Project-Relevant Patterns

### OAuth-to-Widget Completion Funnel
The GHL onboarding uses session-state bridging across 4 steps: install landing → OAuth authorize → OAuth callback (stores `ghl_pending_oauth_token_id`) → URL registration → installation complete. Each step validates prerequisites via middleware (`GhlPendingOAuthToken`) and redirects back on failure.

### Progressive Form Disclosure
The URL registration wizard at `/leadconnector/register-urls` uses a single-page Livewire pattern with conditional sections: primary HTTPS URL (required), additional origins (collapsible), MLS acknowledgment checkbox (gated). This reduces cognitive load while enforcing compliance requirements.

### Activation Checklist Model
`GhlInstalledLocation` tracks completion state with boolean flags: `oauth_token_valid`, `urls_registered_count`, `widget_embedded`, `subscription_status`. The subscriber dashboard queries these to render a visual checklist with "completed / in-progress / pending" states.

## Pitfall
Avoid marking onboarding complete after OAuth alone. The critical activation metric is widget embed (script loaded on registered origin) — many users authorize but never complete URL registration. Track `widget_first_load_at` timestamp separately from `oauth_callback_at`.
