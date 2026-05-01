<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('subscriber_feed_access', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('mls_code', 64)->index();
            $table->string('feed_id', 128)->index();
            $table->string('dataset_code', 128)->index();
            $table->string('status', 32)->default('active')->index();
            $table->json('permissions_json')->nullable();
            $table->timestamp('connected_at')->nullable();
            $table->timestamp('last_verified_at')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'feed_id', 'dataset_code'], 'subscriber_feed_access_unique');
        });

        Schema::create('mls_field_catalog', function (Blueprint $table): void {
            $table->id();
            $table->string('mls_code', 64)->index();
            $table->string('dataset_code', 128)->index();
            $table->string('source_field_key', 256)->index();
            $table->string('canonical_field_key', 256)->nullable()->index();
            $table->string('display_label', 512);
            $table->string('field_type', 32)->index();
            $table->json('operators_json')->nullable();
            $table->json('enum_values_json')->nullable();
            $table->boolean('is_reso_standard')->default(false);
            $table->boolean('is_custom_mls_field')->default(false);
            $table->json('compatibility_tags_json')->nullable();
            $table->string('lookup_version', 64)->nullable();
            $table->timestamps();

            $table->unique(['mls_code', 'dataset_code', 'source_field_key'], 'mls_field_catalog_unique');
        });

        Schema::create('field_mapping_adapters', function (Blueprint $table): void {
            $table->id();
            $table->string('mls_code', 64)->index();
            $table->string('dataset_code', 128)->index();
            $table->string('canonical_field_key', 256)->index();
            $table->string('source_field_key', 256)->index();
            $table->json('transform_in_json')->nullable();
            $table->json('transform_out_json')->nullable();
            $table->timestamps();

            $table->unique(['mls_code', 'dataset_code', 'canonical_field_key'], 'field_mapping_adapters_unique');
        });

        Schema::create('lookup_cache_snapshots', function (Blueprint $table): void {
            $table->id();
            $table->string('cache_key', 512)->unique();
            $table->string('mls_code', 64)->index();
            $table->string('dataset_code', 128)->index();
            $table->string('scope', 64)->index();
            $table->json('payload_json');
            $table->string('checksum', 128);
            $table->string('version_tag', 128)->nullable();
            $table->timestamp('expires_at')->index();
            $table->timestamp('refreshed_at')->index();
            $table->timestamps();
        });

        Schema::create('agent_searches', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('name', 255);
            $table->json('search_state_json')->nullable();
            $table->json('mls_scope_json')->nullable();
            $table->boolean('is_template')->default(false);
            $table->string('source', 32)->default('manual')->index();
            $table->timestamps();
        });

        Schema::create('agent_search_filters', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('agent_search_id')->constrained('agent_searches')->cascadeOnDelete();
            $table->string('canonical_field_key', 256)->index();
            $table->string('operator', 64)->index();
            $table->json('value_json');
            $table->json('applies_to_mls_json')->nullable();
            $table->timestamps();
        });

        Schema::create('agent_search_geometries', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('agent_search_id')->constrained('agent_searches')->cascadeOnDelete();
            $table->string('geometry_type', 32)->index();
            $table->string('mode', 16)->index();
            $table->json('geojson');
            $table->json('bbox_json')->nullable();
            $table->decimal('area_m2', 20, 4)->nullable();
            $table->timestamps();
        });

        Schema::create('agent_alert_templates', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('name', 255);
            $table->string('template_type', 64)->index();
            $table->json('body_json')->nullable();
            $table->timestamps();
        });

        Schema::create('agent_alerts', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('agent_search_id')->nullable()->constrained('agent_searches')->nullOnDelete();
            $table->string('name', 255);
            $table->string('alert_type', 64)->index();
            $table->string('status', 32)->default('active')->index();
            $table->json('schedule_json')->nullable();
            $table->timestamp('next_run_at')->nullable()->index();
            $table->timestamps();
        });

        Schema::create('agent_alert_runs', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('agent_alert_id')->constrained('agent_alerts')->cascadeOnDelete();
            $table->string('status', 32)->index();
            $table->json('metadata_json')->nullable();
            $table->timestamp('ran_at')->nullable()->index();
            $table->timestamps();
        });

        Schema::create('agent_automation_settings', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->json('settings_json')->nullable();
            $table->timestamps();

            $table->unique('user_id');
        });

        Schema::create('agent_share_links', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('agent_search_id')->nullable()->constrained('agent_searches')->nullOnDelete();
            $table->string('token', 64)->unique();
            $table->json('attribution_json')->nullable();
            $table->timestamp('expires_at')->nullable()->index();
            $table->timestamps();
        });
    }

    public function down(): void
    {
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
    }
};
