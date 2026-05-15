<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\QueryException;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * Revenue impact: Postgres jsonb + partial GiST/BRIN accelerate map pan/teaser payloads so IDX pages
 * keep sub-50ms local search without extra Bridge OData ($).
 *
 * Compliance: mirror scope is IDX-facing Active/Pending only; Closed rows excluded at ingest —
 * stale Closed keys are reconciled via delete batch (see BridgeSyncService).
 */
return new class extends Migration
{
    public function up(): void
    {
        $driver = Schema::connection($this->getConnection())->getConnection()->getDriverName();

        if ($driver === 'pgsql') {
            $row = DB::selectOne('select exists(select 1 from pg_extension where extname = ?) as installed', ['postgis']);
            $postgisInstalled = (bool) ($row->installed ?? false);

            if (! $postgisInstalled) {
                try {
                    DB::statement('CREATE EXTENSION IF NOT EXISTS postgis');
                } catch (QueryException $e) {
                    throw new RuntimeException(
                        'PostgreSQL PostGIS is required for listings geography columns. Install it with a superuser '.
                        '(CREATE EXTENSION postgis) or use a database where PostGIS is already enabled, then re-run migrations.',
                        0,
                        $e
                    );
                }
            }
        }

        Schema::create('listings', function (Blueprint $table) use ($driver): void {
            $table->id();
            $table->string('dataset_slug', 64);
            $table->string('listing_key', 255);
            $table->string('mls_listing_id', 128)->nullable();
            $table->string('standard_status', 50)->nullable();

            $table->decimal('list_price', 14, 2)->nullable();
            $table->unsignedSmallInteger('bedrooms_total')->nullable();
            $table->decimal('bathrooms_total_decimal', 5, 2)->nullable();
            $table->unsignedInteger('living_area')->nullable();
            $table->decimal('lot_size_acres', 12, 4)->nullable();
            $table->unsignedSmallInteger('year_built')->nullable();
            $table->unsignedSmallInteger('stories_total')->nullable();

            $table->string('city', 120)->nullable();
            $table->string('county_or_parish', 120)->nullable();
            $table->string('postal_code', 20)->nullable();
            $table->string('state_or_province', 30)->nullable();

            $table->string('property_type', 80)->nullable();
            $table->string('property_sub_type', 80)->nullable();

            $table->date('on_market_date')->nullable();
            $table->date('close_date')->nullable();
            $table->timestampTz('modification_timestamp')->nullable();
            $table->timestampTz('bridge_modification_timestamp')->nullable();
            $table->timestampTz('price_change_timestamp')->nullable();

            $table->decimal('previous_list_price', 14, 2)->nullable();

            $table->string('stellar_flood_zone_code', 80)->nullable();
            $table->decimal('stellar_total_monthly_fees', 14, 2)->nullable();

            $table->double('latitude')->nullable();
            $table->double('longitude')->nullable();

            $table->boolean('waterfront_yn')->nullable();
            $table->boolean('pool_private_yn')->nullable();
            $table->boolean('dock_yn')->nullable();
            $table->boolean('new_construction_yn')->nullable();
            $table->boolean('garage_yn')->nullable();
            $table->boolean('association_yn')->nullable();
            $table->boolean('spa_yn')->nullable();
            $table->boolean('fireplace_yn')->nullable();
            $table->boolean('senior_community_yn')->nullable();

            $table->string('subdivision_name', 160)->nullable();
            $table->string('elementary_school', 160)->nullable();
            $table->string('middle_or_junior_school', 160)->nullable();
            $table->string('high_school', 160)->nullable();

            if ($driver === 'pgsql') {
                $table->jsonb('special_listing_conditions')->nullable();
                $table->jsonb('raw_data')->nullable();
                $table->jsonb('custom_fields')->nullable();
                $table->geography('coordinates', 'point', 4326)->nullable();
            } else {
                $table->json('special_listing_conditions')->nullable();
                $table->json('raw_data')->nullable();
                $table->json('custom_fields')->nullable();
            }

            $table->string('street_number', 40)->nullable();
            $table->string('street_name', 160)->nullable();
            $table->string('list_agent_mls_id', 64)->nullable();
            $table->string('list_office_mls_id', 64)->nullable();

            $table->timestamps();

            $table->unique(['dataset_slug', 'listing_key']);
            $table->index(['dataset_slug', 'listing_key']);
        });

        if ($driver === 'pgsql') {
            $ap = "LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')";

            DB::statement("CREATE INDEX IF NOT EXISTS listings_ap_geom_gix ON listings USING GIST (coordinates) WHERE coordinates IS NOT NULL AND {$ap}");
            DB::statement("CREATE INDEX IF NOT EXISTS listings_ap_mod_brin ON listings USING BRIN (modification_timestamp) WHERE {$ap}");
            DB::statement("CREATE INDEX IF NOT EXISTS listings_ap_ds_price_idx ON listings (dataset_slug, list_price) WHERE {$ap}");
            DB::statement("CREATE INDEX IF NOT EXISTS listings_ap_ds_beds_idx ON listings (dataset_slug, bedrooms_total) WHERE {$ap}");
            DB::statement("CREATE INDEX IF NOT EXISTS listings_ap_ds_mod_ts_idx ON listings (dataset_slug, modification_timestamp DESC) WHERE {$ap} AND modification_timestamp IS NOT NULL");
        } else {
            Schema::table('listings', function (Blueprint $table): void {
                $table->index(['dataset_slug', 'list_price'], 'listings_ds_price_sqlite_idx');
                $table->index(['dataset_slug', 'bedrooms_total'], 'listings_ds_beds_sqlite_idx');
                $table->index(['dataset_slug', 'modification_timestamp'], 'listings_ds_mod_sqlite_idx');
            });
        }

        Schema::create('listing_sync_cursors', function (Blueprint $table): void {
            $table->string('dataset_slug', 64)->primary();
            $table->timestampTz('last_bridge_modification_timestamp')->nullable();
            $table->text('replication_next_url')->nullable();
            $table->boolean('replication_in_progress')->default(false);
            $table->timestampTz('last_sync_finished_at')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('listing_sync_cursors');

        $driver = Schema::connection($this->getConnection())->getConnection()->getDriverName();
        if ($driver === 'pgsql') {
            DB::statement('DROP INDEX IF EXISTS listings_ap_ds_mod_ts_idx');
            DB::statement('DROP INDEX IF EXISTS listings_ap_ds_beds_idx');
            DB::statement('DROP INDEX IF EXISTS listings_ap_ds_price_idx');
            DB::statement('DROP INDEX IF EXISTS listings_ap_mod_brin');
            DB::statement('DROP INDEX IF EXISTS listings_ap_geom_gix');
        }

        Schema::dropIfExists('listings');
    }
};
