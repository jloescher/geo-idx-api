# Engagement & Adoption

When to use: Driving ongoing usage after initial setup through progressive feature discovery, usage-based nudges, and value reinforcement. Use for subscription tiers, API usage dashboards, and widget integration verification.

## Patterns

**Progressive Feature Gates by Subscription Tier**
Dashboard controller checks `subscription_status` on `ghl_installed_locations` and conditionally renders upgrade CTAs for tier-locked features. Pro ($39) sees "Upgrade to Smart" for phone OTP; Smart ($79) sees "Upgrade to Ultra" for developer keys. Teaser listings (3 items) in Bridge proxy serve as soft gates—full access requires active subscription.

**Usage Dashboard with Actionable Context**
Stats endpoint (`/api/leadconnector/stats`) returns `mls_request_count` and `lead_count` with comparison to plan limits. Display visual progress bars toward tier thresholds with inline "View usage history" link. Include widget install verification—check Origin header matches registered URLs and surface "Verify installation" alert if no leads received within 48 hours.

**Re-engagement via Webhook Lifecycle**
GHL webhooks trigger `ProcessGhlWebhookJob` for `INSTALL` events to send welcome emails with next-step checklist. `UNINSTALL` events trigger data retention notices and optional exit survey. Hourly token refresh job surfaces "Reconnection required" dashboard banner when refresh fails, linking directly to `/oauth/leadconnector/authorize`.

## Warning

Avoid nagging users with incomplete checklist items for features they've intentionally skipped. Respect `onboarding_completed` flag—once set, shift from blocking modals to passive dashboard cards. Aggressive interstitials after activation reduce GHL marketplace app ratings.