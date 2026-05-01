<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;

class AgentContactCreateHandoffAlertRequest extends FormRequest
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
            'contact.name' => ['required', 'string', 'max:255'],
            'contact.email' => ['nullable', 'email', 'max:255'],
            'contact.phone' => ['nullable', 'string', 'max:64'],
            'contact.status' => ['nullable', 'string', 'max:64'],
            'agent_search_id' => ['required', 'integer', 'exists:agent_searches,id'],
            'name' => ['required', 'string', 'max:255'],
            'alert_type' => ['sometimes', 'string', 'max:64'],
            'cadence' => ['sometimes', 'string', 'max:64'],
            'status' => ['sometimes', 'string', 'max:64'],
        ];
    }
}
