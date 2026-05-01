<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;

class AgentAlertTemplateUpsertRequest extends FormRequest
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
            'name' => ['required', 'string', 'max:255'],
            'template_type' => ['required', 'string', 'in:listing,market_activity,home_value'],
            'body_json' => ['nullable', 'array'],
            'schedule_json' => ['nullable', 'array'],
            'schedule_json.cadence' => ['nullable', 'string', 'in:immediate,daily,weekly,monthly'],
            'schedule_json.day_of_week' => ['nullable', 'integer', 'min:0', 'max:6'],
            'schedule_json.time_of_day' => ['nullable', 'string'],
        ];
    }
}
