# Content Copy

## When to use
Reference when writing user-facing messaging, API documentation, or marketing copy for the IDX platform and GHL Marketplace integration.

## Patterns

### Capability-Based Headlines
Focus on outcomes, not features. The subscription catalog uses tiered messaging:

| Tier | Headline Pattern |
|------|------------------|
| Pro | "Get started with IDX on your domain" |
| Smart | "Unlock full GHL integration + phone OTP" |
| Ultra | "Scale with unlimited domains + 2M API calls" |
| Mega | "Enterprise SLA with custom branding" |

### Error Messages as Conversion Opportunities
Bridge proxy auth failures return actionable next steps:

```php
// DomainOrTokenAuth middleware
if (!$domain) {
    return response()->json([
        'error' => 'Domain not registered',
        'message' => 'Register your domain at ' . config('idx.platform_url') . '/dashboard',
        'upgrade_url' => route('billing.checkout', ['plan' => 'pro']),
    ], 403);
}
```

### Documentation Hierarchy
Follow the established doc structure for consistency:
1. **Quick Start** - Immediate runnable commands
2. **Goals table** - What this achieves and how
3. **Architecture diagram** - Mermaid flowchart showing data flow
4. **Example requests** - Copy-paste curl commands

## Pitfall
Avoid referencing internal code names ("geo-web", "Stellar") in user-facing copy. Use "your domain", "MLS listings", or "property data" instead.