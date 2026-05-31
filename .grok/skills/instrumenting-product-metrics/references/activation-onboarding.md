# Activation & Onboarding Reference

## Contents
- Activation Funnel Definition
- Current Instrumentation Gaps
- Activation Event Design
- Onboarding Flow Instrumentation
- Anti-Patterns

## Activation Funnel Definition

The idx-api activation funnel spans signup through first successful API call:

```
invitation_accepted → domain_created → domain_verified → token_created → first_api_call
```

Each step maps to a real code surface:

| Step | Handler | File | Currently Audited |
|------|---------|------|-------------------|
| `invitation_accepted` | `AcceptInvitation` | `internal/handler/dashboard/handler.go` | No |
| `domain_created` | `StoreDomain` | `internal/handler/dashboard/handler.go` | No |
| `domain_verified` | `VerifyTXT` | `internal/handler/dashboard/handler.go` | No |
| `token_created` | `CreateToken` / `CreateStagingToken` | `internal/handler/dashboard/handler.go` | No |
| `first_api_call` | Bridge/search handlers | `internal/handler/bridge/handler.go` | Yes (`mls_proxy_audit_logs`) |

Only the last step is captured. Steps 1–4 have zero instrumentation.

## Current Instrumentation Gaps

`mls_proxy_audit_logs` captures proxy API calls only. The dashboard onboarding flow is invisible in analytics.

### WARNING: Silent Onboarding Drop-Off

**The Problem:** No event fires when a user creates a domain, verifies it, or generates their first token. You cannot measure where users abandon onboarding.

**Why This Breaks:**
1. Cannot calculate activation rate (signup → first API call conversion)
2. Cannot identify which onboarding step has highest drop-off
3. Cannot measure time-to-activate by cohort

**The Fix:** Emit a product event at each onboarding mutation:

```go
// new code to add — after each dashboard mutation
l.audit.LogEvent(c.Context(), audit.ProductEvent{
    UserID:     user.ID,
    EventName:  "dashboard.domain.verified",
    Properties: map[string]any{"domain_slug": slug},
})
```

## Activation Event Design

### Event Names (Namespace: `onboarding.*`)

| Event | Trigger | Key Properties |
|-------|---------|----------------|
| `onboarding.invitation_accepted` | `AcceptInvitation` handler succeeds | `inviter_id` |
| `onboarding.domain_created` | `StoreDomain` INSERT succeeds | `domain_slug`, `domain_status` |
| `onboarding.domain_verified` | `VerifyTXT` DNS check passes | `domain_slug` |
| `onboarding.token_created` | `CreateToken` or auto-token on verify | `token_name`, `abilities` |
| `onboarding.first_api_call` | First `mls_proxy_audit_logs` row for domain | `request_type`, `cache_hit` |

### Activation Definition

A user is "activated" when their domain has at least one row in `mls_proxy_audit_logs` with `logged_at` after domain verification:

```sql
-- new code to add — activation rate query
SELECT
  COUNT(DISTINCT u.id) AS total_users,
  COUNT(DISTINCT a.user_id) AS activated_users,
  ROUND(COUNT(DISTINCT a.user_id)::numeric / NULLIF(COUNT(DISTINCT u.id), 0) * 100, 1) AS activation_pct
FROM users u
LEFT JOIN (
  SELECT DISTINCT user_id FROM mls_proxy_audit_logs
  WHERE logged_at > NOW() - INTERVAL '30 days'
) a ON a.user_id = u.id;
```

## Onboarding Flow Instrumentation

### DO: Emit events after mutation succeeds

```go
// GOOD — event only fires on success
result, err := h.domainRepo.Verify(ctx, domainID)
if err != nil { return err }
h.audit.LogEvent(ctx, ProductEvent{EventName: "onboarding.domain_verified", ...})
```

### DON'T: Emit events before mutation

```go
// BAD — event fires even if verification fails
h.audit.LogEvent(ctx, ProductEvent{EventName: "onboarding.domain_verified", ...})
result, err := h.domainRepo.Verify(ctx, domainID)  // might fail
```

## Anti-Patterns

### WARNING: Using slog for Product Events

`slog.Info("domain verified")` is not queryable. Product events must be in PostgreSQL for funnel analysis, cohort queries, and retention calculations. Use the audit logger pattern from `internal/service/audit/logger.go`.

### WARNING: Blocking Requests on Event Write

The existing `audit.Logger.Log` uses `_, _ = l.db.Pool.Exec(...)` — it discards errors. This is correct: never block the user's request because analytics failed.

See the **auth-api-token** skill for invitation and token lifecycle details.
See the **fiber** skill for middleware patterns that resolve `user_id` and `domain_slug` from context.