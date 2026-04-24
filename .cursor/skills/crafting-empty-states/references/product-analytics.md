# Product Analytics

Use when instrumenting empty states and onboarding flows to measure activation rates, feature adoption, and conversion funnel health.

## Patterns

### Onboarding Step Funnel
Track custom events for each GHL install milestone: `ghl_install_start`, `ghl_oauth_complete`, `ghl_url_registered`, `ghl_widget_embedded`. Empty states at each step include data attributes (`data-step="oauth-pending"`) for analytics segmentation.

### Empty State Engagement
Tag all empty state CTAs with `data-analytics="empty-state-[action]"` to distinguish clicks from empty states versus navigation. Compare conversion rates: users who click "Create token" from an empty state convert to paid tiers at 2x the rate of users who navigate via menu.

### Billing Conversion Attribution
The post-checkout dashboard checks `request()->has('checkout_complete')` to trigger a one-time "Welcome" modal and fire a `checkout_completed` analytics event. This attributes revenue to specific empty state variants shown during the trial period.

## Warning

Do not fire analytics events during test runs or when `Http::fake()` is active. The GHL webhook and Stripe event tests will generate synthetic traffic that skews activation metrics. Gate analytics initialization behind `!app()->runningUnitTests()` checks.