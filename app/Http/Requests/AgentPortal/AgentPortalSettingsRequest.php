<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;

class AgentPortalSettingsRequest extends FormRequest
{
    public function authorize(): bool
    {
        return $this->user() !== null;
    }

    /**
     * @return array<string, mixed>
     */
    public function rules(): array
    {
        return [
            'notification_email_enabled' => ['nullable', 'boolean'],
            'notification_sms_enabled' => ['nullable', 'boolean'],
            'weekly_digest_enabled' => ['nullable', 'boolean'],
            'alert_default_cadence' => ['nullable', 'string', 'in:daily,weekly,monthly'],
            'timezone' => ['nullable', 'string', 'max:64'],
            'theme_density' => ['nullable', 'string', 'in:compact,comfortable'],
            'onboarding_tips_enabled' => ['nullable', 'boolean'],
            'feature_flags' => ['nullable', 'array'],
            'feature_flags.*' => ['string', 'max:64'],
        ];
    }
}
