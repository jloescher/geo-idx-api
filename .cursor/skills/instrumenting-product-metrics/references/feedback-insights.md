# Feedback & Insights

Use when collecting, categorizing, and acting on user feedback from GHL locations and widget end-users.

## Patterns

**Widget Error Telemetry**

```php
// Capture client-side errors from widget loader
public function logWidgetError(Request $request)
{
    \DB::table('widget_errors')->insert([
        'api_key' => $request->input('api_key'),
        'error_type' => $request->input('type'), // 'config_load', 'listing_fetch', 'lead_submit'
        'message' => $request->input('message'),
        'user_agent' => $request->header('User-Agent'),
        'url' => $request->input('url'), // Page where widget is embedded
        'occurred_at' => now(),
    ]);
    
    // Alert if error rate spikes
    $recentErrors = \DB::table('widget_errors')
        ->where('api_key', $request->input('api_key'))
        ->where('occurred_at', '>', now()->subMinutes(5))
        ->count();
    
    if ($recentErrors > 10) {
        // Dispatch alert to ops channel
    }
}
```

**Lead Quality Scoring**

```sql
-- Identify leads that sync successfully vs fail
SELECT 
    lead_type,
    sync_status,
    COUNT(*),
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at))) as avg_resolution_seconds,
    STRING_AGG(DISTINCT LEFT(response_payload->>'error', 100), '; ') as common_errors
FROM ghl_sync_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY lead_type, sync_status;
```

**Subscription Cancellation Intent Signals**

```sql
-- Predict churn from usage patterns
SELECT 
    l.ghl_location_id,
    l.subscription_status,
    l.mls_request_count,
    -- Days since last meaningful engagement
    EXTRACT(DAY FROM NOW() - MAX(a.logged_at)) as days_since_last_request,
    -- Trial expiring soon without conversion
    CASE WHEN l.subscription_status = 'trial' 
         AND s.created_at < NOW() - INTERVAL '10 days' 
         THEN 'likely_churn' END as risk_flag
FROM ghl_installed_locations l
LEFT JOIN bridge_proxy_audit_logs a ON l.ghl_location_id = a.user_id::text
LEFT JOIN subscriptions s ON l.ghl_location_id = s.stripe_id  -- via Cashier relation
GROUP BY l.ghl_location_id, l.subscription_status, l.mls_request_count, s.created_at
HAVING EXTRACT(DAY FROM NOW() - MAX(a.logged_at)) > 3;
```

## Warning

Widget errors logged from client-side JavaScript may contain PII in the `message` or `url` fields—always sanitize or hash email addresses and phone numbers before persistence, and retain logs only as long as necessary for debugging.