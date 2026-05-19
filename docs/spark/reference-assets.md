# Spark / Beaches — reference assets

Local files in `docs/spark/` used for development, testing, and compliance review. **Do not commit live API secrets**; tokens belong in `.env` only.

---

## RESO metadata

| File | Description |
|------|-------------|
| [beaches_metadata.xml](beaches_metadata.xml) | OData `$metadata` export for Beaches RESO entity types, standard and encoded custom properties |

Use to look up field types, navigation properties (`Media`, `Room`, `Unit`, `OpenHouse`), and `MLS.OData.Metadata.LocalName` labels for encoded names.

---

## Sample payload

| File | Description |
|------|-------------|
| [beaches_50_listings.json](beaches_50_listings.json) | One replication page (`$top=50`) with `$expand=Media,Unit,Room,OpenHouse` and Active/Pending filter |

Used by:

- `tests/Feature/Spark/ListingMirrorWriterTest` — persist smoke test (`estimated_total_monthly_fees` from first row: `AssociationFee` Monthly + null-frequency `AssociationFee2`)
- Manual inspection of field shapes, association-fee frequencies, and `MediaURL` patterns

Upstream context URL in sample: `https://replication.sparkapi.com/Reso/OData/Property`.

---

## License and policy documents

| File | Format |
|------|--------|
| [MLS Data License.txt](MLS%20Data%20License.txt) | Text |
| [Data Access Agreement.txt](Data%20Access%20Agreement.txt) | Text |
| Consumer Terms of Use - Spark.pdf | PDF |
| Developer Agreement and Spark API Terms of Use - Spark.pdf | PDF |
| FBS Privacy Policy - Spark.pdf | PDF |
| MLS Member Terms of Use - Spark.pdf | PDF |
| Spark Store Terms of Use - Spark.pdf | PDF |

Canonical online terms: https://sparkplatform.com/docs/terms_of_use/

Compliance mapping: [spark-compliance.md](spark-compliance.md).

---

## Regenerating fixtures

**Metadata** (requires valid token):

```bash
curl -sS -H "Authorization: Bearer $SPARK_ACCESS_TOKEN" \
  "https://replication.sparkapi.com/Reso/OData/\$metadata" \
  -o docs/spark/beaches_metadata.xml
```

**Sample listings page** (adjust `$top` as needed):

```bash
curl -sS -H "Authorization: Bearer $SPARK_ACCESS_TOKEN" -H "Accept: application/json" \
  "https://replication.sparkapi.com/Reso/OData/Property?\$top=50&\$expand=Media,Unit,Room,OpenHouse&\$filter=StandardStatus+eq+'Active'+or+StandardStatus+eq+'Pending'" \
  | jq . > docs/spark/beaches_50_listings.json
```

Only refresh fixtures when MLS policy allows storing exported data in the repo.
