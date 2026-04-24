# Roadmap Experiments

## When to use

Evaluating bets that test new features with limited rollouts, A/B variants, or time-boxed pilots. Use when scoring initiatives for feature flags, gradual widget migrations, or tier-gated beta access that limits blast radius on the Bridge proxy or GHL Marketplace surfaces.

## Project-relevant patterns

**Domain-scoped feature flags**: The `domains` table (`domain_slug`, `is_active`) supports lightweight gating. Low-effort experiments add boolean columns (e.g., `gis_beta_enabled`) to enable parcel overlays for select subscribers without code deploys — risk is migration rollback if the experiment fails.

**Widget variant testing**: The `/widget/loader.js` route accepts `data-widget` parameters. Medium-impact bets serve different loader scripts or HTML shells (`/widget/search/{apiKey}` variants) based on `ghl_widget_configs.widget_theme`, though cache invalidation in `GisCache` adds effort.

**Token ability expansion**: Sanctum tokens support abilities (`idx:access`, `idx:full`). Low-risk experiments grant new abilities to limited token sets (e.g., `idx:gis-beta`) before full rollout — but require `DomainOrTokenAuth` middleware updates and careful handling of `teaser` vs `full` access implications.

## Warning

Avoid experiments that modify `ghl_oauth_tokens` encryption or `bridge_proxy_audit_logs` schema. These are compliance-critical tables with Stellar MLS obligations — failed experiments here risk audit trail gaps or token exposure that cannot be rolled back cleanly.