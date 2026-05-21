# Feedback & Insights Reference

## Contents
- Current Feedback Channels
- Audit Log as Feedback Signal
- Dashboard Feedback Patterns
- Support Signal Detection
- DO/DON'T Patterns

## Current Feedback Channels

This project has no in-app feedback widget, no support ticket system, and no NPS survey mechanism. Feedback channels are:

1. **Direct communication** — users contact Quantyra team directly
2. **Audit log patterns** — failed requests, error rates, zero-result searches
3. **GitHub issues** — if the repo is public for API documentation

### WARNING: Missing Feedback Collection

**Detected:** No in-app feedback mechanism, no error reporting service, no user survey system.

**Impact:** Cannot systematically collect user sentiment or identify UX friction points without manual outreach.

**Mitigation:** Use audit log analysis to detect friction (high failure rates, repeated verification attempts, zero-result searches). These are indirect signals but available immediately.

## Audit Log as Feedback Signal

The audit log captures behavioral signals that indicate friction:

| Signal | SQL pattern | Interpretation |
|--------|-------------|----------------|
| High verification failures | Domains with many verify attempts but no `txt_verified_at` | DNS instructions unclear |
| Zero-result searches | `listing_count = 0` in audit logs | Dataset/query mismatch |
| Token churn | Frequent create/revoke cycles | Token UX confusing |
| API errors | Non-200 status in audit | Integration issues |
| Abandonment after signup | User with domain but no audit entries | Onboarding drop-off |

### Pattern: Friction Detection Query

```sql
-- new code to add — find domains stuck in verification
SELECT d.slug, d.hostname, d.created_at,
       COUNT(v.id) AS verify_attempts
FROM domains d
LEFT JOIN domain_verification_attempts v ON v.domain_id = d.id
WHERE d.txt_verified_at IS NULL
AND d.created_at > NOW() - INTERVAL '14 days'
GROUP BY d.id
HAVING COUNT(v.id) > 2
ORDER BY verify_attempts DESC;
```

### Pattern: Zero-Result Search Detection

```sql
-- new code to add — find searches returning no results
SELECT domain_slug, request_type, COUNT(*) AS zero_result_count
FROM mls_proxy_audit_logs
WHERE listing_count = 0
AND request_type LIKE '%search%'
AND created_at > NOW() - INTERVAL '7 days'
GROUP BY domain_slug, request_type
ORDER BY zero_result_count DESC;
```

## Dashboard Feedback Patterns

### Pattern: Simple Feedback Link

Add a feedback link to the dashboard footer using existing layout:

```html
<!-- new code to add — in internal/web/layout.go Page() footer -->
<footer class="site-footer">
    <a href="mailto:support@quantyralabs.cc">Report an issue</a>
</footer>
```

### Pattern: Inline Error Feedback

When a dashboard action fails, show the specific error with context:

```go
// GOOD — specific, actionable error message
func (h *Handler) VerifyTXT() fiber.Handler {
    return func(c *fiber.Ctx) error {
        verified, err := h.domainService.VerifyTXT(domainID)
        if !verified {
            // Show specific reason, not generic error
            c.Redirect("/dashboard?error=dns_not_found&domain=" + hostname)
            return nil
        }
    }
}
```

### WARNING: Generic Error Messages

**The Problem:**

```go
// BAD — user has no idea what to do
c.Redirect("/dashboard?error=verification_failed")
```

**Why This Breaks:**
1. User cannot self-correct without knowing what went wrong
2. Generates support requests that could be avoided
3. No diagnostic information for the team

**The Fix:** Provide specific, actionable error messages:

```go
// GOOD — user can act on this information
c.Redirect("/dashboard?error=dns_txt_not_found&hint=Check+that+the+TXT+record+matches+exactly")
```

## Support Signal Detection

Track these patterns in audit logs to proactively identify users needing support:

| Signal | Detection | Action |
|--------|-----------|--------|
| User stuck at verification | Domain > 3 days old, not verified, multiple verify attempts | Outreach with DNS help |
| User never made API call | Domain verified but no audit entries | Share integration examples |
| User hitting rate limits | Spikes in request count per domain | Suggest caching or batching |
| User searching wrong dataset | Zero results on active dataset | Clarify dataset parameter |

### Pattern: Proactive Support Dashboard

Admin-only view showing users needing attention:

```go
// new code to add — admin support signals
type SupportSignal struct {
    DomainSlug string
    Signal     string // "stuck_verification", "no_api_calls", "high_error_rate"
    DaysSince  int
}
```

## DO/DON'T Patterns

### DO: Analyze audit logs for friction patterns

```sql
-- GOOD — systematic, data-driven
SELECT domain_slug, COUNT(*) FILTER (WHERE listing_count = 0) AS empty_searches
FROM mls_proxy_audit_logs GROUP BY domain_slug;
```

### DON'T: Rely on users reporting problems

```go
// BAD — passive, captures only vocal minority
// "Users will tell us if something is broken" ← they won't
```

### DO: Show contextual help at friction points

```html
<!-- GOOD — help appears where the user is stuck -->
<p class="hint">
    DNS not found? Common issues: trailing dots, wrong record type,
    propagation delay. <a href="/docs/dns-setup">Full guide</a>
</p>
```

### DON'T: Link to generic documentation

```html
<!-- BAD — user must search through docs to find the relevant section -->
<p>See <a href="/docs">documentation</a> for help.</p>
```

## Integration Points

- **Audit logger**: `internal/service/audit/logger.go` — the event recording mechanism.
- **Dashboard handler**: `internal/handler/dashboard/handler.go` — where error feedback renders.
- **Layout templates**: `internal/web/layout.go` — `Page()` footer for feedback links.
- See the **ux** skill for error message design and accessibility.
- See the **product-analytics** skill for audit log queries.