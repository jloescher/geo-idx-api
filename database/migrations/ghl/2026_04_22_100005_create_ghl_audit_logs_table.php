<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_audit_logs', function (Blueprint $table) {
            $table->id();
            $table->timestamp('logged_at')->useCurrent()->index();
            $table->string('ghl_company_id', 64)->nullable()->index();
            $table->string('ghl_location_id', 64)->nullable()->index();
            $table->string('ghl_user_id', 64)->nullable();
            $table->foreignId('ghl_oauth_token_id')->nullable()->constrained('ghl_oauth_tokens')->nullOnDelete();
            $table->string('listing_id', 64)->nullable();
            $table->unsignedInteger('mls_request_count')->default(1);
            $table->string('api_endpoint', 512);
            $table->string('request_method', 16);
            $table->string('lead_type', 64)->nullable();
            $table->string('sync_status', 64)->nullable();
            $table->unsignedSmallInteger('response_status')->nullable();
            $table->unsignedInteger('latency_ms')->nullable();
            $table->text('error_details')->nullable();
            $table->boolean('is_mls_data_access')->default(true);
            $table->boolean('compliance_verified')->default(false);
            $table->timestamps();

            $table->index(['ghl_location_id', 'logged_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_audit_logs');
    }
};
