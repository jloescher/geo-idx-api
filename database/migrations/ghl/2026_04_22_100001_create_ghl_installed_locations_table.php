<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_installed_locations', function (Blueprint $table) {
            $table->id();
            $table->foreignId('ghl_oauth_token_id')->nullable()->constrained('ghl_oauth_tokens')->nullOnDelete();
            $table->string('ghl_company_id', 64);
            $table->string('ghl_location_id', 64);
            $table->string('location_name')->nullable();
            $table->text('location_address')->nullable();
            $table->string('location_timezone', 64)->nullable();
            $table->boolean('is_whitelabel')->default(false);
            $table->string('whitelabel_domain')->nullable();
            $table->text('whitelabel_logo_url')->nullable();
            $table->string('subscription_status', 32)->default('none');
            $table->string('subscription_id')->nullable();
            $table->timestamp('subscription_updated_at')->nullable();
            $table->unsignedInteger('mls_request_count')->default(0);
            $table->unsignedInteger('lead_count')->default(0);
            $table->timestamp('last_activity_at')->nullable();
            $table->timestamp('installed_at')->useCurrent();
            $table->timestamp('uninstalled_at')->nullable();
            $table->string('status', 32)->default('active');
            $table->timestamps();

            $table->unique(['ghl_company_id', 'ghl_location_id']);
            $table->index('subscription_status');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_installed_locations');
    }
};
