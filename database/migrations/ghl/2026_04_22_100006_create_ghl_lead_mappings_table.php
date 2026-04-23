<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_lead_mappings', function (Blueprint $table) {
            $table->id();
            $table->string('quantyra_lead_type', 64)->unique();
            $table->boolean('creates_contact')->default(true);
            $table->boolean('creates_opportunity')->default(false);
            $table->string('opportunity_pipeline')->nullable();
            $table->string('opportunity_stage')->nullable();
            $table->json('default_tags')->nullable();
            $table->boolean('domain_tag_prefix')->default(true);
            $table->json('custom_field_mappings')->nullable();
            $table->boolean('is_high_value')->default(false);
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_lead_mappings');
    }
};
