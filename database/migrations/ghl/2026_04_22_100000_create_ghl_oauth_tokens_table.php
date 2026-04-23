<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ghl_oauth_tokens', function (Blueprint $table) {
            $table->id();
            $table->string('ghl_company_id', 64);
            $table->string('ghl_location_id', 64)->nullable();
            $table->string('ghl_user_id', 64)->nullable();
            $table->text('access_token');
            $table->text('refresh_token');
            $table->string('refresh_token_id', 64)->nullable();
            $table->string('user_type', 32);
            $table->timestamp('expires_at');
            $table->timestamp('refresh_expires_at')->nullable();
            $table->text('scopes');
            $table->boolean('is_bulk_install')->default(false);
            $table->timestamp('installed_at')->useCurrent();
            $table->timestamp('last_refreshed_at')->nullable();
            $table->string('status', 32)->default('active');
            $table->timestamp('revoked_at')->nullable();
            $table->string('revoke_reason')->nullable();
            $table->string('access_token_hash', 64)->unique();
            $table->softDeletes();
            $table->timestamps();

            $table->index(['ghl_company_id', 'ghl_location_id']);
            $table->index(['status', 'expires_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ghl_oauth_tokens');
    }
};
