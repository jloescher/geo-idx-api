<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_widget_configs', function (Blueprint $table) {
            $table->id();
            $table->string('ghl_location_id', 64)->index();
            $table->foreignId('ghl_registered_url_id')->nullable()->constrained('ghl_registered_urls')->nullOnDelete();
            $table->string('widget_theme', 32)->default('light');
            $table->string('primary_color', 16)->default('#2563EB');
            $table->string('secondary_color', 16)->default('#1E40AF');
            $table->string('font_family', 128)->default('Inter');
            $table->string('default_widget_type', 32)->default('search');
            $table->unsignedSmallInteger('listings_per_page')->default(20);
            $table->boolean('map_enabled')->default(true);
            $table->boolean('show_lead_form')->default(true);
            $table->string('default_search_area', 128)->default('Tampa Bay');
            $table->json('property_types')->nullable();
            $table->unsignedInteger('min_price')->default(0);
            $table->unsignedInteger('max_price')->nullable();
            $table->unsignedSmallInteger('gate_after_views')->default(3);
            $table->boolean('require_otp')->default(true);
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_widget_configs');
    }
};
