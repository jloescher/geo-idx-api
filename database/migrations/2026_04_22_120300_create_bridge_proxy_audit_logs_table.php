<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('bridge_proxy_audit_logs', function (Blueprint $table) {
            $table->id();
            $table->timestamp('logged_at')->useCurrent();
            $table->string('domain_slug')->nullable();
            $table->string('token_name')->nullable();
            $table->string('request_type');
            $table->unsignedInteger('listing_count')->nullable();
            $table->string('ip_address', 45)->nullable();
            $table->foreignId('user_id')->nullable()->constrained()->nullOnDelete();
            $table->index('logged_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('bridge_proxy_audit_logs');
    }
};
