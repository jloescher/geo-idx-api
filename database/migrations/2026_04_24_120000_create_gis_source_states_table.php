<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('gis_source_states', function (Blueprint $table) {
            $table->string('source_key', 96)->primary();
            $table->string('fingerprint', 128)->nullable();
            $table->unsignedBigInteger('generation')->default(0);
            $table->timestampTz('last_checked_at')->nullable();
            $table->timestampTz('last_changed_at')->nullable();
            $table->timestampsTz();
        });

        foreach ([
            'florida_statewide_cadastral',
            'pinellas_enterprise_parcels',
            'hillsborough_hc_parcels',
        ] as $key) {
            DB::table('gis_source_states')->insert([
                'source_key' => $key,
                'fingerprint' => null,
                'generation' => 0,
                'last_checked_at' => null,
                'last_changed_at' => null,
                'created_at' => now(),
                'updated_at' => now(),
            ]);
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('gis_source_states');
    }
};
