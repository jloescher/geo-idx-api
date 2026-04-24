<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('gis_cache', function (Blueprint $table) {
            $table->unsignedBigInteger('source_generation')->default(0)->after('source_used');
        });
    }

    public function down(): void
    {
        Schema::table('gis_cache', function (Blueprint $table) {
            $table->dropColumn('source_generation');
        });
    }
};
