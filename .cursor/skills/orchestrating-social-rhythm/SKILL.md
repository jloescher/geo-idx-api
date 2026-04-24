---
name: orchestrating-social-rhythm
description: Plans social content beats and distribution rhythm
version: 1.0.0
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Orchestrating Social Rhythm Skill

This skill schedules and coordinates content distribution across channels, aligning publication timing with audience engagement patterns.

## Quick Start

1. Review scheduled tasks in `routes/console.php` for timing patterns
2. Check `app/Console/Commands/` for existing command implementations
3. Verify queue configuration in `config/queue.php`

## Key Concepts

- **Cadence**: The rhythm of recurring content beats (e.g., listings cache refresh every 15 minutes, GHL token refresh hourly, GIS metadata weekly)
- **Channel**: Distribution surface (webhooks, widgets, API endpoints, email)
- **Batch**: Grouped operations dispatched as a single job to optimize queue throughput
- **Overlap Prevention**: Using Laravel's `withoutOverlapping()` to prevent duplicate job runs

## Common Patterns

### Scheduling Content Beats

```php
// routes/console.php
Schedule::command('ghl:refresh-tokens')
    ->hourly()
    ->withoutOverlapping()
    ->onOneServer();

Schedule::command('bridge-listings-cache-refresh')
    ->everyFifteenMinutes()
    ->withoutOverlapping();
```

### Queue-Based Distribution

```php
// Dispatch to specific queues for channel isolation
ProcessGhlWebhookJob::dispatch($event)
    ->onQueue('webhooks');

SyncLeadToGhlJob::dispatch($lead)
    ->onQueue('sync');
```

### Conditional Publishing

```php
// Skip when not in production
Schedule::command('billing:sync-invoices')
    ->daily()
    ->environments(['production']);
```