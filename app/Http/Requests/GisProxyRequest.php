<?php

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class GisProxyRequest extends FormRequest
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
            'bbox' => ['nullable', 'string', 'max:120'],
            'north' => ['nullable', 'numeric', 'between:-90,90'],
            'south' => ['nullable', 'numeric', 'between:-90,90'],
            'east' => ['nullable', 'numeric', 'between:-180,180'],
            'west' => ['nullable', 'numeric', 'between:-180,180'],
            'lat' => ['nullable', 'numeric', 'between:-90,90'],
            'latitude' => ['nullable', 'numeric', 'between:-90,90'],
            'lng' => ['nullable', 'numeric', 'between:-180,180'],
            'lon' => ['nullable', 'numeric', 'between:-180,180'],
            'longitude' => ['nullable', 'numeric', 'between:-180,180'],
            'radius' => ['nullable', 'numeric', 'min:50', 'max:500000'],
            'limit' => ['nullable', 'integer', 'min:1', 'max:2000'],
            'layers' => ['nullable', 'string', 'max:120'],
            'domain' => ['nullable', 'string', 'max:255'],
        ];
    }
}
