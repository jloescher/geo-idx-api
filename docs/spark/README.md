# Spark Platform (Beaches MLS) — documentation index

Quantyra integrates **BeachesMLS** through the [Spark Platform](https://sparkapi.com/) (FBS). This folder holds Beaches-specific reference material, compliance guides, and idx-api integration notes.

**Official Spark developer docs:** https://sparkplatform.com/docs/

---

## Documents in this folder

| Document | Audience | Description |
|----------|----------|-------------|
| [platform-overview.md](platform-overview.md) | All | Spark vs RESO Web API, API access, hosts (`sparkapi.com` vs `replication.sparkapi.com`), roles, rate limits |
| [idx-api-integration.md](idx-api-integration.md) | Backend / ops | Quantyra env vars, replication mirror, queues, live proxy, hybrid search, dashboard feeds |
| [reso-api-reference.md](reso-api-reference.md) | Integrators | RESO OData paths, replication query shape, catalog keys, smoke tests |
| [spark-compliance.md](spark-compliance.md) | Frontend / ops / legal | Listing display rules (Summary/Detail), disclaimers, IDX logos, engineering checklist |
| [reference-assets.md](reference-assets.md) | Backend | Local fixtures: metadata XML, sample JSON, license agreements |

---

## Quick reference

| Concept | Value |
|---------|--------|
| Catalog feed (`?dataset=`) | `spark_beaches` (alias `beaches`) |
| Postgres mirror partition | `beaches` |
| Replication RESO base | `https://replication.sparkapi.com/Reso/OData` |
| Live RESO base | `https://sparkapi.com/v1/Reso/OData` |
| Scheduled sync | `spark-listings-replica-sync` every 15 min |
| Worker queues | `spark-sync-fetch`, `spark-sync-persist` |

---

## Official Spark links

| Topic | URL |
|-------|-----|
| How to set up API access | https://sparkplatform.com/docs/overview/set_up_access |
| Spark API overview | https://sparkplatform.com/docs/overview/api |
| Which API is right for you? | https://sparkplatform.com/docs/overview/which_api |
| Listings service | https://sparkplatform.com/docs/api_services/listings |
| RESO Web API overview | https://sparkplatform.com/docs/reso/overview |
| RESO replication | https://sparkplatform.com/docs/reso/replication |
| Spark API replication (native) | https://sparkplatform.com/docs/supporting_documentation/replication |
| Listing display compliance | https://sparkplatform.com/docs/supporting_documentation/compliance |
| API support | api-support@fbsdata.com |

---

## Related project docs

| Document | Location |
|----------|----------|
| Docs index | [../INDEX.md](../INDEX.md) |
| Bridge proxy (Stellar) | [../idx-api-bridge-proxy.md](../idx-api-bridge-proxy.md) |
| Coolify deployment | [../coolify-deployment.md](../coolify-deployment.md) |
| Deployment & operations | [../deployment-operations.md](../deployment-operations.md) |

Legacy entry point [../spark-api-documentation.md](../spark-api-documentation.md) redirects here.
