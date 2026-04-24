# Engagement & Adoption

Use when users have completed onboarding but haven't yet adopted key features like API tokens, lead sync, or full widget deployment.

## Patterns

### API Token Empty State
In the dashboard API tokens section, show "No tokens yet" with a lock icon when the user has zero Sanctum tokens. Include a primary "Create token" button and a secondary link to copy-paste examples for geo-web integration. Full-access users see `idx:full` ability highlighted; teaser-tier users see an upgrade nudge.

### Lead Sync Empty List
When `/api/leadconnector/leads` returns empty, display "No leads captured yet" with a code snippet showing the widget POST endpoint. Include a "Test lead" button that fires a sandbox request and refreshes the list, demonstrating the integration works without waiting for real traffic.

### Subscription Teaser Gate
For Pro-tier users approaching their domain limit, replace the "Add domain" button with a soft-gate empty state showing current usage (3/3 domains) and a comparison table highlighting Smart tier benefits. The primary CTA starts an upgrade checkout session.

## Warning

Avoid showing "0 leads" as a failure state. Users early in their lifecycle interpret empty lead lists as broken integration rather than normal startup state. Always pair empty lead lists with "Expected timeline" guidance and a direct link to test/sandbox tools.