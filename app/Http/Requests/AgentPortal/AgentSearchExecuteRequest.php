<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class AgentSearchExecuteRequest extends FormRequest
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
            'filters' => ['required_without:agent_search_id', 'array'],
            'filters.*.field' => ['required_with:filters', 'string', 'max:256'],
            'filters.*.operator' => ['required_with:filters', 'string', 'max:64'],
            'filters.*.value' => ['nullable'],
            'geometries' => ['nullable', 'array'],
            'geometries.*.geometry_type' => ['required_with:geometries', 'string', 'in:polygon,circle'],
            'geometries.*.mode' => ['required_with:geometries', 'string', 'in:include,exclude'],
            'geometries.*.geojson' => ['required_with:geometries', 'array'],
            'telemetry' => ['sometimes', 'array'],
            'telemetry.trigger' => [
                'sometimes',
                'string',
                Rule::in(['manual', 'viewport', 'viewport_pan', 'saved_loaded']),
            ],
        ];
    }
}
