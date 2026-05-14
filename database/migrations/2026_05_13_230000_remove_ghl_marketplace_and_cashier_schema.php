<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::dropIfExists('ghl_lead_mappings');
        Schema::dropIfExists('ghl_sync_logs');
        Schema::dropIfExists('ghl_audit_logs');
        Schema::dropIfExists('ghl_webhook_events');
        Schema::dropIfExists('ghl_widget_configs');
        Schema::dropIfExists('ghl_registered_urls');
        Schema::dropIfExists('ghl_installed_locations');
        Schema::dropIfExists('ghl_oauth_tokens');

        Schema::dropIfExists('subscription_items');
        Schema::dropIfExists('subscriptions');

        if (Schema::hasTable('domains')) {
            DB::table('domains')
                ->where('verification_status', 'verified_ghl')
                ->update([
                    'verification_status' => 'verified',
                    'verification_method' => 'txt',
                ]);
        }

        $userColumnsToDrop = [];
        if (Schema::hasTable('users')) {
            foreach (['stripe_id', 'pm_type', 'pm_last_four', 'trial_ends_at'] as $column) {
                if (Schema::hasColumn('users', $column)) {
                    $userColumnsToDrop[] = $column;
                }
            }
            if ($userColumnsToDrop !== []) {
                Schema::table('users', function (Blueprint $table) use ($userColumnsToDrop): void {
                    $table->dropColumn($userColumnsToDrop);
                });
            }
        }

        if (Schema::hasTable('domains') && Schema::hasColumn('domains', 'ghl_verified_at')) {
            Schema::table('domains', function (Blueprint $table): void {
                $table->dropColumn('ghl_verified_at');
            });
        }
    }

    public function down(): void
    {
        // Intentionally empty: GoHighLevel / LeadConnector and Cashier are permanently removed from this codebase.
    }
};
