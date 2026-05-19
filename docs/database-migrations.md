# Database migrations (PostgreSQL)

Canonical inventory for **`database/migrations/`** (Laravel 13). All application schema lives in this single directory (no secondary `loadMigrationsFrom()` paths).

For a **fresh database**, run `php artisan migrate` (or `migrate:fresh` on disposable dev/test DBs only). Legacy incremental migrations and the old `drop_removed_lead_and_agent_tables` cleanup were consolidated into the files below.

---

## Migration inventory (run order)

| File | Tables / purpose |
|------|------------------|
| `0001_01_01_000000_create_users_table.php` | `users` (Fortify 2FA, widget embed, MLS membership, `is_admin`), `password_reset_tokens`, `sessions` |
| `0001_01_01_000001_create_cache_table.php` | Laravel `cache`, `cache_locks` |
| `0001_01_01_000002_create_jobs_table.php` | `jobs`, `job_batches`, `failed_jobs` |
| `2026_01_01_100000_create_idx_auth_tables.php` | `personal_access_tokens`, `user_invitations` |
| `2026_01_01_200000_create_idx_domain_and_bridge_cache_tables.php` | `domains`, `bridge_search_cache`, `listings_cache`, `bridge_proxy_audit_logs` |
| `2026_01_01_300000_create_gis_tables.php` | `gis_cache`, `gis_source_states` (+ seed rows for FL parcel sources) |
| `2026_01_01_400000_create_crypto_price_snapshots_table.php` | `crypto_price_snapshots` |
| `2026_01_01_500000_create_listings_mirror_tables.php` | PostGIS `listings`, `listing_sync_cursors`, `replica_pages` (Bridge + Spark staging) |
| `2026_04_23_144258_create_telescope_entries_table.php` | Laravel Telescope (diagnostics) |
| `2026_04_23_144551_create_pulse_tables.php` | Laravel Pulse |

**9 migration files** total (3 framework + 5 IDX domain + 2 observability).

---

## Operational notes

### PostGIS

[`2026_01_01_500000_create_listings_mirror_tables.php`](../database/migrations/2026_01_01_500000_create_listings_mirror_tables.php) enables PostGIS when missing (`CREATE EXTENSION IF NOT EXISTS postgis`). On managed Postgres without superuser, create the extension once before migrating.

**Mirror scope:** replication stores **Active + Pending** in `listings`; **Closed** is on-demand via Bridge. See [IDX-API Bridge proxy](idx-api-bridge-proxy.md).

### Fresh install

```bash
php artisan migrate --force
```

Use a dedicated database for **`migrate:fresh`** in CI/local dev (`testing` or `idx_api_testing`; see `tests/TestCase.php`).

### Upgrading existing databases

If an environment already ran migrations that created `bridge_replica_pages` or `2026_05_18_155700_extend_listing_sync_and_replica_pages_for_spark.php`, do **not** swap migration files in place. **Reset** to a fresh database (`migrate:fresh` on disposable dev/staging only) and run the consolidated set above.

---

## Related docs

- [Deployment & operations](deployment-operations.md)
- [Coolify deployment](coolify-deployment.md)
- [IDX-API Bridge proxy](idx-api-bridge-proxy.md)
