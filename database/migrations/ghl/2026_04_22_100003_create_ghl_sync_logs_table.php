<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_sync_logs', function (Blueprint $table) {
            $table->id();
            $table->string('ghl_location_id', 64)->index();
            $table->foreignId('quantyra_lead_id')->nullable()->constrained('quantyra_leads')->nullOnDelete();
            $table->string('ghl_contact_id', 64)->nullable();
            $table->string('ghl_opportunity_id', 64)->nullable();
            $table->string('sync_type', 32);
            $table->string('lead_type', 64)->nullable();
            $table->string('sync_status', 32)->default('pending');
            $table->unsignedSmallInteger('retry_count')->default(0);
            $table->unsignedSmallInteger('max_retries')->default(3);
            $table->text('error_message')->nullable();
            $table->string('error_code', 64)->nullable();
            $table->json('request_payload')->nullable();
            $table->json('response_payload')->nullable();
            $table->timestamps();
            $table->timestamp('completed_at')->nullable();

            $table->index(['ghl_location_id', 'sync_status']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_sync_logs');
    }
};
