<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;

class AgentAutomationSettingsRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    /**
     * @return array<string, mixed>
     */
    public function rules(): array
    {
        return [
            'nurture_enabled' => ['nullable', 'boolean'],
            'nurture_mode' => ['nullable', 'string', 'in:off,basic,aggressive'],
            'dry_run' => ['nullable', 'boolean'],
            'eligibility_tags' => ['nullable', 'array'],
            'eligibility_tags.*' => ['string', 'max:64'],
            'integration_health' => ['nullable', 'string', 'in:connected,degraded,disconnected'],
            'last_sync_at' => ['nullable', 'date'],
            'notes' => ['nullable', 'string', 'max:1000'],
        ];
    }
}
