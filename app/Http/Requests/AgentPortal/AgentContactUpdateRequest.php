<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;

class AgentContactUpdateRequest extends FormRequest
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
            'status' => ['sometimes', 'string', 'max:64'],
            'stage' => ['sometimes', 'string', 'max:64'],
            'tags' => ['sometimes', 'array', 'max:30'],
            'tags.*' => ['string', 'max:64'],
            'notes' => ['sometimes', 'string', 'max:4000'],
        ];
    }
}
