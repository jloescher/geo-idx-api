# In-App Guidance

## When to use

Evaluating bets for contextual help, setup wizards, or proactive nudges within the IDX platform or GHL install flow. Use when scoring initiatives that surface documentation, validate configuration, or guide users through MLS compliance requirements.

## Project-relevant patterns

**Install-time validation**: The `/leadconnector/register-urls` form enforces HTTPS and MLS agreement acknowledgment. High-impact guidance bets add real-time URL validation or GHL location ID lookups via `LocationTokenService` — but risk OAuth session expiry if the flow extends too long.

**Dashboard configuration hints**: The subscriber dashboard (`DashboardController`) surfaces subscription status and API tokens. Medium-effort guidance adds missing-configuration banners (e.g., "Add your first domain" when `domains` table is empty) without requiring new migrations.

**Widget embed instructions**: The `/leadconnector/installation-complete` view shows embed snippets. Low-effort, high-impact bets include copy-to-clipboard helpers or domain-specific snippet generators that reference `IDX_PLATFORM_URL/embed/{locationId}` redirects.

## Warning

Resist building heavy in-app documentation systems. The project already maintains 13+ docs in `docs/` — betting on embedded guidance should leverage these existing files (e.g., linking to `docs/ghl-environment-variables.md`) rather than duplicating content in Blade templates that drifts out of sync.