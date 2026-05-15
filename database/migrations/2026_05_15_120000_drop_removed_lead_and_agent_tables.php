<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * Drops tables that were previously created by removed migrations (agent portal,
 * lead saved searches / alerts, quantyra_leads). Safe on greenfield DBs where
 * these tables never existed. Run after deleting the old create migrations so
 * environments that already ran them can `migrate` forward without stale schema.
 *
 * Wrapped in a single transaction on PostgreSQL so a mid-run failure does not
 * leave a half-dropped dependency graph.
 */
return new class extends Migration
{
    public function up(): void
    {
        $callback = function (): void {
            Schema::dropIfExists('agent_activity_events');
            Schema::dropIfExists('agent_seo_landing_pages');
            Schema::dropIfExists('agent_contact_tags');
            Schema::dropIfExists('agent_portal_settings');
            Schema::dropIfExists('agent_share_links');
            Schema::dropIfExists('agent_automation_settings');
            Schema::dropIfExists('agent_alert_runs');
            Schema::dropIfExists('agent_alerts');
            Schema::dropIfExists('agent_alert_templates');
            Schema::dropIfExists('agent_search_geometries');
            Schema::dropIfExists('agent_search_filters');
            Schema::dropIfExists('agent_searches');
            Schema::dropIfExists('lookup_cache_snapshots');
            Schema::dropIfExists('field_mapping_adapters');
            Schema::dropIfExists('mls_field_catalog');
            Schema::dropIfExists('subscriber_feed_access');

            Schema::dropIfExists('lead_alert_settings');
            Schema::dropIfExists('lead_saved_searches');

            Schema::dropIfExists('quantyra_leads');
        };

        $driver = Schema::getConnection()->getDriverName();

        if ($driver === 'pgsql') {
            DB::transaction($callback);
        } else {
            $callback();
        }
    }

    public function down(): void
    {
        // Intentionally empty: removed feature tables are not recreated on rollback.
    }
};
