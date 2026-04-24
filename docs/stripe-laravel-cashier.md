# Stripe & Laravel Cashier (idx-api)

This application includes **[Laravel Cashier (Stripe)](https://laravel.com/docs/billing)** for subscription billing. Keys and webhook verification are configured through environment variables.

---

## Environment variables

Set these in **`.env`** (and in the runtime environment for Docker/Dokploy where Compose passes them through).

| Variable | Required | Description |
|----------|----------|-------------|
| `STRIPE_KEY` | Yes (for billing) | Stripe **publishable** key (`pk_test_…` / `pk_live_…`). Used with Stripe.js and public client flows. |
| `STRIPE_SECRET` | Yes (for billing) | Stripe **secret** key (`sk_test_…` / `sk_live_…`). Server-side API only; never expose to browsers. |
| `STRIPE_WEBHOOK_SECRET` | Yes (webhooks) | Webhook **signing secret** (`whsec_…`) used to verify `Stripe-Signature` on incoming webhook requests. |
| `CASHIER_CURRENCY` | Recommended | ISO currency code for charges and display defaults (e.g. `usd`). |
| `CASHIER_CURRENCY_LOCALE` | Recommended | Locale for money formatting (e.g. `en_US`). Requires PHP `intl` for non-default locales. |

Optional Cashier settings (see published `config/cashier.php` if you use it):

| Variable | Notes |
|----------|--------|
| `CASHIER_PATH` | Base URI segment for Cashier routes; default `stripe` → webhook URL path segment is `stripe`. |
| `STRIPE_WEBHOOK_TOLERANCE` | Clock skew tolerance in seconds for signature verification (default in config). |
| `CASHIER_LOGGER` | Log channel for Stripe SDK messages. |

**API keys:** [Stripe Dashboard → Developers → API keys](https://dashboard.stripe.com/apikeys).

---

## Webhook signing secret (`STRIPE_WEBHOOK_SECRET`)

Use the secret that matches **how** Stripe delivers webhooks to your app. They are **not** interchangeable.

### Production / staging (Dashboard)

1. [Developers → Webhooks](https://dashboard.stripe.com/webhooks).
2. Add or open the endpoint whose URL matches your app (see **Webhook URL** below).
3. Reveal and copy the endpoint **Signing secret** (`whsec_…`).

### Local or tunneled development (Stripe CLI)

The Stripe CLI signs forwarded events with its **own** secret (different from a Dashboard endpoint on the same account).

1. Install and log in: [Stripe CLI](https://stripe.com/docs/stripe-cli), then `stripe login` once.
2. Print the current CLI signing secret (test mode by default):

   ```bash
   stripe listen --print-secret
   ```

3. Put that value in **`STRIPE_WEBHOOK_SECRET`** in `.env`.

4. Forward events to your Laravel Cashier webhook while developing:

   ```bash
   stripe listen --forward-to "https://YOUR_IDX_API_HOST/stripe/webhook"
   ```

   Replace the host with your real API base (for example the value of `APP_URL`). If forwarding to HTTPS fails TLS verification in your setup, use `stripe listen --help` and consider `--skip-verify` for development only.

Keep **`stripe listen`** running in a terminal while you need webhooks; without it, Stripe will not reach your machine (unless you use a Dashboard endpoint URL that is publicly reachable instead).

---

## Local development commands (project standard)

Use the checked-in scripts so everyone runs the same flow:

```bash
# Start Docker dev services (API, image proxy, tunnel)
./scripts/docker-dev.sh up-watch

# Stop Docker dev services
./scripts/docker-dev.sh down

# Start Stripe webhook forwarding (uses STRIPE_SECRET from .env)
# VS Code: Run Task -> "Stripe Dev: Listen"
```

The Stripe helper resolves the webhook forward URL to:

1. `IDX_PLATFORM_URL` (preferred), otherwise
2. `APP_URL`

and appends `/stripe/webhook`.

You can also run the helper directly:

```bash
# Listen + forward webhooks
./scripts/stripe-dev.sh listen

# Fire a test checkout completion event
./scripts/stripe-dev.sh trigger-checkout-completed
```

---

## Webhook URL (Cashier default)

With the default Cashier path (`CASHIER_PATH` unset or `stripe`), the webhook route is:

```text
{APP_URL}/stripe/webhook
```

If you change `cashier.path` / `CASHIER_PATH`, the path segment changes accordingly; Stripe’s endpoint URL and any CSRF exclusions must match your configuration.

---

## First-time package setup (reference)

The `laravel/cashier` Composer dependency is already part of this project. For a fresh checkout or database:

```bash
php artisan vendor:publish --tag=cashier-migrations
php artisan migrate
php artisan vendor:publish --tag=cashier-config
```

Apply the **`Billable`** trait to the Eloquent model that owns subscriptions (often `User`), and follow the [Laravel Cashier documentation](https://laravel.com/docs/billing) for subscriptions, payment confirmation, and webhook behavior.

---

## Related documentation

- **[GHL environment variables](ghl-environment-variables.md)** — `GHL_SUBSCRIPTION_TAG_*` and other IDX/GHL env vars.
- **[Documentation index](INDEX.md)** — full `docs/` map.
