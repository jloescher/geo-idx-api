# In-App Guidance

Use for contextual tooltips, progressive disclosure, and inline help within Livewire components and Blade templates where users make configuration decisions.

## Patterns

### Progressive Form Disclosure
On `/leadconnector/register-urls`, the "Additional URLs" section starts collapsed with a "Add more domains" text button. Expanding reveals input fields with inline help: "Used for staging environments and subdomains." This keeps the initial form minimal while supporting complex multi-domain setups.

### Contextual Webhook Help
Near the webhook URL field in configuration UIs, show a "How do I set this up in GHL?" disclosure. Clicking expands step-by-step instructions with the exact URLs to paste into the GHL Marketplace dashboard, reducing support tickets from copy-paste errors.

### Billing Interval Toggle
The Livewire `SalesLandingPage` uses a segmented control for monthly/annual billing with inline savings math ("Save 20%"). When toggled, prices animate to reinforce the value, and a tooltip explains prorated upgrades for existing subscribers.

## Warning

Inline guidance must not block form submission. Users on slow connections or small screens may never see below-the-fold help text. Critical compliance information (like MLS domain requirements) must appear above the fold or as validation errors, not just in collapsible help sections.