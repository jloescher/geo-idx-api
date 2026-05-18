# Bridge API Documentation

## Overview

Bridge Data Output provides a comprehensive API platform for accessing real estate data, including property listings, market reports, agent information, and public data. The API is organized into several distinct services:

- **RESO Web API** - Standard real estate data following RESO standards
- **Bridge Web API** - Bridge's proprietary API format
- **Public Data** - Public parcel, assessment, and transaction data

Note: **Zestimates**, **Zillow Group Econ Data**, and **Zillow Agent Reviews** endpoints are documented by Bridge but **not available** through the Quantyra idx-api proxy.

Base URL: `https://bridgedataoutput.com`

**Quantyra implementation:** the secured MLS proxy, image masking, caching, and auth layers in **idx-api** are documented in **[idx-api-bridge-proxy.md](idx-api-bridge-proxy.md)** (this file remains vendor-style endpoint reference only).

## Dataset Configuration

**Default Dataset:** `stellar`

The API endpoints documented below use the `stellar` dataset by default. The idx-api proxy supports **multiple MLS datasets** (e.g., `stellar`, `miami`) with per-domain access control via `allowed_mls_datasets`.

When making requests:
- Pass `?mls_dataset=<dataset>` query parameter
- For search: include `"mls_dataset": "<dataset>"` in the JSON body
- If omitted, the domain's default `mls_dataset` is used
- Returns **403** if the requested dataset is not in the domain's allowed list

---

## RESO Web API

The RESO Web API follows the Real Estate Standards Organization (RESO) Data Dictionary standard, providing standardized access to real estate data.

### Property

#### GET /stellar/Property

Retrieves a collection of properties from the stellar dataset.

**Response:** Collection of property records

---

#### GET /stellar/Property('{ListingKey}')

Retrieves a specific property by its listing key.

**Parameters:**
- `ListingKey` (path) - The unique identifier for the property listing

**Response:** Single property record

---

### Member

#### GET /stellar/Member

Retrieves a collection of members (agents) from the stellar dataset.

**Response:** Collection of member records

---

#### GET /stellar/Member('{MemberKey}')

Retrieves a specific member by their key.

**Parameters:**
- `MemberKey` (path) - The unique identifier for the member

**Response:** Single member record

---

### Office

#### GET /stellar/Office

Retrieves a collection of offices from the stellar dataset.

**Response:** Collection of office records

---

#### GET /stellar/Office('{OfficeKey}')

Retrieves a specific office by its key.

**Parameters:**
- `OfficeKey` (path) - The unique identifier for the office

**Response:** Single office record

---

### OpenHouse

#### GET /stellar/OpenHouse

Retrieves a collection of open house events from the stellar dataset.

**Response:** Collection of open house records

---

#### GET /stellar/OpenHouse('{OpenHouseKey}')

Retrieves a specific open house by its key.

**Parameters:**
- `OpenHouseKey` (path) - The unique identifier for the open house event

**Response:** Single open house record

---

### Lookup

#### GET /stellar/Lookup

Retrieves lookup values for standardized field values from the stellar dataset.

**Response:** Collection of lookup values

---

## Bridge Web API

Bridge's proprietary API provides simplified access to real estate data with a more intuitive structure.

### Listings

#### GET /stellar/listings

Retrieves a collection of property listings from the stellar dataset.

**Response:** Collection of listing records

---

#### GET /stellar/listings/{listingId}

Retrieves a specific listing by its ID.

**Parameters:**
- `listingId` (path) - The unique identifier for the listing

**Response:** Single listing record

---

### Agents

#### GET /stellar/agents

Retrieves a collection of agents from the stellar dataset.

**Response:** Collection of agent records

---

#### GET /stellar/agents/{agentId}

Retrieves a specific agent by their ID.

**Parameters:**
- `agentId` (path) - The unique identifier for the agent

**Response:** Single agent record

---

### Offices

#### GET /stellar/offices

Retrieves a collection of offices from the stellar dataset.

**Response:** Collection of office records

---

#### GET /stellar/offices/{officeId}

Retrieves a specific office by its ID.

**Parameters:**
- `officeId` (path) - The unique identifier for the office

**Response:** Single office record

