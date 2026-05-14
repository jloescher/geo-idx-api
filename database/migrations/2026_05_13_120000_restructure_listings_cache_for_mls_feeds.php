<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Revenue impact: row-level cache enables precise eviction of closed listings without nuking whole-domain cache hits.
     *
     * Compliance: PostgreSQL retention for Active/Pending only; closed history is never persisted here (MLS GRID IDX).
     */
    public function up(): void
    {
        Schema::dropIfExists('listings_cache');

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
    }

    public function down(): void
    {
        Schema::dropIfExists('listings_cache');

        Schema::create('listings_cache', function (Blueprint $table) {
            $table->string('domain_slug')->primary();
            $table->binary('compressed_data');
            $table->timestamp('last_updated');
            $table->string('etag')->nullable();
        });
    }
};
