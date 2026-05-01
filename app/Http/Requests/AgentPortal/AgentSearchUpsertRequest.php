<?php

namespace App\Http\Requests\AgentPortal;

use Illuminate\Foundation\Http\FormRequest;

class AgentSearchUpsertRequest extends FormRequest
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
            'is_template' => ['nullable', 'boolean'],
            'source' => ['nullable', 'string', 'in:manual,template,widget,alert'],
            'search_state_json' => ['nullable', 'array'],
            'mls_scope_json' => ['nullable', 'array'],
            'mls_scope_json.*.mls_code' => ['required_with:mls_scope_json', 'string', 'max:64'],
            'mls_scope_json.*.dataset_code' => ['required_with:mls_scope_json', 'string', 'max:128'],
            'filters' => ['nullable', 'array'],
            'filters.*.canonical_field_key' => ['required_with:filters', 'string', 'max:256'],
            'filters.*.operator' => ['required_with:filters', 'string', 'max:64'],
            'filters.*.value_json' => ['nullable'],
            'filters.*.applies_to_mls_json' => ['nullable', 'array'],
            'geometries' => ['nullable', 'array'],
            'geometries.*.geometry_type' => ['required_with:geometries', 'string', 'in:polygon,circle'],
            'geometries.*.mode' => ['required_with:geometries', 'string', 'in:include,exclude'],
            'geometries.*.geojson' => ['required_with:geometries', 'array'],
            'geometries.*.bbox_json' => ['nullable', 'array'],
            'geometries.*.area_m2' => ['nullable', 'numeric', 'min:0'],
        ];
    }
}
