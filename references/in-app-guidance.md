# In-App Guidance

When to use: Contextual help, tooltips, and empty-state messaging inside the subscriber dashboard and installation wizard.

## Project-Relevant Patterns

### Step-Based Contextual Hints
The GHL OAuth flow uses Blade components with conditional `step` prop to show guidance relevant to current state: on `/leadconnector/install` show "You'll authorize Quantyra in your GHL dashboard"; on `/leadconnector/register-urls` show "Enter the exact domain where your website is hosted (e.g., https://yoursite.com)". Keep hints within 12 words.

### Empty State Templates
The subscriber dashboard uses Livewire conditional rendering for zero-data scenarios: no widgets yet → show embed code snippet with copy button; no leads yet → show "Your lead form will appear here once visitors submit"; no subscription → show pricing tier comparison with highlighted recommendation based on estimated traffic.

### Toast Notifications for Async Operations
Token refresh and lead sync happen via queued jobs. Use Livewire's `$dispatch('notify', [...])` pattern to show "Syncing your leads to GHL..." on dispatch and "X leads synced" on completion via `LeadSyncService` event broadcasting.

## Pitfall
Avoid showing all guidance at once. The installation wizard specifically sequences information: OAuth step hides widget embed instructions, and embed step hides MLS compliance details. Violating this sequencing causes users to skip critical steps while trying to read ahead.
