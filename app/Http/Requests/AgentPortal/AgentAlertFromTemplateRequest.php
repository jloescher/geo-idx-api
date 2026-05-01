<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class AgentAlertFromTemplateRequest extends FormRequest
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
            'template_id' => [
                'required',
                'integer',
                Rule::exists('agent_alert_templates', 'id')->where(static function ($query) use ($userId): void {
                    if ($userId === null) {
                        $query->whereRaw('1 = 0');

                        return;
                    }
                    $query->where('user_id', $userId);
                }),
            ],
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
        ];
    }
}
