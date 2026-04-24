# Conversion Optimization

## When to use
Optimizing subscription funnels, widget lead capture, GHL OAuth onboarding completion, or checkout flows in the Quantyra IDX platform.

## Patterns

**Teaser-to-full progression in listings**
The Bridge proxy uses `idx:access` vs `idx:full` token abilities to gate listing data. In the frontend, use progressive disclosure: show 3 listings with a "View more" CTA that triggers the login/subscription modal. The `SalesLandingPage` Livewire component already implements billing interval toggling—extend this pattern for feature comparisons.

**Widget embed conversion gates**
Widget surfaces at `/widget/*` routes use API keys (`qh_*` prefix) with origin validation. Add lead capture modals after 3 listing views using `gate_after_views` from `ghl_widget_configs`. Keep widget prompts under 40KB total payload to avoid slowing host sites.

**GHL OAuth flow completion**
The `/leadconnector/register-urls` step requires MLS domain registration. Add inline validation with `wire:loading` states on the URL submission form. Use `wire:poll` to show "OAuth pending" states when waiting for GHL token exchange—reduces abandonment at the install step.

## Warning
Avoid gating Bridge API calls with frontend-only checks. Always enforce teaser limits server-side via `BridgeTeaser` service—domain-authenticated requests bypass client-side gates. The `DomainOrTokenAuth` middleware returns `idx:full` only for valid Sanctum tokens with that ability.