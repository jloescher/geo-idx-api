---
name: streamlining-signup-steps
description: Reduces friction in signup and trial activation
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Streamlining Signup Steps Skill

Skill for reducing friction in the signup and trial activation flow for the Quantyra GeoIDX platform, covering GHL Marketplace OAuth, user registration, billing checkout, and post-install URL registration.

## Quick Start

Key files for signup flow modifications:

| Path | Purpose |
|------|---------|
| `app/Ghl/OAuth/Controllers/` | OAuth authorization, callback, and URL registration controllers |
| `app/Http/Controllers/Billing/SubscriptionCheckoutController.php` | Stripe Checkout session creation |
| `app/Livewire/Marketing/SalesLandingPage.php` | Marketing page with pricing and trial CTA |
| `app/Http/Responses/Auth/RegisterResponse.php` | Post-registration redirect behavior |
| `routes/ghl-web.php` | OAuth and install routes |
| `resources/views/leadconnector/` | Install, URL registration, and completion Blade templates |

## Key Concepts

### GHL Marketplace OAuth Flow
The three-phase install: `GET /leadconnector/install` → OAuth authorize → callback → `GET /leadconnector/register-urls` (MLS compliance step) → `GET /leadconnector/installation-complete`. Session carries `ghl_pending_oauth_token_id` between steps.

### Trial Activation via Stripe
`SubscriptionCheckoutController::checkout()` creates Stripe Checkout with trial days. Cashier webhook `customer.subscription.created` activates the subscription. Plans defined in `app/Billing/SubscriptionCatalog.php` and `config/billing.php`.

### Post-Auth Redirects
- `RegisterResponse` redirects new users to `route('dashboard')` after registration
- `LoginResponse` redirects authenticated users to dashboard
- GHL embed route `/leadconnector/embed/{locationId}` 302-redirects to `IDX_PLATFORM_URL/embed/{locationId}`

### URL Registration Gate
After OAuth, users must register HTTPS origins (`primary_url`, `additional_urls`) before widget installation completes. This enforces Stellar MLS compliance—stored in `ghl_registered_urls` table with generated widget API key (`qh_...` prefix).

## Common Patterns

### Skip URL registration for specific user types
In `app/Ghl/OAuth/Controllers/CallbackController.php`, after token persistence, check `$token->user_type === 'Company'` and redirect directly to `installation-complete` for agency-only installs:

```php
if ($token->user_type === 'Company') {
    return redirect()->route('leadconnector.installation-complete');
}
return redirect()->route('leadconnector.register-urls');
```

### Auto-redirect to billing after OAuth completion
In `app/Ghl/OAuth/Controllers/UrlRegistrationController.php` `store()` method, append checkout session creation:

```php
return redirect()->route('billing.checkout', ['plan' => 'pro']);
```

### Pre-populate URLs from Referer
In `app/Ghl/OAuth/Controllers/UrlRegistrationController.php` `create()` method, extract domain from `url()->previous()` to pre-fill the primary URL field in the Blade template.

### Streamline: combine OAuth + registration
For direct signups (non-GHL), the `SalesLandingPage` Livewire component handles plan selection and redirects to `/register` with plan stored in session, then to checkout post-registration.

### Conditionally require OAuth based on subscription
Modify `app/Http/Middleware/EnsureUserHasPlan.php` to allow trial users to skip GHL OAuth and use dashboard tokens directly—gate on `auth()->user()->subscriptions()->active()->exists()` rather than GHL token presence.