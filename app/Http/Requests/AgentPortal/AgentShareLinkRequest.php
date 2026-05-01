<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;

class AgentShareLinkRequest extends FormRequest
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
            'agent_search_id' => ['nullable', 'integer', 'exists:agent_searches,id'],
            'attribution_json' => ['nullable', 'array'],
            'expires_at' => ['nullable', 'date'],
            'template_kind' => ['nullable', 'string', 'in:standard,seo_landing'],
        ];
    }
}
