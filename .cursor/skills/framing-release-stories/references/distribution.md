# Distribution

## When to use
Use when planning how releases reach users across the three public surfaces (idx.quantyralabs.cc, idx-api.quantyralabs.cc, idx-images.quantyralabs.cc) and coordinating rollouts that affect GHL Marketplace installs, widget embeds, or Stripe billing flows.

## Project-relevant patterns

**Surface-sequenced rollout**
Deploy order matters when changes touch multiple surfaces:
1. `idx-api` first (API contract changes, webhook handlers)
2. `idx-images` second (if URL rewriting logic changed in `BridgeImageUrlRewriter`)
3. `idx` (marketing/dashboard) last (depends on API stability)

**GHL Marketplace propagation**
OAuth tokens are long-lived. When releasing breaking changes to `/api/leadconnector/*` endpoints, the `WebhookDispatcher` must maintain backward compatibility for 24 hours or coordinate with GHL to force token refresh via their admin panel.

**Widget cache invalidation**
Widget consumers cache `loader.js`. Use versioned query parameters (`?v=1.2.3`) in embed snippets shown on `/leadconnector/installation-complete` to force client-side refresh when widget logic changes.

## Pitfalls
Do not assume all locations receive updates simultaneously. Agency tokens (`user_type=Company`) sync to multiple locations via `LocationTokenService`—a release affecting `ghl_widget_configs` schema must handle rows where `ghl_location_id` is null until the agency propagates the change or the location-specific token is exchanged.