<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('crypto_price_snapshots', function (Blueprint $table) {
            $table->id();
            $table->string('asset_id', 24);
            $table->string('vs_currency', 12);
            $table->decimal('price', 20, 8);
            $table->timestampTz('as_of');
            $table->json('payload')->nullable();
            $table->timestampsTz();

            $table->unique(['asset_id', 'vs_currency']);
            $table->index('as_of');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('crypto_price_snapshots');
    }
};
