<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class AgentAlertUpsertRequest extends FormRequest
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
        $userId = $this->user()?->id;

        return [
            'name' => ['required', 'string', 'max:255'],
            'agent_search_id' => [
                'nullable',
                'integer',
                Rule::exists('agent_searches', 'id')->where(static function ($query) use ($userId): void {
                    if ($userId === null) {
                        $query->whereRaw('1 = 0');

                        return;
                    }
                    $query->where('user_id', $userId);
                }),
            ],
            'alert_type' => ['required', 'string', 'in:listing,market_activity,home_value'],
            'status' => ['nullable', 'string', 'in:active,paused,disabled'],
            'schedule_json' => ['nullable', 'array'],
            'next_run_at' => ['nullable', 'date'],
        ];
    }
}
