<?php

namespace App\Ghl\Sync\Services;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Sync\Models\GhlInstalledLocation;
use App\Ghl\Sync\Models\GhlLeadMapping;
use App\Ghl\Sync\Models\GhlSyncLog;
use App\Ghl\Sync\Models\QuantyraLead;

class LeadSyncService
{
    public function __construct(
        private readonly ContactCreator $contacts,
        private readonly OpportunityCreator $opportunities,
        private readonly TagManager $tags,
    ) {}

    public function sync(QuantyraLead $lead): void
    {
        $locationId = $lead->ghl_location_id;
        if (! $locationId) {
            $this->fail($lead, 'missing_location', 'Lead has no ghl_location_id');

            return;
        }

        $token = GhlOAuthToken::query()
            ->where('ghl_location_id', $locationId)
            ->where('status', 'active')
            ->orderByDesc('id')
            ->first();

        if (! $token) {
            $this->fail($lead, 'no_token', 'No active GHL OAuth token for location');

            return;
        }

        $mapping = GhlLeadMapping::query()->where('quantyra_lead_type', $lead->lead_type)->first();
        if (! $mapping) {
            $this->fail($lead, 'no_mapping', 'Unknown lead_type mapping');

            return;
        }

        $syncLog = GhlSyncLog::query()->create([
            'ghl_location_id' => $locationId,
            'quantyra_lead_id' => $lead->id,
            'sync_type' => 'contact',
            'lead_type' => $lead->lead_type,
            'sync_status' => 'pending',
            'request_payload' => $lead->payload,
        ]);

        $payload = $lead->payload ?? [];
        $first = (string) ($payload['first_name'] ?? $payload['firstName'] ?? 'Lead');
        $last = (string) ($payload['last_name'] ?? $payload['lastName'] ?? 'Quantyra');
        $email = (string) ($payload['email'] ?? '');
        $phone = (string) ($payload['phone'] ?? '');

        $tags = $mapping->default_tags ?? ['quantyra-lead'];
        if ($mapping->domain_tag_prefix && $lead->quantyra_domain) {
            $tags[] = $lead->quantyra_domain;
        }

        $contactBody = array_filter([
            'locationId' => $locationId,
            'firstName' => $first,
            'lastName' => $last,
            'email' => $email ?: null,
            'phone' => $phone ?: null,
            'source' => 'Quantyra GeoIDX',
            'tags' => array_values(array_unique($tags)),
        ]);

        $contactResponse = $this->contacts->create($token, $contactBody);

        if ($contactResponse->failed()) {
            $syncLog->update([
                'sync_status' => 'failed',
                'error_message' => $contactResponse->body(),
                'response_payload' => ['status' => $contactResponse->status()],
                'completed_at' => now(),
            ]);

            return;
        }

        $contactJson = $contactResponse->json();
        $contactId = $contactJson['contact']['id'] ?? $contactJson['id'] ?? null;

        $syncLog->ghl_contact_id = $contactId;
        $syncLog->sync_status = 'success';
        $syncLog->response_payload = $contactJson;
        $syncLog->completed_at = now();
        $syncLog->save();

        if ($mapping->creates_opportunity && $contactId) {
            $oppPayload = array_filter([
                'locationId' => $locationId,
                'contactId' => $contactId,
                'name' => trim($first.' '.$last).' — '.$lead->lead_type,
                'pipelineId' => $mapping->opportunity_pipeline,
                'status' => 'open',
            ]);
            if ($mapping->opportunity_stage) {
                $oppPayload['pipelineStageId'] = $mapping->opportunity_stage;
            }
            $opp = $this->opportunities->create($token, $oppPayload);
            if ($opp->successful()) {
                $oj = $opp->json();
                $syncLog->ghl_opportunity_id = $oj['id'] ?? $oj['opportunity']['id'] ?? null;
                $syncLog->sync_type = 'opportunity';
                $syncLog->save();
            }
        }

        GhlInstalledLocation::query()
            ->where('ghl_location_id', $locationId)
            ->increment('lead_count');
    }

    private function fail(QuantyraLead $lead, string $code, string $message): void
    {
        GhlSyncLog::query()->create([
            'ghl_location_id' => $lead->ghl_location_id ?? 'unknown',
            'quantyra_lead_id' => $lead->id,
            'sync_type' => 'contact',
            'lead_type' => $lead->lead_type,
            'sync_status' => 'failed',
            'error_code' => $code,
            'error_message' => $message,
            'completed_at' => now(),
        ]);
    }
}
