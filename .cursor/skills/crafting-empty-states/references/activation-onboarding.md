# Activation & Onboarding

Use when designing first-run experiences for GHL Marketplace installs, Stripe checkout flows, and widget configuration wizards where users must complete steps to unlock value.

## Patterns

### GHL OAuth Install Flow
The `/leadconnector/install` page uses a step-indicator pattern: (1) Connect GHL → (2) Register URLs → (3) Complete. Each step shows a contextual empty state explaining why the step matters for MLS compliance, with the primary action button disabled until prerequisites are met.

### Widget Setup Empty State
Before a user registers their first domain URL in `/leadconnector/register-urls`, display an empty preview area with a headline "Preview your widget" and body text explaining that MLS data requires approved domains. The CTA links to the registration form with a secondary link to documentation.

### Post-Checkout Activation
After Stripe checkout, the dashboard shows a "Welcome to [Plan Name]" empty state with a quick-setup checklist (create API token, configure first widget, test lead capture). Completed items show checkmarks; pending items have direct links.

## Warning

Do not auto-redirect users through the OAuth flow without showing the landing state first. GHL app installs that immediately redirect to `chooselocation` cause users to lose context and abandon at higher rates. Always show value proposition first, then trigger OAuth via explicit CTA.