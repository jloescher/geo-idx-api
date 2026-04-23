<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_registered_urls', function (Blueprint $table) {
            $table->id();
            $table->foreignId('ghl_oauth_token_id')->constrained('ghl_oauth_tokens')->cascadeOnDelete();
            $table->string('ghl_location_id', 64);
            $table->string('ghl_company_id', 64);
            $table->string('primary_url', 512);
            $table->json('additional_urls')->nullable();
            $table->boolean('urls_validated')->default(false);
            $table->json('validation_errors')->nullable();
            $table->string('widget_api_key', 64)->unique();
            $table->boolean('widget_access_enabled')->default(true);
            $table->string('integration_type', 32);
            $table->boolean('mls_agreement_acknowledged')->default(false);
            $table->boolean('mls_compliance_verified')->default(false);
            $table->boolean('stellar_mls_approved')->default(false);
            $table->timestamps();

            $table->index(['ghl_location_id', 'primary_url']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_registered_urls');
    }
};
