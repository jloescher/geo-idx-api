<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('listing_sync_cursors', function (Blueprint $table): void {
            $table->timestampTz('incremental_window_end')->nullable()->after('last_bridge_modification_timestamp');
        });

        Schema::table('bridge_replica_pages', function (Blueprint $table): void {
            $table->string('provider', 16)->default('bridge')->after('dataset_slug');
            $table->text('upstream_url')->nullable()->after('bridge_url');
        });

        Schema::table('bridge_replica_pages', function (Blueprint $table): void {
            $table->index(['provider', 'dataset_slug', 'status'], 'bridge_replica_pages_provider_dataset_status_idx');
        });
    }

    public function down(): void
    {
        Schema::table('bridge_replica_pages', function (Blueprint $table): void {
            $table->dropIndex('bridge_replica_pages_provider_dataset_status_idx');
            $table->dropColumn(['provider', 'upstream_url']);
        });

        Schema::table('listing_sync_cursors', function (Blueprint $table): void {
            $table->dropColumn('incremental_window_end');
        });
    }
};
