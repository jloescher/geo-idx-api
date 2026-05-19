<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('domains', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->nullable()->constrained()->nullOnDelete();
            $table->string('domain_slug')->unique()->comment('Hostname registered for MLS domain authorization');
            $table->boolean('is_active')->default(true);
            $table->string('mls_dataset')->nullable();
            $table->json('allowed_mls_datasets')->nullable();
            $table->string('verification_status', 32)->default('pending');
            $table->string('verification_method', 32)->nullable();
            $table->string('txt_verification_name')->nullable();
            $table->string('txt_verification_value')->nullable();
            $table->timestamp('txt_verified_at')->nullable();
            $table->timestamp('verification_checked_at')->nullable();
            $table->json('verification_metadata')->nullable();
            $table->timestamps();
        });

        Schema::create('mls_search_cache', function (Blueprint $table) {
            $table->string('partition_key', 255);
            $table->string('fingerprint', 64);
            $table->binary('compressed_data');
            $table->timestamp('last_updated');
            $table->primary(['partition_key', 'fingerprint']);
        });

        Schema::create('listings_cache', function (Blueprint $table) {
            $table->string('domain_slug');
            $table->string('feed_code', 64);
            $table->string('listing_key', 191);
            $table->string('standard_status', 64);
            $table->binary('compressed_payload');
            $table->timestamp('first_cached_at');
            $table->timestamp('last_refreshed_at');
            $table->primary(['domain_slug', 'feed_code', 'listing_key']);
            $table->index(['domain_slug', 'feed_code', 'last_refreshed_at'], 'listings_cache_domain_feed_refreshed_idx');
        });

        Schema::create('mls_proxy_audit_logs', function (Blueprint $table) {
            $table->id();
            $table->timestamp('logged_at')->useCurrent();
            $table->string('domain_slug')->nullable();
            $table->string('token_name')->nullable();
            $table->string('request_type');
            $table->unsignedInteger('listing_count')->nullable();
            $table->string('ip_address', 45)->nullable();
            $table->foreignId('user_id')->nullable()->constrained()->nullOnDelete();
            $table->index('logged_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('mls_proxy_audit_logs');
        Schema::dropIfExists('listings_cache');
        Schema::dropIfExists('mls_search_cache');
        Schema::dropIfExists('domains');
    }
};
