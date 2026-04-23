<?php

namespace Database\Seeders;

use App\Ghl\Sync\Models\GhlLeadMapping;
use Illuminate\Database\Seeder;

class GhlConfigSeeder extends Seeder
{
    public function run(): void
    {
        $rows = [
            [
                'quantyra_lead_type' => 'showing_request',
                'creates_contact' => true,
                'creates_opportunity' => true,
                'opportunity_pipeline' => null,
                'opportunity_stage' => null,
                'default_tags' => ['quantyra-lead', 'showing-request'],
                'domain_tag_prefix' => true,
                'is_high_value' => true,
            ],
            [
                'quantyra_lead_type' => 'pre_approval',
                'creates_contact' => true,
                'creates_opportunity' => true,
                'opportunity_pipeline' => null,
                'opportunity_stage' => null,
                'default_tags' => ['quantyra-lead', 'pre-approval'],
                'domain_tag_prefix' => true,
                'is_high_value' => true,
            ],
            [
                'quantyra_lead_type' => 'journey_question',
                'creates_contact' => true,
                'creates_opportunity' => false,
                'default_tags' => ['quantyra-lead', 'journey-question'],
                'domain_tag_prefix' => true,
                'is_high_value' => false,
            ],
            [
                'quantyra_lead_type' => 'general_inquiry',
                'creates_contact' => true,
                'creates_opportunity' => false,
                'default_tags' => ['quantyra-lead', 'general-inquiry'],
                'domain_tag_prefix' => true,
                'is_high_value' => false,
            ],
            [
                'quantyra_lead_type' => 'property_view',
                'creates_contact' => true,
                'creates_opportunity' => false,
                'default_tags' => ['quantyra-lead', 'property-view'],
                'domain_tag_prefix' => true,
                'is_high_value' => false,
            ],
        ];

        foreach ($rows as $row) {
            GhlLeadMapping::query()->updateOrCreate(
                ['quantyra_lead_type' => $row['quantyra_lead_type']],
                $row
            );
        }
    }
}
