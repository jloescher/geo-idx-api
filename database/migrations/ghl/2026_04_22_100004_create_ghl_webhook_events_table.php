<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_webhook_events', function (Blueprint $table) {
            $table->id();
            $table->string('webhook_id', 128)->nullable()->unique();
            $table->string('event_type', 128)->index();
            $table->string('ghl_app_id', 64)->nullable();
            $table->string('ghl_company_id', 64)->nullable()->index();
            $table->string('ghl_location_id', 64)->nullable()->index();
            $table->string('ghl_user_id', 64)->nullable();
            $table->json('payload');
            $table->string('processing_status', 32)->default('received')->index();
            $table->timestamp('processed_at')->nullable();
            $table->text('processing_error')->nullable();
            $table->string('handler_class')->nullable();
            $table->timestamp('received_at')->useCurrent();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_webhook_events');
    }
};
