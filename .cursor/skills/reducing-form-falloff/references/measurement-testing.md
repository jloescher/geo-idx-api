# Measurement & Testing

## When to use
Tracking funnel metrics, identifying drop-off points, or validating form changes. Use when building dashboards or running optimization experiments.

## Patterns

### Audit Logging for Drop-off Analysis
Log form step completions to `ghl_audit_logs` with `is_mls_data_access=false` and a `step` metadata field. Query for sequences: view → form_open → email_entered → submit to identify abandonment points.

```php
GhlAuditService::log([
    'event' => 'widget_form_step',
    'step' => 'email_entered',
    'is_mls_data_access' => false,
]);
```

### A/B Test Assignment
Store variant assignment in the widget API key metadata or pass `?variant=` through the loader.js query string. Ensure the variant parameter propagates to `quantyra_leads.payload` for downstream attribution.

### Lead Quality Scoring
Tag leads in `ghl_lead_mappings` based on form completion depth. Partial submissions (email only) vs. full profiles (phone + timeline) should map to different GHL pipelines or opportunity stages for appropriate follow-up cadence.

## Warning
Avoid logging PII (emails, phones) in `ghl_audit_logs`—the table is query-optimized and may be exposed to analytics tools. Store sensitive lead data only in `quantyra_leads` with proper encryption.