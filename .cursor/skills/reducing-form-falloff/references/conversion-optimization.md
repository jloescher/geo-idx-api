# Conversion Optimization

## When to use
Improving form completion rates for the GHL widget lead capture flows. Use when analyzing drop-off in the funnel from MLS listing views to submitted leads.

## Patterns

### Progressive Gating with View Thresholds
Configure `gate_after_views` in `ghl_widget_configs` to delay the lead form requirement. Users browse ungated MLS content for N listing views before the form appears. Pair with `require_otp` for phone verification gating on high-intent actions like saving favorites or requesting showings.

```php
// In widget config or middleware logic
if ($viewCount >= config('ghl.widgets.gate_after_views', 3)) {
    // Require lead form completion before showing more listings
}
```

### Minimize Required Fields
Reduce `QuantyraLead` payload to email-only for initial capture. Use GHL contact enrichment to backfill name, phone, and address data after the lead enters the CRM. Map minimal fields in `GhlLeadMapping` to avoid overwhelming users.

### Inline Validation with Clear Errors
Return structured validation errors from `POST /widget/api/leads` that the widget JS can display inline. Avoid generic "validation failed" messages; specify which field needs attention.

## Warning
Do not set `gate_after_views` to 0 on the free tier—this eliminates the teaser value proposition and can violate Stellar MLS display guidelines. Always maintain the teaser/listings boundary per the subscription tier.