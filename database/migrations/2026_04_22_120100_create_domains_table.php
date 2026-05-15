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
            $table->string('domain_slug')->unique()->comment('Hostname registered for Stellar MLS Exhibit A routing');
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
        Schema::dropIfExists('domains');
    }
};
