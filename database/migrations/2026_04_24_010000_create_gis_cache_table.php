<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('gis_cache', function (Blueprint $table) {
            $table->id();
            $table->string('query_hash', 64)->unique();
            $table->longText('geojson');
            $table->string('county', 48)->nullable();
            $table->timestampTz('expires_at');
            $table->string('source_used', 96);
            $table->timestampsTz();

            $table->index('expires_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('gis_cache');
    }
};
