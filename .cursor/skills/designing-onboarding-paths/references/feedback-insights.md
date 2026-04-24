# Feedback & Insights

When to use: Collecting qualitative input during onboarding, identifying friction points in setup flows, and gathering testimonials from successful integrations. Use for post-install surveys, NPS collection, and support ticket correlation.

## Patterns

**Exit Intent on Registration Abandonment**
URL registration form (`/leadconnector/register-urls`) tracks `last_activity_at` in session. If user navigates away without submitting, next page load triggers modal: "Need help completing setup?" with options: "I have a question," "My MLS isn't listed," "I'll finish later." Log responses to `ghl_audit_logs` with `event_type=registration_feedback`.

**Widget Integration Health Check**
Lead sync job (`SyncLeadToGhlJob`) surfaces failure patterns in dashboard: "Last 3 leads failed to sync—check GHL location permissions." Include one-click "Test sync" button that creates dummy `quantyra_lead` record and reports GHL API response. Success shows green check; failure shows exact HTTP status and GHL error message.

**Post-Activation Testimonial Prompt**
14 days after `subscription_status` changes to `active`, dashboard shows contextual request: "You've processed 50+ leads. Share your experience?" Link to Typeform with `location_id` pre-filled. Offer 1 month free on annual plan for video testimonials. Track submission in `ghl_installed_locations.testimonial_provided_at`.

## Warning

Do not interrupt critical setup flows (OAuth callback, Stripe checkout) with feedback modals—only capture feedback on non-blocking pages like dashboard or after explicit "Help" clicks. GHL marketplace policies require installations complete within 5 minutes; friction during setup violates their UX guidelines.