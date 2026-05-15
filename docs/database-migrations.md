# Database migrations (PostgreSQL)

Canonical inventory for **`database/migrations/`** (Laravel 13). Billing / CRM-specific GHL tables are **not** defined in this repository; all schema for this service lives in the paths below.

---

## Layout

| Path | Role |
|------|------|
| [`database/migrations/`](../database/migrations/) | All migrations the app runs (`php artisan migrate`). |

---

## Migration inventory (chronological)

| File | Purpose |
|------|---------|
| `0001_01_01_000000_create_users_table.php` | `users`, `password_reset_tokens`, `sessions` |
| `0001_01_01_000001_create_cache_table.php` | Framework cache store |
| `0001_01_01_000002_create_jobs_table.php` | Queues / failed jobs |
| `2026_04_22_115800_add_quantyra_user_profile_columns_to_users_table.php` | Fortify 2FA, widget embed key, MLS membership fields, `widget_palette` |
| `2026_04_22_120000_create_personal_access_tokens_table.php` | Sanctum PATs (`idx:access` / `idx:full`) |
| `2026_04_22_120100_create_domains_table.php` | `domains` + `bridge_search_cache` (compressed Bridge search payloads) |
| `2026_04_22_120200_create_listings_cache_table.php` | Row-level `listings_cache` (`compressed_payload` per listing key) |
| `2026_04_22_120300_create_bridge_proxy_audit_logs_table.php` | MLS proxy audit rows |
| `2026_04_23_144258_create_telescope_entries_table.php` | Laravel Telescope (dev/diagnostics) |
| `2026_04_23_144551_create_pulse_tables.php` | Laravel Pulse |
| `2026_04_24_010000_create_gis_cache_table.php` | GIS parcel cache (`query_hash`, `source_generation`) |
| `2026_04_24_120000_create_gis_source_states_table.php` | Per-source generation / fingerprint for cache invalidation |
| `2026_04_26_131500_create_crypto_price_snapshots_table.php` | Cached FX / crypto quotes for listing enrichment |
| `2026_04_30_210000_create_listings_and_sync_cursors_tables.php` | PostGIS `listings` mirror + `listing_sync_cursors` |
| `2026_05_15_120000_drop_removed_lead_and_agent_tables.php` | **Cleanup only:** `dropIfExists` for legacy agent / saved-search / `quantyra_leads` tables if a database still has them after older migrations were removed from the repo |

---

## Operational notes

### PostGIS

[`2026_04_30_210000_create_listings_and_sync_cursors_tables.php`](../database/migrations/2026_04_30_210000_create_listings_and_sync_cursors_tables.php) requires the **PostGIS** extension for geography columns. The migration skips `CREATE EXTENSION` when PostGIS is already installed; otherwise a superuser must run `CREATE EXTENSION postgis` once on the database (typical on RDS/Coolify managed Postgres).

### Legacy table drop migration

`2026_05_15_120000_drop_removed_lead_and_agent_tables.php` is safe on **new** databases (no-op `dropIfExists`). On databases that previously ran removed migrations, it removes leftover tables in **FK-safe order**; on PostgreSQL the drops run inside a **single transaction** so a failure does not leave a half-removed graph.

### Deploy

```bash
php artisan migrate --force
```

Use a dedicated disposable database for **`migrate:fresh`** in CI and local dev (see `phpunit.xml` / `tests/TestCase.php` for allowed test DB names).

---

## Related docs

- [Deployment & operations](deployment-operations.md) — migrate, workers, scheduler.
- [Coolify deployment](coolify-deployment.md) — production/staging layout.
- [IDX-API Bridge proxy](idx-api-bridge-proxy.md) — cache and audit behavior at the application layer.
