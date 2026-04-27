<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('domains', function (Blueprint $table): void {
            $table->foreignId('user_id')->nullable()->after('id')->constrained()->nullOnDelete();
            $table->string('verification_status', 32)->default('pending')->after('allowed_mls_datasets');
            $table->string('verification_method', 32)->nullable()->after('verification_status');
            $table->string('txt_verification_name')->nullable()->after('verification_method');
            $table->string('txt_verification_value')->nullable()->after('txt_verification_name');
            $table->timestamp('txt_verified_at')->nullable()->after('txt_verification_value');
            $table->timestamp('ghl_verified_at')->nullable()->after('txt_verified_at');
            $table->timestamp('verification_checked_at')->nullable()->after('ghl_verified_at');
            $table->json('verification_metadata')->nullable()->after('verification_checked_at');
        });
    }

    public function down(): void
    {
        Schema::table('domains', function (Blueprint $table): void {
            $table->dropConstrainedForeignId('user_id');
            $table->dropColumn([
                'verification_status',
                'verification_method',
                'txt_verification_name',
                'txt_verification_value',
                'txt_verified_at',
                'ghl_verified_at',
                'verification_checked_at',
                'verification_metadata',
            ]);
        });
    }
};
