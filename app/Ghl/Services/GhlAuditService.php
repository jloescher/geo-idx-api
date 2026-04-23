<?php

namespace App\Ghl\Services;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Schema;

class GhlAuditService
{
    public function record(array $row): void
    {
        if (! config('ghl.audit.enabled')) {
            return;
        }

        $data = array_merge([
            'logged_at' => now(),
            'mls_request_count' => $row['mls_request_count'] ?? 1,
            'is_mls_data_access' => $row['is_mls_data_access'] ?? true,
            'compliance_verified' => $row['compliance_verified'] ?? false,
        ], $row);

        if (Schema::hasTable('ghl_audit_logs')) {
            DB::table('ghl_audit_logs')->insert([
                'logged_at' => $data['logged_at'],
                'ghl_company_id' => $data['ghl_company_id'] ?? null,
                'ghl_location_id' => $data['ghl_location_id'] ?? null,
                'ghl_user_id' => $data['ghl_user_id'] ?? null,
                'ghl_oauth_token_id' => $data['ghl_oauth_token_id'] ?? null,
                'listing_id' => $data['listing_id'] ?? null,
                'mls_request_count' => $data['mls_request_count'],
                'api_endpoint' => $data['api_endpoint'],
                'request_method' => $data['request_method'] ?? 'GET',
                'lead_type' => $data['lead_type'] ?? null,
                'sync_status' => $data['sync_status'] ?? null,
                'response_status' => $data['response_status'] ?? null,
                'latency_ms' => $data['latency_ms'] ?? null,
                'error_details' => $data['error_details'] ?? null,
                'is_mls_data_access' => $data['is_mls_data_access'],
                'compliance_verified' => $data['compliance_verified'],
                'created_at' => now(),
                'updated_at' => now(),
            ]);
        }

        $path = config('ghl.audit.file');
        if ($path) {
            Log::build([
                'driver' => 'single',
                'path' => $path,
            ])->info('ghl_audit', $data);
        }
    }

    public function forToken(?GhlOAuthToken $token, string $endpoint, string $method = 'GET', array $extra = []): void
    {
        $this->record(array_merge([
            'ghl_oauth_token_id' => $token?->id,
            'ghl_company_id' => $token?->ghl_company_id,
            'ghl_location_id' => $token?->ghl_location_id,
            'ghl_user_id' => $token?->ghl_user_id,
            'api_endpoint' => $endpoint,
            'request_method' => $method,
        ], $extra));
    }
}
