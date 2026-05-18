# Spark Platform (Beaches MLS) — RESO replication and proxy

Quantyra IDX mirrors **Beaches MLS** listings from the Spark Platform RESO OData API and proxies live RESO requests for authenticated domains and API tokens.

## Credentials (server-side only)

| Variable | Purpose |
|----------|---------|
| `SPARK_ACCESS_TOKEN` | Bearer token for all Spark RESO HTTP (replication + live proxy) |
| `SPARK_API_FEED_ID` | OAuth / API Feed ID from the Spark dashboard (logging; not sent as Bearer) |
| `SPARK_RESO_BASE_URL` | OData root (default derived from host + root below) |

Legacy alias: `SPARK_API_KEY` maps to `SPARK_ACCESS_TOKEN` in `config/spark.php`.

## RESO base URL

Production replication responses use:

```http
https://replication.sparkapi.com/Reso/OData/
```

Spark also documents v3 at `https://replication.sparkapi.com/Version/3/Reso/OData/`. Set `SPARK_RESO_BASE_URL` to whichever responds to `$metadata` for your token.

Smoke test:

```bash
curl -sS -H "Authorization: Bearer $SPARK_ACCESS_TOKEN" -H "Accept: application/json" \
  "${SPARK_RESO_BASE_URL}/\$metadata" | head
```

## Catalog and mirror partition

| Catalog key (`?dataset=`) | Mirror `listings.dataset_slug` | Provider |
|-------------------------|--------------------------------|----------|
| `spark_beaches` | `beaches` | Spark |
| `beaches` (wire alias) | `beaches` | Spark |

Bridge feeds remain `bridge_{dataset}` (e.g. `bridge_stellar` → `stellar`).

## Replication rules

- **Host:** replication-enabled keys must use the replication RESO host ([Spark replication docs](https://sparkplatform.com/docs/reso/replication)).
- **Scope:** Active and Pending only — `StandardStatus eq 'Active' or StandardStatus eq 'Pending'`.
- **Page size:** `$top` max **1000** (config: `SPARK_SYNC_REPLICATION_TOP`).
- **Expand:** `$expand=Media,Unit,Room,OpenHouse` (no `$select` on replication).
- **Incremental:** dual-bound filter `ModificationTimestamp gt {cursor} and ModificationTimestamp lt {window_end}`; upper bound stored in `listing_sync_cursors.incremental_window_end`.
- **Pagination:** `@odata.nextLink` stored in `listing_sync_cursors.replication_next_url`.
- **Staging:** gzip pages in `bridge_replica_pages` with `provider = spark`.

Scheduled kickoff: `spark-listings-replica-sync` every 15 minutes (`SparkSyncJob` on `spark-sync-fetch`).

## Queues (Coolify worker)

Set worker env:

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

## Live API proxy

RESO Property/Member/Office/OpenHouse routes use `SparkClient` when the resolved feed is `spark_beaches`. Hybrid map/search reads the `beaches` mirror partition in Postgres.

Listing photos are rewritten to the idx-images host; the image proxy resolves `MediaURL` from expanded Property `Media` by `MediaKey`.

## Reference files

- [`docs/spark/beaches_metadata.xml`](spark/beaches_metadata.xml) — RESO metadata
- [`docs/spark/beaches_50_listings.json`](spark/beaches_50_listings.json) — sample replication page

## Dashboard

Domains may allow `spark_beaches` (or `beaches`) in **Allowed MLS datasets**. The dashboard shows the label **Beaches MLS (Spark)** with the internal code beneath.
