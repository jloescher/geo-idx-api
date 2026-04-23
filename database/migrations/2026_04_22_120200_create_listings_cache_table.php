<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('listings_cache', function (Blueprint $table) {
            $table->string('domain_slug')->primary();
            $table->binary('compressed_data');
            $table->timestamp('last_updated');
            $table->string('etag')->nullable();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('listings_cache');
    }
};
