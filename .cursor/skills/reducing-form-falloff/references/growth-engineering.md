# Growth Engineering

## When to use
Scaling lead capture infrastructure, optimizing queue throughput, or handling traffic spikes from marketing campaigns. Use when the widget volume exceeds single-server capacity.

## Patterns

### Async Lead Persistence
Queue writes to `quantyra_leads` via `SyncLeadToGhlJob` to prevent API latency from blocking the widget response. Return 202 Accepted immediately; retry GHL contact creation failures with exponential backoff.

### Cache Layer for Widget Config
The 3-phase middleware fetches `ghl_widget_configs` on every request. Cache location-specific configs in Laravel Cache for the duration of the HTTP request lifecycle to reduce PostgreSQL queries.

### Horizontal Scaling Considerations
Store widget sessions client-side (JWT or encrypted cookie) rather than server-side session to keep idx-api stateless. The Bearer token lookup via `access_token_hash` is database-bound—consider read replicas for high-volume widget endpoints.

## Warning
Do not enable server-side session affinity for widgets unless absolutely required—it complicates deployment and conflicts with the Octane worker model. Use signed client-side tokens for any session-like state.