---

### Open Houses

#### GET /stellar/openhouses

Retrieves a collection of open house events from the stellar dataset.

**Response:** Collection of open house records

---

#### GET /stellar/openhouses/{openhouseId}

Retrieves a specific open house by its ID.

**Parameters:**
- `openhouseId` (path) - The unique identifier for the open house event

**Response:** Single open house record

---

## Public Data

Access public parcel, assessment, and transaction data without requiring dataset-specific access.

### Parcels

#### GET /pub/parcels

Retrieves a collection of parcel records.

**Response:** Collection of parcel records

---

#### GET /pub/parcels/{parcelId}

Retrieves a specific parcel by its ID.

**Parameters:**
- `parcelId` (path) - The unique identifier for the parcel

**Response:** Single parcel record

---

#### GET /pub/parcels/{parcelId}/assessments

Retrieves assessment records for a specific parcel.

**Parameters:**
- `parcelId` (path) - The unique identifier for the parcel

**Response:** Collection of assessment records for the parcel

---

#### GET /pub/parcels/{parcelId}/transactions

Retrieves transaction records for a specific parcel.

**Parameters:**
- `parcelId` (path) - The unique identifier for the parcel

**Response:** Collection of transaction records for the parcel

---

### Assessments

#### GET /pub/assessments

Retrieves a collection of assessment records.

**Response:** Collection of assessment records

---

### Transactions

#### GET /pub/transactions

Retrieves a collection of transaction records.

**Response:** Collection of transaction records


## Authentication

The Bridge API requires authentication. Please refer to the Bridge Data Output documentation for authentication details and API key management.

## Rate Limits

Bridge Data Output enforces per-token quotas on RESO Web API traffic. idx-api replica sync and proxy code assume the following (see also [idx-api-bridge-proxy.md](idx-api-bridge-proxy.md)):

| Rule | Limit | idx-api behavior |
|------|--------|------------------|
| Standard `Property` OData | `$top` max **200**; paginate with `$skip` | Incremental sync only; **>10,000** rows via `$skip` requires replication catch-up |
| `Property/replication` | `$top` max **2,000**; **no** `$skip` / `$orderby` | Follow `Link: rel="next"` or `@odata.nextLink` only |
| Hourly quota | **5,000 requests/hour** per token | Default proactive cap **4800 req/hour** (`BRIDGE_SYNC_MAX_REQUESTS_PER_HOUR`) plus **280 req/min** (`BRIDGE_SYNC_MAX_REQUESTS_PER_MINUTE`) under the **334/min** burst ceiling |
| Burst | **334 requests/minute** (1/15 of hourly) | `BridgeRateLimitGuard` on every server Bridge GET (sync, MLS cache, proxy); **no** throttle on Postgres persist jobs (`bridge-sync-persist` queue) |
| Response headers | `Application-RateLimit-*`, `Burst-RateLimit-*` | Parsed after each GET; HTTP **429** retried via `BRIDGE_SYNC_MAX_HTTP_RETRIES` |

Official policy details: [Bridge RESO Web API explorer](https://bridgedataoutput.com/docs/explorer/reso-web-api).

## Additional Resources

- Official Documentation: [https://bridgedataoutput.com/docs/explorer/reso-web-api](https://bridgedataoutput.com/docs/explorer/reso-web-api)
- API Explorer: Available on the Bridge Data Output website for interactive testing

## Notes

- All endpoints use the GET HTTP method (POST for structured search via idx-api proxy)
- **Supported datasets:** `stellar`, `miami`, and others (configured per-domain in the proxy)
- RESO Web API endpoints follow OData conventions with automatic cursor pagination support
- Bridge Web API provides a simplified, RESTful interface
- Public Data endpoints are accessible without dataset-specific credentials

## Related documentation

| Document | Topic |
|----------|--------|
| [idx-api-bridge-proxy.md](idx-api-bridge-proxy.md) | Secured proxy implementation, caching, auth, image rewriting, search endpoint, dataset gates. |
| [api.md](api.md) | idx-api HTTP API overview, obtaining Bearer tokens. |
| [gis-api.md](gis-api.md) | GIS parcel/geometry proxy (public data, not MLS). |