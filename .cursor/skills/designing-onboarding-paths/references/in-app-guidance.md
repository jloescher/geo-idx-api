# In-App Guidance

When to use: Contextual help within Livewire components and Blade views where users perform complex configuration. Use for domain registration forms, API token management, and embed code customization.

## Patterns

**Inline Field Tooltips with MLS Compliance Context**
Domain registration form includes help text below each input explaining Stellar MLS Exhibit A requirements: "Must match approved hostname exactly. Wildcards not permitted." Use Tailwind `group` and `peer` classes to show/hide detailed explanations on focus without JavaScript. Include link to `docs/idx-api-bridge-proxy.md` for domain verification troubleshooting.

**Copy-to-Clipboard with Visual Feedback**
API token controller (`DashboardApiTokenController`) returns tokens masked by default with reveal toggle. Copy button uses Alpine.js (`x-on:click`) to copy plain text and show temporary "Copied!" toast. Include usage example (cURL command) directly below token that auto-updates with the revealed key.

**Contextual Empty States with Next Steps**
Widget config page shows empty state when no URLs registered: "No domains configured" with primary CTA to `/leadconnector/register-urls` and secondary link to documentation. For GIS proxy users without bbox queries yet, show sample Clearwater coordinates from `docs/gis-api.md` as placeholder text in query builder.

## Warning

Do not embed third-party JavaScript widgets for guidance tours—they conflict with GHL's own iframe embed environment and violate MLS data handling policies. Keep all guidance server-rendered in Blade to maintain Content Security Policy integrity.