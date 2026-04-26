<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('domains', function (Blueprint $table): void {
            $table->json('allowed_mls_datasets')->nullable()->after('mls_dataset');
        });

        Schema::create('bridge_search_cache', function (Blueprint $table): void {
            $table->string('partition_key', 255);
            $table->string('fingerprint', 64);
            $table->binary('compressed_data');
            $table->timestamp('last_updated');
            $table->primary(['partition_key', 'fingerprint']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('bridge_search_cache');

        Schema::table('domains', function (Blueprint $table): void {
            $table->dropColumn('allowed_mls_datasets');
        });
    }
};
