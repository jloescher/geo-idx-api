# Spark RESO Web API — Beaches reference

RESO OData reference for **BeachesMLS** via Spark. Official upstream: [RESO Web API overview](https://sparkplatform.com/docs/reso/overview), [Property](https://sparkplatform.com/docs/reso/properties), [replication](https://sparkplatform.com/docs/reso/replication).

Quantyra idx-api exposes the same OData shapes through `/api/v1/*` when `?dataset=spark_beaches`. See [idx-api-integration.md](idx-api-integration.md).

---

## Base URLs

| Mode | Base URL | idx-api usage |
|------|----------|---------------|
| **Replication** | `https://replication.sparkapi.com/Reso/OData` | Mirror sync only |
| **Live** | `https://sparkapi.com/v1/Reso/OData` | Proxy, hybrid live search, images |
| **Live (v3 fallback)** | `https://sparkapi.com/v1/Version/3/Reso/OData` | Set `SPARK_LIVE_RESO_ROOT=Version/3/Reso/OData` if v1 path 404s |

Authentication: `Authorization: Bearer {SPARK_ACCESS_TOKEN}`

---

## Core resources

| Resource | Collection | Entity |
|----------|------------|--------|
| Property | `GET .../Property` | `GET .../Property('{ListingKey}')` |
| Member | `GET .../Member` | `GET .../Member('{MemberKey}')` |
| Office | `GET .../Office` | `GET .../Office('{OfficeKey}')` |
| OpenHouse | `GET .../OpenHouse` | `GET .../OpenHouse('{OpenHouseKey}')` |
| Lookup | `GET .../Lookup` | — |
| Metadata | `GET .../$metadata` | XML |
| Lookup enumerations | `GET .../Lookup` | Per [Spark Lookup docs](https://sparkplatform.com/docs/reso/lookup) |

idx-api proxy routes (same as Bridge): `/api/v1/properties`, `/api/v1/members`, `/api/v1/offices`, `/api/v1/openhouses`, `/api/v1/lookup`, etc.

---

## Replication query (mirror)

Initial and paged replication (Active/Pending only):

```http
GET https://replication.sparkapi.com/Reso/OData/Property
  ?$top=1000
  &$expand=Media,Unit,Room,OpenHouse
  &$filter=StandardStatus eq 'Active' or StandardStatus eq 'Pending'
```

Follow `@odata.nextLink` as returned (stays on replication host).

**Incremental** (dual-bound window):

```http
GET .../Property
  ?$top=1000
  &$expand=Media,Unit,Room,OpenHouse
  &$filter=(StandardStatus eq 'Active' or StandardStatus eq 'Pending')
    and ModificationTimestamp gt 2024-07-01T00:00:00Z
    and ModificationTimestamp lt 2024-07-02T00:00:00Z
```

idx-api stores the upper bound in `listing_sync_cursors.incremental_window_end`.

---

## Live queries (proxy)

**Collection search** (Summary-style in compliance terms):

```http
GET https://sparkapi.com/v1/Reso/OData/Property
  ?$filter=...
  &$orderby=...
  &$top=50
  &$skip=0
```

**Single listing** (Detail-style):

```http
GET https://sparkapi.com/v1/Reso/OData/Property('20240712154755555836000000')
```

**Media for image proxy:**

```http
GET https://sparkapi.com/v1/Reso/OData/Property('20240712154755555836000000')?$expand=Media
```

Match `MediaKey` in expanded `Media` array to resolve `MediaURL` (often `cdn.photos.sparkplatform.com`).

---

## idx-api dataset parameter

| Request | Effect |
|---------|--------|
| `?dataset=spark_beaches` | Spark live host, beaches mirror for hybrid |
| `?dataset=beaches` | Normalized to `spark_beaches` |
| Domain default `mls_dataset` | Must be in `allowed_mls_datasets` |

Returns **403** if feed not enabled for domain/token.

---

## Field notes (Beaches)

- Standard RESO fields per [beaches_metadata.xml](beaches_metadata.xml).
- `BathroomsTotalInteger` and `BathroomsTotalDecimal` — mirror uses decimal for `bathrooms_total_decimal`.
- `LivingArea` — indexed as `living_area`; falls back to `BuildingAreaTotal` when `LivingArea` is missing.
- `ModificationTimestamp` — cursor driver (no `BridgeModificationTimestamp`).
- Encoded custom field names (`*_sp_*`, `*_co_*`) and other unmapped RESO keys — stored in `listings.custom_fields` at persist; **flat-merged** onto the root Property object in mirror-backed API responses (not returned as a nested `custom_fields` key). Human labels in metadata `MLS.OData.Metadata.LocalName` annotations.
- `Media` expanded on replication (`$expand=Media`) — photo array in `listings.media`; property JSON (minus `Media`) in `raw_data`. Sample shape: [beaches_50_listings.json](beaches_50_listings.json).

### Normalized mirror columns (idx-api)

| `listings` column | Beaches RESO inputs |
|-------------------|---------------------|
| `flood_zone_code` | `Location_sp_and_sp_Legal_co_Flood_sp_Zone2` |
| `estimated_total_monthly_fees` | `AssociationFee` + `AssociationFeeFrequency`, `AssociationFee2` + `AssociationFee2Frequency` (monthly equivalent sum; see [idx-api-integration.md](idx-api-integration.md#normalized-mirror-columns-persist--replication-updates)) |

**Association fee frequencies:** `Monthly`, `Annually`, `Semi-Annually`, `Quarterly`, `Weekly`, `Daily`, `One Time` (exact MLS strings). Null frequency or `One Time` does not contribute to the monthly total.

Sample replication page: [beaches_50_listings.json](beaches_50_listings.json).

---

## Spark native API (not used by idx-api proxy)

The [Spark Listings API](https://sparkplatform.com/docs/api_services/listings) at `https://sparkapi.com/v1/listings` uses a different JSON envelope (`D.Success.Results`, `StandardFields`). idx-api does **not** translate this format; clients use RESO OData through idx-api.

Use native API documentation when building tools that call Spark directly outside idx-api.

---

## Related

- [platform-overview.md](platform-overview.md) — hosts and access setup
- [spark-compliance.md](spark-compliance.md) — display rules
- [idx-api-integration.md](idx-api-integration.md) — implementation
