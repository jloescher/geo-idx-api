# Measurement & Testing Reference

## Contents
- Observable Metrics
- Audit Log as Conversion Tracking
- Testing Decision Cues
- Anti-Patterns

## Observable Metrics

Quantyra IDX has no client-side analytics, no tracking scripts, and no A/B testing framework. Conversion measurement must be derived from server-side data.

### Available Data Sources

| Source | Table / mechanism | What it measures |
|--------|-------------------|------------------|
| `audit_logs` | API request logging | Token usage frequency, endpoint adoption |
| `personal_access_tokens` | Token CRUD | Token creation rate, revocation rate |
| `domains` | Domain lifecycle | Verification completion rate, time-to-verify |
| `listings` | Mirror data | PostGIS search volume (via query patterns) |
| `jobs` | Queue processing | Replication throughput, freshness |

### Key Funnel Metrics (SQL-Derivable)

```sql
-- Domain verification completion rate
SELECT
  COUNT(*) FILTER (WHERE verification_status IN ('verified','verified_ghl'))::float
  / NULLIF(COUNT(*), 0) AS verification_rate
FROM domains;

-- Time from domain creation to verification
SELECT AVG(EXTRACT(EPOCH FROM (txt_verified_at - created_at)) / 3600) AS hours_to_verify
FROM domains WHERE txt_verified_at IS NOT NULL;

-- Token creation per user (engagement signal)
SELECT tokenable_id, COUNT(*) AS tokens_created,
  COUNT(*) FILTER (WHERE name = 'Production') AS production_tokens
FROM personal_access_tokens
WHERE tokenable_type = 'App\Models\User'
GROUP BY tokenable_id;
```

## Audit Log as Conversion Tracking

The `audit_logs` table records authenticated API requests. This is the closest thing to conversion tracking in the platform.

### DO: Use audit data to measure API engagement

```sql
-- API endpoint adoption by domain
SELECT d.domain_slug, COUNT(a.id) AS requests,
  COUNT(DISTINCT DATE(a.created_at)) AS active_days
FROM audit_logs a
JOIN domains d ON d.id = a.domain_id
GROUP BY d.domain_slug
ORDER BY requests DESC;
```

### DON'T: Add client-side analytics to the dashboard

The dashboard is server-rendered HTML with no JavaScript framework. Adding analytics scripts would require significant re-architecture and provides minimal value for an invite-only B2B tool.

## Testing Decision Cues

### Testing Copy Changes

Since copy lives in Go string literals, test changes through build verification and manual review:

1. Edit the string literal in the handler
2. Run: `go build ./cmd/api`
3. Run: `go test ./internal/handler/...`
4. Start API and verify in browser

### Testing GIS Teaser Behavior

```go
// Test that teaser truncation works correctly
// Verify with: go test ./internal/service/gis/...
//
// Key assertions:
// - Full-access tokens receive unmodified GeoJSON
// - Teaser tokens receive capped feature count
// - Teaser tokens receive rounded coordinates
// - Truncation flag is returned when features exceed cap
```

### Testing Token One-Time Display

The one-time display guarantee is architectural: the plain token is never stored. Verify by:
1. Completing DNS verification
2. Confirming the token appears on the success page
3. Navigating back — the token page cannot be re-rendered with the plain text

## Anti-Patterns

### WARNING: Adding Analytics Without Privacy Review

**The Problem:** Adding tracking to an API platform without considering that MLS data may have licensing restrictions on usage tracking.

**Why This Breaks:** MLS agreements may prohibit tracking how listing data is consumed. Audit logs track API access patterns (legitimate) but tracking listing-level interactions may violate terms.

**The Fix:** Only track request-level metadata (endpoint, timestamp, domain) in `audit_logs`. Never track which specific listings a user viewed or searched.

### WARNING: A/B Testing in String Literals

**The Problem:** Go string literals don't support dynamic content selection. Adding A/B logic to inline HTML creates unmaintainable handler code.

**The Fix:** For now, make copy decisions based on the SQL-derived funnel metrics above. If A/B testing becomes necessary, extract copy into a data-driven template system first.

See the **cache-postgres** skill for audit log storage patterns.
See the **go** skill for testing patterns in Go.