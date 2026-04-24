# Distribution

## When to use
Deploy acquisition tools across multiple surfaces—GHL Marketplace widgets, standalone embeds, and direct API access—while maintaining security and origin validation.

## Patterns

**GHL Marketplace Distribution**
The OAuth flow (`/leadconnector/install`) → URL registration (`/leadconnector/register-urls`) → widget key issuance creates the distribution chain. Agency tokens exchange to location tokens via `LocationTokenService` so SaaS agencies can push widgets to sub-accounts without re-authentication.

**Three-Phase Widget Middleware**
All widget routes use: (1) API key validation against `ghl_registered_urls.widget_api_key`, (2) Origin/Referer validation against registered URLs, (3) CORS header append. This allows cross-origin embedding while preventing unauthorized domains from consuming quota.

**Standalone Public Tools**
Build Livewire components in `app/Livewire/Marketing/` using `DomainOrTokenAuth` middleware. These bypass GHL OAuth entirely for direct traffic acquisition, funneling users into Stripe Checkout via `SubscriptionCheckoutController`.

## Warning
The `GHL_WEBHOOK_REQUIRE_SIGNATURE` toggle must remain `true` in production. Disabling signature verification exposes the `/webhooks/leadconnector` endpoint to spoofed uninstall events that could orphan active subscriptions.