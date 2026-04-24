# Content Copy

## When to use

Crafting messaging for the sales landing page, subscription tier descriptions, and widget embed instructions. Reference when updating pricing displays or value propositions.

## Patterns

**Subscription Tier Messaging** (`app/Billing/SubscriptionCatalog.php`)
- Pro ($39/mo): "3 domains, teaser gating, basic GHL app"
- Smart ($79/mo): "5 domains, full GHL app, phone + email OTP"
- Ultra ($179/mo): "Unlimited domains, 2M API calls/mo, developer keys"
- Mega ($449/mo): "Unlimited everything, custom branding, SLA targets"

**Sales Landing Page** (`app/Livewire/Marketing/SalesLandingPage.php`)
- Interval toggle: Monthly vs annual (20% discount highlighted)
- Social proof: Lead volume estimates, GHL integration badge
- CTA: "Start 14-day trial" with plan selection state

**Widget Embed Instructions** (`resources/views/ghl/`)
- Installation complete page: Copy-pasteable loader.js snippet
- Value prop: "Add IDX search to any GHL website in 60 seconds"
- API key hint: `qh_...` prefix explained as "Quantyra Hash"

## Warning

Do not expose `BRIDGE_API_KEY` or Sanctum plaintext tokens in any public-facing copy. The `idx-images.quantyralabs.cc` URLs shown in JSON responses are safe to display as they require domain/token authentication to resolve.