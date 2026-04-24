# Feedback & Insights

Use when collecting qualitative feedback from users who abandon flows or encounter repeated empty states.

## Patterns

### Abandonment Micro-survey
If a user visits `/leadconnector/register-urls` but does not submit the form within 5 minutes (tracked via session timestamp), show a non-blocking banner: "Stuck? Tell us what's blocking your setup." Submission goes to `ghl_audit_logs` with event_type `setup_feedback` for support team review.

### Empty State NPS
After 30 days post-install, users with zero `quantyra_leads` see a dashboard empty state with an embedded NPS question: "How easy was it to set up your IDX widget?" Scores 0-6 trigger a follow-up "What went wrong?" text area; scores 9-10 trigger a "Write a review" CTA.

### Support Ticket Integration
In widget configuration empty states, include a "Contact support" secondary CTA that pre-fills a ticket with context: current `ghl_location_id`, widget API key prefix, and last 3 webhook delivery statuses from `ghl_webhook_events`. This reduces back-and-forth in support threads.

## Warning

Feedback prompts must respect GHL's rate limits and user privacy. Do not send automated emails to GHL location users based on empty state analytics without explicit opt-in. Use the webhook inbox and in-app banners only; email integration requires separate GHL permissions and unsubscribe handling.