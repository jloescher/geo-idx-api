<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('quantyra_leads', function (Blueprint $table) {
            $table->id();
            $table->string('ghl_location_id', 64)->nullable()->index();
            $table->string('lead_type', 64)->index();
            $table->string('source', 64)->default('widget');
            $table->json('payload');
            $table->string('listing_id', 64)->nullable()->index();
            $table->string('quantyra_domain')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('quantyra_leads');
    }
};
