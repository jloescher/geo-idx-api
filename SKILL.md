---
name: calibrating-paid-campaigns
description: Aligns paid acquisition with landing pages and pixels for the Quantyra IDX marketing funnel
allowed-tools: ["Read", "Edit", "Write", "Glob", "Grep", "Bash"]
---

# Calibrating Paid Campaigns Skill

Configure landing pages, tracking pixels, and campaign attribution across the IDX marketing funnel from ad click to subscription.

## Quick Start

```bash
# Find marketing assets
grep -r "facebook\|meta\|gtag\|ga\|pixel" resources/views/ --include="*.blade.php"
grep -r "utm_\|campaign\|source" app/Livewire/Marketing/ --include="*.php"

# Check subscription checkout tracking
grep -r "checkout\|subscribe" app/Livewire/Marketing/ app/Http/Controllers/Billing/

# Verify GHL widget campaign attribution
grep -r "api_key\|lead_type" app/Ghl/Widgets/ --include="*.php"
```

## Key Concepts

### Landing Page Architecture

Marketing pages use **Livewire 3 + Blade + Tailwind 4** with Vite bundling. Entry points:

| Page | Component | Route |
|------|-----------|-------|
| Sales landing | `SalesLandingPage` | `marketing.sales` |
| Checkout | `SubscriptionCheckoutController` | `billing.checkout` |
| Dashboard | `DashboardController` | `dashboard` |

Tracking pixels belong in `resources/views/components/tracking/` or inline in `resources/views/marketing/`.

### Campaign Attribution Flow

```
Ad Click → Landing Page (UTM capture) → Session Store → Checkout/Lead → Stripe/GHL
                ↓                           ↓
         Meta Pixel (PageView)      Conversion event
```

UTM persistence uses Laravel session or encrypted cookies via `app/Http/Middleware/`—check for `EnsureUtmParameters` or similar campaign middleware.

### Pixel Placement Rules

| Pixel | Location | Event Triggers |
|-------|----------|----------------|
| Meta (Facebook) | `resources/views/layouts/marketing.blade.php` | PageView, InitiateCheckout, Subscribe |
| Google Analytics 4 | Same base layout | page_view, purchase, generate_lead |
| LinkedIn Insight | Conditional by campaign | `resources/views/components/tracking/linkedin.blade.php` |

GDPR/CCPA: Honor `DO_NOT_TRACK` cookies before firing pixels—check `app/Http/Middleware/RespectDoNotTrack.php`.

## Common Patterns

### Pattern: UTM Persistence Through Widget Flow

When a GHL widget loads via `/widget/loader.js`, pass campaign context:

```javascript
// In loader injection - widget client captures parent UTM
const utm = new URLSearchParams(window.location.search);
widget.dataset.utmSource = utm.get('utm_source') || '';
widget.dataset.utmCampaign = utm.get('utm_campaign') || '';
```

Widget POST to `/widget/api/leads` includes `api_key` and should persist UTM to `quantyra_leads.payload` JSON. Verify in `app/Ghl/Sync/Services/LeadSyncService.php` that UTM fields map to GHL contact custom fields.

### Pattern: Checkout Conversion Tracking

Stripe Checkout redirects to `IDX_PLATFORM_URL/dashboard?checkout=success`. Fire conversion pixels in `DashboardController` or `resources/views/dashboard/index.blade.php`:

```blade
@if(request('checkout') === 'success' && $user->subscriptions()->first()?->created_at->isAfter(now()->subMinute()))
    <script>fbq('track', 'Purchase', {value: {{ $planPrice }}, currency: 'USD'});</script>
@endif
```

### Pattern: Teaser vs Full Gating for Campaign Segments

Paid campaigns targeting "full access" must ensure `idx:full` Sanctum ability. Check token abilities in `DomainOrTokenAuth` middleware before allowing pixel conversion events for paid subscribers only.

### Pattern: GHL Install Attribution

When agencies install via `/leadconnector/install`, capture `utm_source` in session and persist to `ghl_installed_locations` or `ghl_oauth_tokens` metadata for LTV analysis. Query via `ghl_installed_locations` table joined with subscription status.

### Pattern: Campaign-Specific Landing Pages

Create variants in `app/Livewire/Marketing/` with `#[Layout('layouts.marketing-campaign')]`:

```php
// app/Livewire/Marketing/SalesLandingPageVariant.php
namespace App\Livewire\Marketing;

use Livewire\Attributes\Layout;
use Livewire\Component;

#[Layout('layouts.marketing-campaign')]
class SalesLandingPageVariant extends Component
{
    public string $campaign = 'q3-2026-agents'; // For pixel event parameter

    public function mount(Request $request): void
    {
        // Store campaign in session for attribution
        session(['campaign_attribution' => $this->campaign]);
    }
}
```

### Pattern: Testing Pixels Locally

```bash
# Start dev with tunnel for pixel testing
./scripts/docker-dev.sh up-watch

# Verify pixel requests in browser Network tab or use:
# - Meta Pixel Helper Chrome extension
# - Google Tag Assistant
```

### Pattern: Audit Trail for Compliance

All pixel events and UTM attributions log to `ghl_audit_logs` or `bridge_proxy_audit_logs`. Query for campaign effectiveness:

```sql
SELECT
    payload->>'utm_source' as source,
    payload->>'utm_campaign' as campaign,
    COUNT(*) as leads,
    DATE(created_at) as date
FROM quantyra_leads
WHERE payload ? 'utm_source'
GROUP BY source, campaign, date;
```

## Files to Know

| Path | Purpose |
|------|---------|
| `app/Livewire/Marketing/SalesLandingPage.php` | Main paid acquisition entry |
| `app/Http/Controllers/Billing/SubscriptionCheckoutController.php` | Stripe checkout, conversion trigger |
| `app/Ghl/Widgets/` | Widget UTM capture and lead ingestion |
| `app/Ghl/Sync/Services/LeadSyncService.php` | GHL contact creation with attribution |
| `resources/views/layouts/marketing.blade.php` | Base layout for pixel injection |
| `config/billing.php` | Plan definitions with revenue values for pixel events |
| `routes/web.php` | Marketing and billing routes |

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `IDX_PLATFORM_URL` | Canonical base for campaign URLs and redirects |
| `STRIPE_KEY` | Publishable key for checkout flow |
| `GHL_CLIENT_ID` | For OAuth install attribution |
| `TELESCOPE_ENABLED` | Set `false` in prod to reduce pixel noise in logs |
