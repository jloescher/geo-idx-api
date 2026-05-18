# Spark Platform overview

[FBS Spark](https://sparkapi.com/) powers Flexmls-backed MLS data APIs. **BeachesMLS** (BEACHMLS, INC.) is licensed to Quantyra through an **IDX**-labeled API feed on the Spark Platform.

This page summarizes how Spark documents its platform. For Quantyra wiring (env, mirror, proxy), see [idx-api-integration.md](idx-api-integration.md).

---

## Two APIs, one credential

Spark offers two read-oriented APIs that share **authentication** (Bearer access token from the Spark dashboard):

| API | Best for | Wire format |
|-----|----------|-------------|
| **[Spark API](https://sparkplatform.com/docs/overview/api)** | Single-MLS apps, rich Flexmls features (contacts, market stats, native listings) | JSON envelope `D.Success.Results` with `StandardFields` |
| **[RESO Web API](https://sparkplatform.com/docs/reso/overview)** | Multi-MLS portability, RESO Data Dictionary field names | OData JSON under `/Reso/OData` |

Quantyra idx-api uses **RESO OData** for Beaches so clients can share the same Property/Member/Office patterns as Bridge (Stellar). See [Which API is right for you?](https://sparkplatform.com/docs/overview/which_api).

Both APIs support **real-time** requests and **replication** (bulk sync with pagination). Replication requires an API key with replication permission.

---

## Production vs replication hosts

| Host | Purpose | Example |
|------|---------|---------|
| **`https://sparkapi.com`** | Live API (IDX proxy, search, photos, closed listings) | `GET /v1/Reso/OData/Property` |
| **`https://replication.sparkapi.com`** | Bulk replication only | `GET /Reso/OData/Property?$top=1000&...` |

**Critical rule:** API keys with replication access **must** use `replication.sparkapi.com` for bulk sync. Replication requests to `sparkapi.com` **fail** ([Spark API replication](https://sparkplatform.com/docs/supporting_documentation/replication)).

idx-api enforces this split in application code — see [idx-api-integration.md](idx-api-integration.md#api-hosts).

---

## API access setup (official process)

From [How to Set Up API Access](https://sparkplatform.com/docs/overview/set_up_access):

1. **Register** as a Spark API developer (free; demo credentials in ~3 business days).
2. **Choose role** — for consumer-facing IDX sites, typically **IDX** (also Public, VOW, Portal, Private).
3. **Spark Datamart** — search for the MLS and enroll in a data plan that issues the correct role.
4. If no suitable plan exists, **contact the MLS** with: what you are building, who sees data, required role, and associated Flexmls user (or generic agent key request).

Support: **api-support@fbsdata.com**

### BeachesMLS feed (Quantyra)

Configure in server `.env` only (never commit secrets):

- **Access Token** → `SPARK_ACCESS_TOKEN` (Bearer)
- **API Feed ID** → `SPARK_API_FEED_ID` (dashboard identity; not the Bearer token)
- Designated usage URL should match platform host (e.g. `https://idx.quantyralabs.cc`)

Confirm with Spark whether the key has **replication** permission if running the Postgres mirror.

---

## API URIs and versioning

From [API Overview (read first)](https://sparkplatform.com/docs/api_services/read_first):

- Production API: **`https://sparkapi.com/`**
- Listings (Spark API v1): **`https://sparkapi.com/v1/listings`**
- Authentication / OpenID: **`https://sparkplatform.com`**
- Latest Spark API version path: **`/v1`**

RESO Web API is nested under the Spark API (e.g. `/v1/Reso/OData` on `sparkapi.com` for live traffic).

---

## Roles (summary)

| Role | Reads | Typical use |
|------|-------|-------------|
| IDX | Yes | Public consumer IDX sites |
| Public | Yes | Unauthenticated-style public display (policy-defined) |
| VOW | Yes | Logged-in consumers; more fields than IDX |
| Portal | Yes | Agent customer portals |
| Private | Yes | MLS member / back-office style apps |

Role determines visible fields and listing statuses. [Roles documentation](https://sparkplatform.com/docs/supporting_documentation/roles).

---

## Rate limits

From Spark API overview:

| Key type | Limit |
|----------|-------|
| IDX | 1,500 requests per 5 minutes |
| VOW / Broker Back Office | 4,000 requests per 5 minutes |

Exceeded limits return HTTP **429**. Replication pagination counts each page as a separate request.

idx-api uses conservative sync rate limits (`SPARK_SYNC_MAX_REQUESTS_PER_SECOND`, default 2) on replication fetch jobs.

---

## Listing display compliance

All listing UIs must follow MLS **DisplayCompliance** rules — Summary vs Detail views, required fields, disclaimers, IDX logos. See [spark-compliance.md](spark-compliance.md) and Spark’s [Listing Data Display Rules](https://sparkplatform.com/docs/supporting_documentation/compliance).

The [Listings service](https://sparkplatform.com/docs/api_services/listings) explicitly references the compliance page for every display surface.

---

## Replication best practices (Spark)

From [Spark API replication](https://sparkplatform.com/docs/supporting_documentation/replication):

- Initial download: `_limit=1000`, paginate with `_skiptoken` (preferred over `_skip`).
- Incremental updates: dual-bound `ModificationTimestamp` filter (`gt` and `lt`) — not a single open-ended `gt`.
- Purge stale keys daily (listing keys no longer returned for your role).
- Replicate **Accounts** for agent/office display; listing payloads may embed stale agent fields.

Quantyra mirror implementation details: [idx-api-integration.md](idx-api-integration.md#replication-pipeline).

---

## Terms and local license files

On-file agreements in this directory (not substitutes for legal review):

| File | Description |
|------|-------------|
| [MLS Data License.txt](MLS%20Data%20License.txt) | MLS data license terms |
| [Data Access Agreement.txt](Data%20Access%20Agreement.txt) | Firm/consultant data access |
| PDFs | Spark/FBS consumer, developer, privacy, and store terms |

Spark canonical terms: https://sparkplatform.com/docs/terms_of_use/
