# Conversion Optimization

## When to use

Optimizing the GHL OAuth install funnel, widget lead capture rates, and subscription tier upgrades. Reference when analyzing drop-off points from initial widget impression to paid subscription activation.

## Patterns

**OAuth Install Funnel** (`routes/ghl-web.php`)
- Entry: `GET /leadconnector/install` → Blade landing with value props
- Friction reducer: Optional `user_type=Company` param for agency installs
- Conversion: Callback persists token, session handoff to URL registration
- Activation: `POST /leadconnector/register-urls` issues widget API key (`qh_...` prefix)
- Metric: `ghl_oauth_tokens.status = 'active'` with valid `ghl_registered_urls.widget_api_key`

**Widget Lead Capture Optimization** (`routes/ghl-widget.php`)
- Entry points: `/widget/search/{apiKey}`, `/widget/lead-form/{apiKey}`, `/widget/showcase/{apiKey}`
- Conversion: `POST /widget/api/leads` creates `quantyra_leads` row
- Success signal: `ghl_sync_logs.sync_status = 'success'` with non-null `ghl_contact_id`
- Teaser gate: Non-`idx:full` domains see 3-listing cap → natural upgrade prompt

**Subscription Tier Upgrade Path** (`app/Billing/SubscriptionCatalog.php`)
- Trial: 14-day default via Stripe Checkout
- Upgrade triggers: Domain limit exceeded (Pro=3, Smart=5), API overage, `idx:full` capability request
- Checkout: `SubscriptionCheckoutController` with `?plan=pro|smart|ultra|mega` param
- Activation: Stripe `checkout.completed` webhook → `ghl_installed_locations.subscription_status = 'active'`

## Warning

The `idx:full` Sanctum ability bypasses teaser gating but does not automatically grant unlimited domains. Check `subscription_status` in `ghl_installed_locations` before allowing additional domain registrations—token authentication alone does not indicate paid subscription tier.