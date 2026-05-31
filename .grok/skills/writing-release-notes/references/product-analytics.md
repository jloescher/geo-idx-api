# Product Analytics Reference

How release notes track and communicate measurable product changes for Quantyra IDX API.

## Measurable Changes in Release Notes

Quantyra IDX API changes fall into measurable categories. Good release notes tie features to observable metrics.

### Change Categories and Metrics

| Category | Example Change | Observable Metric |
|----------|---------------|-------------------|
| Performance | PostGIS mirror search | Response latency p50/p95 |
| Data coverage | New MLS dataset support | `listings` row count per `dataset_slug` |
| Feature adoption | Comps investor modes | `POST /api/v1/comps/run` mode distribution |
| Reliability | Fair replication pipeline | `replica_pages` throughput, queue depth |
| Cost | Proxy cache purge | `mls_search_cache` row count after purge |

### DO: Include measurable impact when available

```markdown
## v2.2.0

### Performance
- **Search**: PostGIS mirror leg returns Active/Pending results from indexed columns (no live Bridge call) — reduces p95 latency for covered statuses
- **Replication**: Fair reservation across `bridge-sync-fetch` and `spark-sync-persist` queues prevents Bridge backlog from starving Spark fetch
```

### DON'T: Claim improvements without grounding

```markdown
<!-- BAD — no basis for the claim -->
### Performance
- Search is now much faster
- Replication is significantly improved
```

### Analytics-Adjacent Release Patterns

When a release adds instrumentation or monitoring:

1. Note new env vars for observability (e.g., `SCHEDULER_STANDBY_POLL_SECONDS`)
2. Reference log format changes (slog structured output)
3. Mention new health endpoints or stat routes (`GET /api/v1/bridge/stats`)

### Release Notes for Queue/Worker Changes

Worker changes are analytics-relevant because they affect throughput and job completion:

- New queue names must be added to `WORKER_QUEUES`
- New job types (e.g., `crypto.refresh_pricing`) must be documented
- Chunk size changes affect `pg_stat_statements` patterns

See the **queue-postgresql** skill for job type documentation patterns.

### Analytics Release Checklist

Copy this checklist when release notes reference measurable changes:
- [ ] Performance claims reference specific endpoints or queues
- [ ] New metrics/health endpoints are named
- [ ] Operator-facing env vars include defaults
- [ ] Queue changes specify new `WORKER_QUEUES` values
- [ ] Data coverage changes reference `dataset_slug` values

## Related References

- See the **queue-postgresql** skill for queue metrics documentation
- See the **cache-postgres** skill for cache-related release notes
- See `docs/deployment-operations.md` for operational metrics