# Activation & Onboarding

## When to use

Evaluating initiatives that affect the GHL Marketplace install flow (`/leadconnector/install`), OAuth callback completion, or subscriber dashboard first-run experience. Use this when scoring bets related to reducing drop-off between install and first MLS data access.

## Project-relevant patterns

**OAuth completion funnel**: The GHL Marketplace flow spans `routes/ghl-web.php` — from `/leadconnector/install` through `/oauth/leadconnector/callback` to `/leadconnector/register-urls`. Impact scoring should measure completion rate at each step, especially the MLS domain registration form that gates widget API key issuance.

**Dashboard activation**: The Livewire `DashboardController` loads subscription status from Cashier and API tokens from Sanctum. Effort estimates for onboarding bets should account for the `ghl_installed_locations` + `ghl_registered_urls` data model when adding progress checklists or guided tours.

**Widget first-render**: The `/widget/loader.js` route (`routes/ghl-widget.php`) is the first impression for embedded users. Risk assessment should consider that widget failures block lead capture — a high-revenue surface — even when the main dashboard is functional.

## Warning

Don't conflate GHL OAuth completion with actual MLS data usage. A location may complete OAuth (token in `ghl_oauth_tokens`) but never register URLs or embed widgets. Measure activation by `ghl_registered_urls.widget_api_key` issuance, not just token persistence.