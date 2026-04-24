# Measurement & Testing

## When to use
Apply when implementing observability, compliance audit trails, or load testing for the IDX API and GHL integrations.

## Patterns

### Dual-Channel Audit Logging
GHL subsystem writes to both database and file for compliance durability:

```php
// GhlAuditService - Stellar MLS audit requirements
public function log(array $data): void
{
    GhlAuditLog::create([
        'logged_at' => now(),
        'api_endpoint' => $data['endpoint'],
        'latency_ms' => $data['latency'],
        'is_mls_data_access' => $data['is_mls'],
        'compliance_verified' => true,
    ]);

    // Secondary file channel for log aggregation
    if (config('ghl.audit_log_enabled')) {
        Log::channel('ghl_audit')->info('API call', $data);
    }
}
```

### Feature Test Isolation
All tests use in-memory SQLite with faked HTTP clients:

```php
// TestCase.php - Ephemeral database guard
protected function setUp(): void
{
    parent::setUp();

    if (config('database.default') !== 'sqlite' 
        && !env('ALLOW_DESTRUCTIVE_TEST_DB')) {
        $this->fail('Tests require SQLite in-memory database');
    }

    Http::fake([
        'api.bridgedataoutput.com/*' => Http::response(['value' => []]),
        'api.stripe.com/*' => Http::response(['id' => 'pi_test']),
    ]);
}
```

### Health Check Probes
GIS metadata probing tracks source freshness:

```php
// RefreshGisSourceMetadataJob - fingerprint-based invalidation
$response = Http::timeout(config('gis.metadata_timeout', 12))
    ->get($sourceUrl . '?f=json');

$fingerprint = md5($response->json('currentVersion') . 
                   json_encode($response->json('editingInfo')));

if ($fingerprint !== $sourceState->last_fingerprint) {
    $sourceState->increment('generation'); // Invalidates cache
}
```

## Pitfall
Never enable Telescope or Pulse in production without HTTP Basic Auth. The `ProtectMonitoringDashboard` middleware gates these, but exposing them exposes OAuth tokens and Stripe keys.